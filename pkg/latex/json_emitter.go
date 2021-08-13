package latex

import (
	"TEFS-BE/pkg/log"
	"encoding/json"
	"github.com/Jeffail/leaps/lib/api"
	"github.com/Jeffail/leaps/lib/api/events"
)

// ReadWriteJSONCloser - An interface for writing and reading JSON content.
type ReadWriteJSONCloser interface {
	ReadJSON(v interface{}) error
	WriteJSON(v interface{}) error
	Close() error
}


// Handlers for request types are given the unparsed JSON body of the request.
type JSONEmitter struct {
	requestHandlers  map[string][]api.RequestHandler
	responseHandlers map[string][]api.ResponseHandler
	closeHandlers    []api.EventHandler

	rw ReadWriteJSONCloser
}

// NewJSONEmitter - Constructs a new JSONEmitter around a ReadWriteJSONCloser.
func NewJSONEmitter(rw ReadWriteJSONCloser) *JSONEmitter {
	return &JSONEmitter{
		requestHandlers:  make(map[string][]api.RequestHandler),
		responseHandlers: make(map[string][]api.ResponseHandler),
		rw:               rw,
	}
}

// OnReceive - Register a handler for a particular incoming event type.
func (w *JSONEmitter) OnReceive(reqType string, reqHandler api.RequestHandler) {
	w.requestHandlers[reqType] = append(w.requestHandlers[reqType], reqHandler)
}

// OnSend - Register a handler for a particular outgoing event type.
func (w *JSONEmitter) OnSend(resType string, resHandler api.ResponseHandler) {
	w.responseHandlers[resType] = append(w.responseHandlers[resType], resHandler)
}

// OnClose - Register an event handler for a close event.
func (w *JSONEmitter) OnClose(eventHandler api.EventHandler) {
	w.closeHandlers = append(w.closeHandlers, eventHandler)
}

// Close - Close the underlying network connection.
func (w *JSONEmitter) Close() error {
	return w.rw.Close()
}

// Send - Send data to the connected client.
func (w *JSONEmitter) Send(resType string, data interface{}) error {
	if handlers, ok := w.responseHandlers[resType]; ok {
		for _, h := range handlers {
			if !h(data) {
				return nil
			}
		}
	}
	return w.rw.WriteJSON(struct {
		Type string      `json:"type"`
		Body interface{} `json:"body"`
	}{
		Type: resType,
		Body: data,
	})
}

// ListenAndEmit - Begins reading to the underlying io.ReadWriteJSONCloser and
// emitting events accordingly.
func (w *JSONEmitter) ListenAndEmit(power int64) {
	defer func() {
		for _, h := range w.closeHandlers {
			h()
		}
	}()
	for {
		var req struct {
			Type string          `json:"type"`
			Body json.RawMessage `json:"body"`
		}
		if err := w.rw.ReadJSON(&req); err != nil {
			log.Error(err.Error())
			return
		}
		if power != 1 && req.Type == "transform" {
			continue
		}

		if handlers, ok := w.requestHandlers[req.Type]; ok {
			for _, h := range handlers {
				if err := h(req.Body); err != nil {
					w.Send("error", events.ErrorMessage{
						Error: events.APIError{T: err.Type(), Err: err.Error()},
					})
				}
			}
		} else {
			w.Send("error", events.ErrorMessage{
				Error: events.APIError{T: "command_err", Err: "command not recognised"},
			})
		}
	}
}

