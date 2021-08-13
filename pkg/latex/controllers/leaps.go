package controllers

import (
	admin "TEFS-BE/pkg/admin/service"
	latexServer "TEFS-BE/pkg/latex"
	"TEFS-BE/pkg/latex/localleaps"
	"TEFS-BE/pkg/latex/models"
	"TEFS-BE/pkg/latex/utils"
	"TEFS-BE/pkg/latex/localcommand"
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/Jeffail/leaps/lib/acl"
	"github.com/Jeffail/leaps/lib/api"
	apiio "github.com/Jeffail/leaps/lib/api/io"
	"github.com/Jeffail/leaps/lib/audit"
	"github.com/Jeffail/leaps/lib/curator"
	"github.com/Jeffail/leaps/lib/store"
	"github.com/Jeffail/leaps/lib/util"
	leapsLog "github.com/Jeffail/leaps/lib/util/service/log"
	"github.com/Jeffail/leaps/lib/util/service/metrics"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type shellRunner struct{}

func (s shellRunner) CMDRun(cmdStr string) (stdout, stderr []byte, err error) {
	var errRead, outRead io.ReadCloser
	cmd := exec.Command("sh", "-c", cmdStr)
	if outRead, err = cmd.StdoutPipe(); err != nil {
		return
	}
	if errRead, err = cmd.StderrPipe(); err != nil {
		return
	}
	if err = cmd.Start(); err != nil {
		return
	}
	if stdout, err = ioutil.ReadAll(outRead); err != nil {
		return
	}
	if stderr, err = ioutil.ReadAll(errRead); err != nil {
		return
	}
	if err = cmd.Wait(); err != nil {
		return
	}
	return
}

type LeapsFileExists struct {
	acl.FileExists
}

func (f *LeapsFileExists) Authenticate(_ interface{}, _, documentID string) acl.AccessLevel {
	return acl.EditAccess
}

type Broker struct {
	Curator      *curator.Impl
	GlobalBroker *api.GlobalMetadataBroker
	CmdBroker    *api.CMDBroker
	Logger       *leapsLog.Modular
	Stats        *metrics.Type
}

var BrokerMap sync.Map

func GetBroker(targetPath string) (broker *Broker, err error) {
	value, ok := BrokerMap.Load(targetPath)
	if ok {
		broker = value.(*Broker)
		return
	}

	logConf := leapsLog.NewLoggerConfig()
	logConf.Prefix = "leaps"
	logConf.LogLevel = "INFO"
	logger := leapsLog.NewLogger(os.Stdout, logConf)
	statConf := metrics.NewConfig()
	statConf.Type = "http"
	statConf.HTTP.Prefix = "leaps"
	stats, err := metrics.NewHTTP(statConf)
	if err != nil {
		return
	}
	defer stats.Close()

	docStore, err := store.NewFile(targetPath, true)
	if err != nil {
		log.Error(err.Error())
		return
	}

	leapsCOTPath := filepath.Join(targetPath, ".leaps_cot.json")

	// Authenticator
	storeConf := acl.NewFileExistsConfig()
	storeConf.Path = targetPath
	storeConf.ShowHidden = true // 是否显示所有文件
	storeConf.ReservedIgnores = append(storeConf.ReservedIgnores, leapsCOTPath)

	authenticator := LeapsFileExists{}
	authenticator.FileExists = *acl.NewFileExists(storeConf, logger)

	auditors := audit.NewToJSON()
	curatorConf := curator.NewConfig()
	curator1, err := curator.New(curatorConf, logger, stats, &authenticator, docStore, auditors)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer curator1.Close()

	cmds := []string{}
	globalBroker := api.NewGlobalMetadataBroker(time.Second*300, logger, stats)
	cmdBroker := api.NewCMDBroker(cmds, shellRunner{}, time.Second*300, logger, stats)

	broker = &Broker{
		Curator:      curator1,
		GlobalBroker: globalBroker,
		CmdBroker:    cmdBroker,
		Logger:       &logger,
		Stats:        &stats,
	}
	//BrokerMap.Store(targetPath, broker)
	// 存在该key则load，否则store，源码已实现双重校验
	value, ok = BrokerMap.LoadOrStore(targetPath, broker)
	if ok {
		broker = value.(*Broker)
		return
	}
	return
}

func (c Controller) Leaps(ctx *gin.Context) {
	// url get token
	params := ctx.Query("params")
	privatePath := viper.GetString("latex.privatePath")
	params, err := utils.RsaDecrypt(params, privatePath)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrRsaDecrypt, "")
		return
	}
	paramsSplit := strings.Split(params, "=")
	if len(paramsSplit) < 2 {
		fail(ctx, ErrNotFoundToken, "")
		return
	}
	token := paramsSplit[1]
	user, newToken, err := admin.HandleToken(token)
	if err != nil || user.Id <= 0 {
		fail(ctx, ErrToken, "")
		return
	}

	latexId, err := conversionId(ctx.Param("id"))
	if err != nil {
		fail(ctx, ErrId, newToken)
		return
	}
	latex := models.Latex{}
	if err = latex.Get(latexId); err != nil || latex.Id <= 0 {
		fail(ctx, ErrNotFoundLatexRecord, newToken)
		return
	}

	userLatex := &models.UserLatex{}
	if err := userLatex.GetForUserAndLatex(user.Id, latex.Id); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrNotFoundTagRecord, newToken)
		return
	}

	// latex git
	latexBasePath := fmt.Sprintf(latexServer.LatexDirFormat, latexServer.LatexBaseDir, latex.UserId, latex.Id)
	latexGit := localcommand.NewGitCli(latexBasePath)
	if !latexGit.GitIsInit() {
		if err := latexGit.Init(); err != nil {
			log.Error(err.Error())
			fail(ctx, ErrLatexGitInit, newToken)
			return
		}
	}

	ups := make(map[string]interface{})
	if err := userLatex.Update(ups); err != nil {
		log.Error(err.Error())
		fail(ctx, ErrUpdateDB, newToken)
		return
	}

	conn, err := upGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Error(err.Error())
		fail(ctx, ErrCreateWebsocket, newToken)
		return
	}

	latexPath := filepath.Join(latexBasePath, LatexRaw)
	targetPath := latexPath

	broker, err := GetBroker(targetPath)
	if err != nil {
		fail(ctx, ErrSort, newToken)
		return
	}
	globalBroker := broker.GlobalBroker
	cmdBroker := broker.CmdBroker

	//// 发送异步任务
	//taskArg := mchTasks.Arg{
	//	Name:  "experimentId",
	//	Type:  "int64",
	//	Value: latexId,
	//}
	//// 延时5分钟执行
	//eta := time.Now().UTC().Add(time.Second * task.ETA)
	//_, err = task.SendLatexHistoryHandleJob(context.Background(), 0, task.LatexHistoryHandleJobKey, latexId, &eta, taskArg)
	//if err != nil {
	//	log.Error(err.Error())
	//}

	userName := user.Name
	if len(userName) == 0 {
		userName = fmt.Sprintf("user%d", user.Id)
	}
	uuid := util.GenerateUUID()
	jsonEmitter := localleaps.NewJSONEmitter(&apiio.ConcurrentJSON{C: conn})
	globalBroker.NewEmitter(userName, uuid, jsonEmitter)
	cmdBroker.NewEmitter(userName, uuid, jsonEmitter)
	localleaps.NewCuratorSession(userName, uuid, jsonEmitter, broker.Curator, time.Second*300, *broker.Logger, *broker.Stats)

	jsonEmitter.ListenAndEmit(userLatex.Power, userLatex.LatexId, user.Id)
}
