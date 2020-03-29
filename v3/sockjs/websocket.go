package sockjs

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func (h *Handler) sockjsWebsocket(rw http.ResponseWriter, req *http.Request) {
	upgrader := h.options.WebsocketUpgrader
	if upgrader == nil {
		upgrader = new(websocket.Upgrader)
	}
	conn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		return
	}
	sessID, _ := h.parseSessionID(req.URL)
	sess := newSession(req, sessID, h.options.DisconnectDelay, h.options.HeartbeatDelay)
	receiver := newWsReceiver(conn, h.options.WebsocketWriteTimeout)
	sess.attachReceiver(receiver)
	if h.handlerFunc != nil {
		go h.handlerFunc(sess)
	}
	readCloseCh := make(chan struct{})
	go func() {
		var d []string
		for {
			err := conn.ReadJSON(&d)
			if err != nil {
				close(readCloseCh)
				return
			}
			sess.accept(d...)
		}
	}()

	select {
	case <-readCloseCh:
	case <-receiver.doneNotify():
	}
	sess.close()
	conn.Close()
}

type wsReceiver struct {
	conn         *websocket.Conn
	closeCh      chan struct{}
	writeTimeout time.Duration
}

func newWsReceiver(conn *websocket.Conn, writeTimeout time.Duration) *wsReceiver {
	return &wsReceiver{
		conn:         conn,
		closeCh:      make(chan struct{}),
		writeTimeout: writeTimeout,
	}
}

func (w *wsReceiver) sendBulk(messages ...string) {
	if len(messages) > 0 {
		w.sendFrame(fmt.Sprintf("a[%s]", strings.Join(transform(messages, quote), ",")))
	}
}

func (w *wsReceiver) sendFrame(frame string) {
	if w.writeTimeout != 0 {
		w.conn.SetWriteDeadline(time.Now().Add(w.writeTimeout))
	}
	if err := w.conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
		w.close()
	}
}

func (w *wsReceiver) close() {
	select {
	case <-w.closeCh: // already closed
	default:
		close(w.closeCh)
	}
}
func (w *wsReceiver) canSend() bool {
	select {
	case <-w.closeCh: // already closed
		return false
	default:
		return true
	}
}
func (w *wsReceiver) doneNotify() <-chan struct{}        { return w.closeCh }
func (w *wsReceiver) interruptedNotify() <-chan struct{} { return nil }
