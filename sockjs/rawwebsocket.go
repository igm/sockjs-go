package sockjs

import (
	"net/http"

	"github.com/gorilla/websocket"
)

func (h *handler) rawWebsocket(rw http.ResponseWriter, req *http.Request) {
	conn, err := websocket.Upgrade(rw, req, nil, WebSocketReadBufSize, WebSocketWriteBufSize)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(rw, `Can "Upgrade" only to "WebSocket".`, http.StatusBadRequest)
		return
	} else if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessID := ""
	sess := newSession(req, sessID, h.options.DisconnectDelay, h.options.HeartbeatDelay)
	sess.raw = true
	if h.handlerFunc != nil {
		go h.handlerFunc(sess)
	}

	receiver := newRawWsReceiver(conn)
	sess.attachReceiver(receiver)
	readCloseCh := make(chan struct{})
	go func() {
		for {
			frameType, p, err := conn.ReadMessage()
			if err != nil {
				close(readCloseCh)
				return
			}
			if frameType == websocket.TextMessage || frameType == websocket.BinaryMessage {
				sess.accept(string(p))
			}
		}
	}()

	select {
	case <-readCloseCh:
	case <-receiver.doneNotify():
	}
	sess.close()
	conn.Close()
}

type rawWsReceiver struct {
	conn    *websocket.Conn
	closeCh chan struct{}
}

func newRawWsReceiver(conn *websocket.Conn) *rawWsReceiver {
	return &rawWsReceiver{
		conn:    conn,
		closeCh: make(chan struct{}),
	}
}

func (w *rawWsReceiver) sendBulk(messages ...string) {
	if len(messages) > 0 {
		for _, m := range messages {
			err := w.conn.WriteMessage(websocket.TextMessage, []byte(m))
			if err != nil {
				w.close()
				break
			}

		}
	}
}

func (w *rawWsReceiver) sendFrame(frame string) {
	if err := w.conn.WriteMessage(websocket.TextMessage, []byte(frame)); err != nil {
		w.close()
	}
}

func (w *rawWsReceiver) close() {
	select {
	case <-w.closeCh: // already closed
	default:
		close(w.closeCh)
	}
}
func (w *rawWsReceiver) canSend() bool {
	select {
	case <-w.closeCh: // already closed
		return false
	default:
		return true
	}
}
func (w *rawWsReceiver) doneNotify() <-chan struct{}        { return w.closeCh }
func (w *rawWsReceiver) interruptedNotify() <-chan struct{} { return nil }
