package sockjs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (h *handler) jsonp(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("content-type", "application/javascript; charset=UTF-8")

	req.ParseForm()
	callback := req.Form.Get("c")
	if callback == "" {
		http.Error(rw, `"callback" parameter required`, http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.(http.Flusher).Flush()

	sess, _ := h.sessionByRequest(req)
	recv := newHttpReceiver(rw, 1, &jsonpFrameWriter{callback})
	if err := sess.attachReceiver(recv); err != nil {
		recv.sendFrame(cFrame)
		return
	}
	select {
	case <-recv.doneNotify():
	case <-recv.interruptedNotify():
	}
}

func (h *handler) jsonp_send(rw http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	var data io.Reader = req.Body

	formReader := strings.NewReader(req.PostFormValue("d"))
	if formReader.Len() != 0 {
		data = formReader
	}

	var messages []string
	err := json.NewDecoder(data).Decode(&messages)
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
	if sess, ok := h.sessions[sessionID]; !ok {
		http.NotFound(rw, req)
	} else {
		_ = sess.accept(messages...) // TODO(igm) reponse with http.StatusInternalServerError in case of err?
		rw.Header().Set("content-type", "text/plain; charset=UTF-8")
		rw.Write([]byte("ok"))
	}
}

type jsonpFrameWriter struct {
	callback string
}

func (j *jsonpFrameWriter) write(w io.Writer, frame string) (int, error) {
	payload, _ := json.Marshal(frame)
	return fmt.Fprintf(w, "%s(%s);\r\n", j.callback, string(payload))
}
