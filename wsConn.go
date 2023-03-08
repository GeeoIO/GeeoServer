package main

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WsConn models a ws connection
// It's optimized to either send a JSON array of objects
// or send an immediate JSON object
type wsConn struct {
	sync.Mutex
	conn   *websocket.Conn
	buffer []JSONChangeMessage

	Name    string
	closing bool
}

// NewWSConn creates a new wsConn handler
func newWSConn(conn *websocket.Conn) *wsConn {
	ws := &wsConn{sync.Mutex{}, conn, nil, "error: uninitialized", false}
	return ws
}

// writeJSON sends a new object in the array, to be sent later
func (ws *wsConn) writeJSON(msg JSONChangeMessage) {
	if ws.closing {
		return
	}

	ws.Lock()
	defer ws.Unlock()

	// if nothing was buffered yet, schedule a flush for later
	if ws.buffer == nil {
		time.AfterFunc(MessageSendInterval, func() {
			ws.Flush()
		})
	}

	// add to buffer
	ws.buffer = append(ws.buffer, msg)
}

// writeImmediateJSON sends a new object immediately (no buffering)
func (ws *wsConn) writeImmediateJSON(msg interface{}) {
	// log.Debug("WS enter writeImmediateJSON")
	// defer log.Debug("WS done writeImmediateJSON")
	if ws.closing {
		return
	}
	ws.Lock()
	defer ws.Unlock()

	err := ws.conn.WriteJSON(msg)
	if err != nil {
		log.Errorf("%s: error %s", ws.Name, err.Error())
	}
}

func (ws *wsConn) close() {
	log.Debug("WS closing ", ws.Name)
	ws.closing = true
	ws.buffer = nil
	ws.conn = nil
}

func (ws *wsConn) Flush() {
	log.Debug("WS flush ", ws.Name, ", len=", len(ws.buffer))
	ws.Lock()
	if ws.buffer != nil {
		ws.Unlock()
		ws.writeImmediateJSON(ws.buffer)
		ws.buffer = nil
	} else {
		ws.Unlock()
	}
}
