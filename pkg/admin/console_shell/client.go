package console_shell

import (
	"fmt"
	ws "github.com/gorilla/websocket"
	"sync"
)

type wsCli struct {
	url     string
	origin  string
	headers map[string]string
	conn    *ws.Conn
	lock     sync.RWMutex
	rlock     sync.RWMutex
}

func NewWsCli(url, origin string, headers map[string]string) *wsCli {
	return &wsCli{
		url:     url,
		origin:  origin,
		headers: headers,
		lock:  sync.RWMutex{},
		rlock: sync.RWMutex{},
	}
}

func (w *wsCli) Conn() (err error) {
	if w.conn != nil {
		return
	}
	if w.url == "" {
		return fmt.Errorf("url is empty")
	}

	dialer := ws.DefaultDialer
	dialer.ReadBufferSize = 1024
	dialer.WriteBufferSize = 1024
	dialer.Subprotocols = []string{"webtty"}

	w.conn,_, err = dialer.Dial(w.url, nil)
	if err != nil {
		return
	}
	return
}

func (w *wsCli) ReadMsg() (bytes []byte, err error) {
	w.rlock.Lock()
	defer func() {
		w.rlock.Unlock()
	}()
	if err = w.Conn(); err != nil {
		return
	}

	_,msg,err := w.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return msg,nil
}

func (w *wsCli) Write(msg []byte) error {
	w.lock.Lock()
	defer func() {
		w.lock.Unlock()
	}()
	return w.conn.WriteMessage(ws.TextMessage, msg)
}

func (w *wsCli) Close() (err error) {
	if w.conn == nil {
		return
	}
	return w.conn.Close()
}
