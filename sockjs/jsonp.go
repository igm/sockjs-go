package sockjs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

type jsonpProtocol struct{ callback string }

func (ctx *context) JsonpHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessid := vars["sessionid"]

	err := req.ParseForm()
	if err != nil {
		http.Error(rw, "Bad query", http.StatusInternalServerError)
		return
	}
	callback := req.Form.Get("c")
	if callback == "" {
		http.Error(rw, `"callback" parameter required`, http.StatusInternalServerError)
		return
	}

	httpTx := &httpTransaction{
		protocolHelper: jsonpProtocol{callback},
		req:            req,
		rw:             rw,
		sessionId:      sessid,
		done:           make(chan bool),
	}
	ctx.baseHandler(httpTx)
}

func (jsonpProtocol) isStreaming() bool   { return false }
func (jsonpProtocol) contentType() string { return "application/javascript; charset=UTF-8" }

func (proto jsonpProtocol) writeOpenFrame(w io.Writer) (int, error) {
	return fmt.Fprintf(w, "%s(\"o\");\r\n", proto.callback)
}
func (proto jsonpProtocol) writeHeartbeat(w io.Writer) (int, error) {
	return fmt.Fprintf(w, "%s(\"h\");\r\n", proto.callback)
}
func (jsonpProtocol) writePrelude(w io.Writer) (int, error) {
	return 0, nil
}
func (proto jsonpProtocol) writeClose(w io.Writer, code int, msg string) (int, error) {
	return fmt.Fprintf(w, "%s(\"c[%d,\\\"%s\\\"]\");\r\n", proto.callback, code, msg)
}

func (proto jsonpProtocol) writeData(w io.Writer, frames ...[]byte) (int, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s(\"", proto.callback)
	frame := createDataFrame(frames...)
	bb, _ := json.Marshal(string(frame))
	b.Write(bb[1 : len(bb)-1])
	fmt.Fprintf(b, "\");\r\n")
	return w.Write(b.Bytes())
}
