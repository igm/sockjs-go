package sockjs

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func (h *handler) sockjsWebsocket(rw http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Origin") != "http://"+req.Host {
		http.Error(rw, "Origin not allowed", 403)
		return
	}
	conn, err := websocket.Upgrade(rw, req, nil, 1024, 1024)
	if hse, ok := err.(websocket.HandshakeError); ok {
		http.Error(rw, hse.Error(), http.StatusBadRequest)
		// rw.WriteHeader(http.StatusBadRequest)
		// fmt.Fprintf(rw, hse.Error())
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	sess := newSession(h.options.DisconnectDelay, h.options.HeartbeatDelay)
	if h.handlerFunc != nil {
		go h.handlerFunc(sess)
	}

	receiver := newWsReceiver(conn)
	sess.attachReceiver(receiver)
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
	conn    *websocket.Conn
	closeCh chan struct{}
}

func newWsReceiver(conn *websocket.Conn) *wsReceiver {
	return &wsReceiver{
		conn:    conn,
		closeCh: make(chan struct{}),
	}
}

func (w *wsReceiver) sendBulk(messages ...string) {
	if len(messages) > 0 {
		w.sendFrame(fmt.Sprintf("a[%s]", strings.Join(transform(messages, quote), ",")))
	}
}

func (w *wsReceiver) sendFrame(frame string) {
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
func (w *wsReceiver) doneNotify() <-chan struct{}        { return w.closeCh }
func (w *wsReceiver) interruptedNotify() <-chan struct{} { return nil }
