package sockjs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	cFrame              = closeFrame(2010, "Another connection still open")
	xhrStreamingPrelude = strings.Repeat("h", 2048)
)

func (h *Handler) xhrSend(rw http.ResponseWriter, req *http.Request) {
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
	if _, ok := err.(*json.SyntaxError); ok || err == io.ErrUnexpectedEOF {
		httpError(rw, "Broken JSON encoding.", http.StatusInternalServerError)
		return
	}
	sessionID, err := h.parseSessionID(req.URL)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	h.sessionsMux.Lock()
	defer h.sessionsMux.Unlock()
	if sess, ok := h.sessions[sessionID]; !ok {
		http.NotFound(rw, req)
	} else {
		_ = sess.accept(messages...)                                 // TODO(igm) reponse with SISE in case of err?
		rw.Header().Set("content-type", "text/plain; charset=UTF-8") // Ignored by net/http (but protocol test complains), see https://code.google.com/p/go/source/detail?r=902dc062bff8
		rw.WriteHeader(http.StatusNoContent)
	}
}

type xhrFrameWriter struct{}

func (*xhrFrameWriter) write(w io.Writer, frame string) (int, error) {
	return fmt.Fprintf(w, "%s\n", frame)
}

func (h *Handler) xhrPoll(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("content-type", "application/javascript; charset=UTF-8")
	sess, err := h.sessionByRequest(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	receiver := newHTTPReceiver(rw, req, 1, new(xhrFrameWriter))
	if err := sess.attachReceiver(receiver); err != nil {
		receiver.sendFrame(cFrame)
		receiver.close()
		return
	}

	select {
	case <-receiver.doneNotify():
	case <-receiver.interruptedNotify():
	}
}

func (h *Handler) xhrStreaming(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("content-type", "application/javascript; charset=UTF-8")
	fmt.Fprintf(rw, "%s\n", xhrStreamingPrelude)
	rw.(http.Flusher).Flush()

	sess, err := h.sessionByRequest(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	receiver := newHTTPReceiver(rw, req, h.options.ResponseLimit, new(xhrFrameWriter))

	if err := sess.attachReceiver(receiver); err != nil {
		receiver.sendFrame(cFrame)
		receiver.close()
		return
	}

	select {
	case <-receiver.doneNotify():
	case <-receiver.interruptedNotify():
	}
}
