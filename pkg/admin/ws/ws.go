package ws

import (
	"TEFS-BE/pkg/log"
	"fmt"
	"golang.org/x/crypto/ssh"
	"time"

	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"sync"
)

// websocket设置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// websocket连接ssh
type wsSSHConn struct {
	conn          *websocket.Conn
	lock          sync.RWMutex
	rLock         sync.RWMutex
	connIsClose   bool
	sshSession    *ssh.Session
	containerName string
	sshCli        *ssh.Client
}

func newWsSSHConn(conn *websocket.Conn) *wsSSHConn {
	return &wsSSHConn{
		conn:  conn,
		lock:  sync.RWMutex{},
		rLock: sync.RWMutex{},
	}
}

func (c *wsSSHConn) ReadMsg() (int, []byte, error) {
	c.rLock.Lock()
	defer func() {
		c.rLock.Unlock()
	}()
	c.conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(60*30)))
	return c.conn.ReadMessage()
}

// 读取websocket message 作为 ssh输入
func (c *wsSSHConn) Read(p []byte) (n int, err error) {
	_, msg, err := c.ReadMsg()
	if err != nil {
		log.Error(err.Error())
		c.conn.Close()
		c.connIsClose = true
		c.sshSession.Close()
		c.RmDockerContainer()
		return 0, err
	}
	msgStr := string(msg) + "\n"
	r := strings.NewReader(msgStr)
	return r.Read(p)
}

// ssh输出写入到websocket
func (c *wsSSHConn) Write(p []byte) (n int, err error) {
	c.lock.Lock()
	defer func() {
		c.lock.Unlock()
	}()
	wc, err := c.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	defer wc.Close()
	defer c.conn.WriteMessage(1, p)
	return wc.Write(p)
}

func (c *wsSSHConn) RmDockerContainer() error {
	if len(c.containerName) > 0 {
		session, err := c.sshCli.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()
		cmd := fmt.Sprintf("docker stop %s && docker rm %s", c.containerName, c.containerName)
		return session.Run(cmd)
	}
	return nil
}
