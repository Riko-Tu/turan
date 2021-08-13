package console_shell

import (
	"TEFS-BE/pkg/admin/auth"
	"TEFS-BE/pkg/admin/model"
	"TEFS-BE/pkg/log"
	"TEFS-BE/pkg/utils"
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
	"net/http"
	"strconv"
	"strings"
)

var upGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols:[]string{"webtty"},
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

func wsB64Msg(data string) string {
	var b bytes.Buffer
	w := base64.NewEncoder(base64.URLEncoding, &b)
	w.Write([]byte(data))
	w.Close()
	return "1" + b.String()
}

func ConsoleShell(w http.ResponseWriter, r *http.Request) () {
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer conn.Close()

	params := r.URL.Query().Get("params")
	privatePath := viper.GetString("privatePath")
	paramsMap, err := utils.GetEncryptParams(params, privatePath)
	if err != nil {
		log.Error(err.Error())
		conn.WriteMessage(1, []byte(wsB64Msg("params err")))
		return
	}

	projectIdStr, ok := paramsMap["project_id"]
	if !ok {
		conn.WriteMessage(1, []byte(wsB64Msg("project_id not found")))
		return
	}
	token, ok := paramsMap["token"]
	if !ok {
		conn.WriteMessage(1, []byte(wsB64Msg("token not found")))
		return
	}

	projectId, err := strconv.ParseInt(projectIdStr, 10, 64)
	if err != nil {
		conn.WriteMessage(1, []byte(wsB64Msg("project_id err")))
		return
	}
	user, _, _, err := auth.ParseJwtToken(token)
	if err != nil {
		fmt.Println(err.Error())
		conn.WriteMessage(1, []byte(wsB64Msg("token err")))
		return
	}
	shell := &model.ConsoleShell{}
	if err := shell.Get(user.Id, projectId); err != nil {
		fmt.Println(err.Error())
		conn.WriteMessage(1, []byte(wsB64Msg("query record failed")))
		return
	}
	if shell.Id <= 0 {
		conn.WriteMessage(1, []byte(wsB64Msg("shell tefs_file_server not start")))
		return
	}

	address := shell.Address
	tmp := strings.Split(address, ":")
	if len(tmp) < 3 {
		conn.WriteMessage(1, []byte(wsB64Msg("shell record address err")))
		return
	}
	wsAddress := fmt.Sprintf("%s:%s", tmp[0], tmp[1])
	wsUrl := fmt.Sprintf("ws://%s/ws", wsAddress)
	origin := fmt.Sprintf("http://%s", wsAddress)
	headers := make(map[string]string)
	headers["Sec-WebSocket-Protocol"] = "webtty"

	wsCli := NewWsCli(wsUrl, origin, headers)
	defer func() {
		wsCli.Close()
	}()

	if err := wsCli.Conn(); err != nil {
		log.Error(err.Error())
		conn.WriteMessage(1, []byte(wsB64Msg("conn shell tefs_file_server failed")))
		return
	}

	go func() {
		for {
			cliMsg, err := wsCli.ReadMsg()
			if err != nil {
				log.Error(err.Error())
				return
			}
			if err := conn.WriteMessage(1, cliMsg); err != nil {
				log.Error(err.Error())
				return
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Error(err.Error())
			return
		}
		if err = wsCli.Write(msg); err != nil {
			log.Error(err.Error())
			return
		}
	}
}
