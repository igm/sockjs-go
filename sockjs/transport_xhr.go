package sockjs

import (
	"encoding/json"
	"io"
	"net/http"
)

func (h *handler) xhrSend(rw http.ResponseWriter, req *http.Request) {
	if req.Body == nil {
		httpError(rw, "Payload expected.", http.StatusInternalServerError)
		return
	}
	var messages []string
	err := json.NewDecoder(req.Body).Decode(&messages)
	if err == io.EOF {
		httpError(rw, "Payload expected.", http.StatusInternalServerError)
		return
	}
	if _, ok := err.(*json.SyntaxError); ok {
		httpError(rw, "Broken JSON encoding.", http.StatusInternalServerError)
		return
	}
	sessionID, _ := h.parseSessionID(req.URL) // TODO(igm) handle error
	if sess, ok := h.sessions[sessionID]; !ok {
		http.NotFound(rw, req)
	} else {
		rw.Header().Set("content-type", "text/plain; charset=UTF-8")
		rw.WriteHeader(http.StatusNoContent)
		err := sess.accept(messages...)
		_ = err // TODO(igm) handle err, sockjs-protocol test does not specify, send 410? 404? or ignore? (session is closing/closed)
	}
}

func (h *handler) xhrPoll(rw http.ResponseWriter, req *http.Request) {
	h.sessionsMux.Lock()
	sessionID, _ := h.parseSessionID(req.URL) // TODO(igm) add err handling, although err should not happen as handler should not pass req in that case
	sess, exists := h.sessions[sessionID]
	if !exists {
		sess = newSession(h.options.DisconnectDelay, h.options.HeartbeatDelay)
		h.sessions[sessionID] = sess
		if h.handlerFunc != nil {
			go h.handlerFunc(sess)
		}
		go func() {
			<-sess.closeCh
			h.sessionsMux.Lock()
			delete(h.sessions, sessionID)
			h.sessionsMux.Unlock()
		}()
	}
	h.sessionsMux.Unlock()

	rw.Header().Set("content-type", "application/javascript; charset=UTF-8")
	receiver := h.newXhrReceiver(rw, 1)
	if err := sess.attachReceiver(receiver); err != nil {
		receiver.sendFrame(closeFrame(2010, "Another connection still open"))
		return
	}
	defer sess.detachReceiver()

	var closeNotifyCh <-chan bool
	if closeNotifier, ok := rw.(http.CloseNotifier); ok {
		closeNotifyCh = closeNotifier.CloseNotify()
	}
	select {
	case <-receiver.done():
	case <-closeNotifyCh:
		sess.close()
	}
}
