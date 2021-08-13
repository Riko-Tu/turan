package service

import (
	"TEFS-BE/pkg/admin/model"
	pb "TEFS-BE/pkg/admin/proto"
	"TEFS-BE/pkg/laboratory/client"
	"TEFS-BE/pkg/log"
	"context"
	"fmt"
	"github.com/go-basic/uuid"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const labPort int64 = 32500

// create console shell tefs_file_server GRPC API handle
func (s Service) CreateShellServer(ctx context.Context, in *pb.CreateShellServerRequest) (*pb.CreateShellServerReply, error) {
	user := ctx.Value("user").(*model.User)
	projectId := in.GetProjectId()
	if projectId <= 0 {
		return nil, InvalidId.Error()
	}

	ip, err := model.GetInstanceIp(user.Id, projectId)
	if err != nil {
		log.Error(err.Error())
		return nil, QueryDbFailed.Error()
	}

	address := fmt.Sprintf("%s:%d", ip, labPort)
	gpcCli, err := client.GetClient(address)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	secret := strings.Replace(uuid.New(), "-", "", -1)
	tefsUrl := viper.GetString("tefsUrl")
	webConsolePort, err := client.CreateWebConsole(gpcCli, user.Id, projectId, secret, tefsUrl)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	if webConsolePort != "" {
		nowTime := time.Now().Unix()
		consoleShell := &model.ConsoleShell{
			ProjectId: projectId,
			UserId:    user.Id,
			Address:   fmt.Sprintf("%s:%s", ip, webConsolePort),
			Secret:    secret,
			CreateAt:  nowTime,
			UpdateAt:  nowTime,
		}
		if err := consoleShell.Create(); err != nil {
			log.Error(err.Error())
			log.Error(fmt.Sprintf("start console shell (%s) success, save db failed。user_id=%d, project=%d, port=%s", ip, user.Id, projectId, webConsolePort))
			return nil, UpdateDbFailed.Error()
		}
	}
	return &pb.CreateShellServerReply{Data: "ok"}, nil
}
