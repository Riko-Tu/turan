package ws

import (
	"TEFS-BE/pkg/admin/auth"
	"TEFS-BE/pkg/admin/model"
	admin "TEFS-BE/pkg/admin/service"
	labCli "TEFS-BE/pkg/laboratory/client"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/utils"
	"TEFS-BE/pkg/utils/ssh"
	"bytes"
	"encoding/json"
	"fmt"
	xj "github.com/basgys/goxml2json"
	gossh "golang.org/x/crypto/ssh"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var experimentService = model.ExperimentService{}
var randomStrRe = regexp.MustCompile(`^[a-zA-z0-9]{6}$`)

type VaspkitServer struct {
	ip       string
	port     int
	username string
	password string
}

func (v *VaspkitServer) Setup(ip, username, password string, port int) () {
	v.ip = ip
	v.port = port
	v.username = username
	v.password = password
}

func (v *VaspkitServer) newSshCli() *ssh.Cli {
	return ssh.New(v.ip, v.username, v.password, v.port)
}

func GetUserAndExperiment(r *http.Request) (user *model.User, experiment *model.Experiment, randomStr string, err error) {
	query := r.URL.Query()
	header := query.Get("authorization")
	randomStr = query.Get("randomStr")
	experimentIdStr := query.Get("experiment_id")

	if !randomStrRe.MatchString(randomStr) {
		err = admin.InvalidParams.ErrorParam("randomStr", randomStr)
		return
	}
	user, _, err = admin.HandleToken(header)
	if err != nil {
		return
	}

	experimentId, err := strconv.ParseInt(experimentIdStr, 10, 64)
	if err != nil {
		err = admin.InvalidParams.ErrorParam("experiment_id", experimentIdStr)
		return
	}
	experiment, err = experimentService.Get(experimentId)
	if err != nil {
		log.Error(err.Error())
		err = admin.InvalidId.Error()
		return
	}
	if experiment.UserId != user.Id {
		err = admin.NotFoundRecord.ErrorRecord("experiment")
		return
	}
	return
}

func (v VaspkitServer) WsVaspkit(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	user, experiment, randomStr, err := GetUserAndExperiment(r)
	if err != nil {
		conn.WriteMessage(1, []byte(err.Error()))
		return
	}
	labAddress := experiment.LaboratoryAddress
	labGrpcCli, err := labCli.GetClient(labAddress)
	tmpSecret, cosBaseUrl, err := labCli.GetCosTmpSecret(labGrpcCli, "*")
	if err != nil {
		log.Error(err.Error())
		conn.WriteMessage(1,
			[]byte(admin.GetCosTmpSercretFailed.Error().Error()))
		return
	}

	strList := strings.Split(*cosBaseUrl, "//")
	strList = strings.Split(strList[1], ".")
	bucket := strList[0]
	ourl := "http://" + strings.Join(strList[1:], ".")

	cli := v.newSshCli()
	sshCli, err := cli.GetClient()
	if err != nil {
		log.Error(err.Error())
		conn.WriteMessage(1,
			[]byte(admin.ConnErr.Error().Error()))
		return
	}

	ws := newWsSSHConn(conn)

	// docker 容器名
	container := fmt.Sprintf("u%de%d-%s", user.Id, experiment.Id, randomStr)
	ws.containerName = container
	ws.sshCli = sshCli
	ws.RmDockerContainer()

	baseDir := fmt.Sprintf("/root/dev_tefs/cos/%s", container)

	os.Mkdir(fmt.Sprintf("/root/cos/%s", container), 0666)
	cmd := fmt.Sprintf(`docker run -dit --cap-add SYS_ADMIN --device /dev/fuse --privileged -v %s:/mnt/cosfs --name %s vaspkit bash -c 'echo "COSAccessKeyId=%s" > /tmp/passwd-sts && echo "COSSecretKey=%s" >> /tmp/passwd-sts && echo "COSAccessToken=%s" >> /tmp/passwd-sts && echo "COSAccessTokenExpire=%s" >> /tmp/passwd-sts && cosfs %s:/users/%d/experiments/%d /mnt/cosfs -ourl=%s -odbglevel=info -oallow_other -ocam_role=sts -opasswd_file=/tmp/passwd-sts && python'`, baseDir, container, tmpSecret.TmpSecretID, tmpSecret.TmpSecretKey, tmpSecret.SessionToken, tmpSecret.Expiration, bucket, user.Id, experiment.Id, ourl)
	if err := ssh.RemoteCmd(sshCli, cmd); err != nil {
		log.Error(err.Error())
		conn.WriteMessage(1,
			[]byte(admin.ConnErr.Error().Error()))
		return
	}

	conn.SetCloseHandler(func(code int, text string) error {
		ws.connIsClose = true
		ws.sshSession.Close()
		ws.RmDockerContainer()
		return nil
	})
	conn.SetPingHandler(nil)

	vaspkitCmd := fmt.Sprintf(`docker exec -it %s bash -c 'cd /mnt/cosfs && vaspkit'`, container)

	var opCount int = 0
	for !ws.connIsClose {
		_, msgByte, err := ws.ReadMsg()
		if err != nil {
			log.Error(err.Error())
			return
		}
		opType := string(msgByte)

		sshSession, err := sshCli.NewSession()
		if err != nil {
			log.Error(err.Error())
			conn.WriteMessage(1,
				[]byte(admin.ConnErr.Error().Error()))
		}
		ws.sshSession = sshSession

		if opCount == 0 {
			opCount += 1
			continue
		}

		switch opType {
		case "vaspkit":
			if err := cli.RunTerminal(vaspkitCmd, ws, sshSession); err != nil {
				log.Error(err.Error())
				conn.WriteMessage(1,
					[]byte(admin.ConnErr.Error().Error()))
			}
		default:
			conn.WriteMessage(1, []byte(fmt.Sprintf("unknown type:%s", opType)))
		}
		conn.WriteMessage(1, []byte(fmt.Sprintf("%s end", opType)))
		opCount += 1
	}
}

func (v VaspkitServer) WsFileTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	user, experiment, randomStr, err := GetUserAndExperiment(r)
	if err != nil {
		auth.WriteResponse("", err.Error(), 3001, w)
		return
	}

	dockerContainer := fmt.Sprintf("u%de%d-%s", user.Id, experiment.Id, randomStr)
	cmd := fmt.Sprintf(`docker exec %s bash -c "cd /mnt/cosfs && tree -X"`, dockerContainer)

	cli := v.newSshCli()
	ret, err := cli.Run(cmd)
	if err != nil {
		auth.WriteResponse("", "exec cmd failed", 3002, w)
		log.Error(err.Error())
		return
	}
	xml := strings.NewReader(string(ret))
	json, err := xj.Convert(xml)
	if err != nil {
		auth.WriteResponse("", string(ret), 3003, w)
	} else {
		auth.WriteResponse(json.String(), "", 0, w)
	}
	return

	//conn, err := upgrader.Upgrade(w, r, nil)
	//if err != nil {
	//	panic(err)
	//}
	//defer conn.Close()
	//
	//user, experiment, randomStr, err := GetUserAndExperiment(r)
	//if err != nil {
	//	conn.WriteMessage(1, []byte(err.Error()))
	//	return
	//}
	//
	//dockerContainer := fmt.Sprintf("u%de%d-%s", user.Id, experiment.Id, randomStr)
	//cmd := fmt.Sprintf(`docker exec %s bash -c "cd /mnt/cosfs && tree -X"`, dockerContainer)
	//
	//conn.SetCloseHandler(nil)
	//conn.SetPingHandler(nil)
	//
	//cli := v.newSshCli()
	//
	//for {
	//	ret, err := cli.Run(cmd)
	//	if err != nil {
	//		log.Error(err.Error())
	//		return
	//	}
	//	xml := strings.NewReader(string(ret))
	//	json, err := xj.Convert(xml)
	//	if err != nil {
	//		conn.WriteMessage(1, ret)
	//	} else {
	//		conn.WriteMessage(1, []byte(json.String()))
	//	}
	//	time.Sleep(time.Second)
	//	conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(60*30)))
	//	_, _, err = conn.ReadMessage()
	//	if err != nil {
	//		return
	//	}
	//}
}

type httpSshConn struct {
	w       http.ResponseWriter
	session *gossh.Session
	lineNum int
}
func (h httpSshConn) Write(p []byte) (n int, err error) {
	h.w.Header().Set("Content-Type", "text/html")
	i := bytes.Index(p, []byte("\n"))
	j := bytes.LastIndex(p, []byte("\n"))
	if i > 0 && j > i {
		if h.lineNum == 0 {
			h.w.Write(p[: j])
		} else {
			h.w.Write(p[i+1: j])
		}
	} else {
		h.w.Write(p)
	}
	h.session.Close()
	return
}
func (h httpSshConn) Read(p []byte) (n int, err error) {
	return
}

func (v VaspkitServer) WsFileContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	user, experiment, randomStr, err := GetUserAndExperiment(r)
	if err != nil {
		auth.WriteResponse("", err.Error(), 3001, w)
		return
	}
	query := r.URL.Query()
	filePath := query.Get("filePath")
	startLineStr := query.Get("startLine")
	if len(startLineStr) == 0 {
		startLineStr = "0"
	}

	lineNumStr := query.Get("lineNum")
	countStr := query.Get("count")
	if countStr == "" {
		countStr = "40"
	}
	startLine, err := strconv.Atoi(lineNumStr)
	readLine, err := strconv.Atoi(countStr)
	if err != nil {
		auth.WriteResponse("", "lineNum is int", 3004, w)
		return
	}

	container := fmt.Sprintf("u%de%d-%s", user.Id, experiment.Id, randomStr)
	cmd := fmt.Sprintf(`docker exec %s bash -c "cd /mnt/cosfs && tree -X"`, container)
	cli := v.newSshCli()
	xml, err := cli.Run(cmd)
	if err != nil {
		log.Error(err.Error())
		auth.WriteResponse("", err.Error(), 3004, w)
		return
	}

	isFind, err := utils.FindFile(string(xml), filePath)
	if err != nil || !isFind {
		auth.WriteResponse("", fmt.Sprintf("not found file:%s", filePath), 3004, w)
		return
	}

	cmd = fmt.Sprintf(`docker exec %s bash -c "cd /mnt/cosfs && gmore %s %d %d"`, container, filePath, startLine, readLine)
	ret, err  := cli.Run(cmd)
	if err != nil {
		auth.WriteResponse("", fmt.Sprintf("open file error:%s", err.Error()), 3004, w)
		return
	}
	w.Write(ret)


	//conn, err := upgrader.Upgrade(w, r, nil)
	//if err != nil {
	//	panic(err)
	//}
	//defer conn.Close()
	//
	//user, experiment, randomStr, err := GetUserAndExperiment(r)
	//if err != nil {
	//	conn.WriteMessage(1, []byte(err.Error()))
	//	return
	//}
	//
	//query := r.URL.Query()
	//filePath := query.Get("filePath")
	//
	//cli := v.newSshCli()
	//
	//container := fmt.Sprintf("u%de%d-%s", user.Id, experiment.Id, randomStr)
	//cmd := fmt.Sprintf(`docker exec %s bash -c "cd /mnt/cosfs && tree -X"`, container)
	//xml, err := cli.Run(cmd)
	//if err != nil {
	//	log.Error(err.Error())
	//	conn.WriteMessage(1,
	//		[]byte(admin.ConnErr.Error().Error()))
	//	return
	//}
	//
	//isFind, err := utils.FindFile(string(xml), filePath)
	//if err != nil || !isFind {
	//	conn.WriteMessage(1, []byte(fmt.Sprintf("not found file:%s", filePath)))
	//	return
	//}
	//cmd = fmt.Sprintf(`docker exec -it %s bash -c "cd /mnt/cosfs && less %s"`, container, filePath)
	//
	//conn.SetCloseHandler(nil)
	//conn.SetPingHandler(nil)
	//
	//ws := newWsSSHConn(conn)
	//sshCli, err := cli.GetClient()
	//if err != nil {
	//	log.Error(err.Error())
	//	conn.WriteMessage(1,
	//		[]byte(admin.ConnErr.Error().Error()))
	//	return
	//}
	//sshSession, err := sshCli.NewSession()
	//if err != nil {
	//	log.Error(err.Error())
	//	conn.WriteMessage(1,
	//		[]byte(admin.ConnErr.Error().Error()))
	//}
	//ws.sshSession = sshSession
	//
	//err = cli.RunTerminal(cmd, ws, sshSession)
	//if err != nil {
	//	log.Error(err.Error())
	//	return
	//}
}

func (v VaspkitServer) GetCosTmpSecret(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	user, experiment, _, err := GetUserAndExperiment(r)
	if err != nil {
		auth.WriteResponse("", err.Error(), 3001, w)
		return
	}
	labAddress := experiment.LaboratoryAddress
	labGrpcCli, err := labCli.GetClient(labAddress)
	tmpSecret, cosBaseUrl, err := labCli.GetCosDownloadTmpSecret(labGrpcCli, user.Id)
	if err != nil {
		log.Error(err.Error())
		auth.WriteResponse("", "get cos tmp secret failed", 3004, w)
		return
	}
	data := make(map[string]interface{})
	data["cosTmpSecret"] = tmpSecret
	data["cosBaseUrl"] = cosBaseUrl
	data["baseDir"] = fmt.Sprintf("/users/%d/experiments/%d/", user.Id, experiment.Id)

	dataJson, err := json.Marshal(data)
	if err != nil {
		log.Error(err.Error())
		auth.WriteResponse("", "cos tmp secret to json failed", 3004, w)
		return
	}
	w.Write(dataJson)
}