package sockjs

import (
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

type xhrPollingProtocol struct{ xhrStreamingProtocol }

func (ctx *context) XhrPollingHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessid := vars["sessionid"]

	httpTx := &httpTransaction{
		protocolHelper: xhrPollingProtocol{xhrStreamingProtocol{}},
		req:            req,
		rw:             rw,
		sessionId:      sessid,
		done:           make(chan bool),
	}
	ctx.baseHandler(httpTx)
}
func (xhrPollingProtocol) isStreaming() bool                           { return false }
func (xhrPollingProtocol) contentType() string                         { return "application/javascript; charset=UTF-8" }
func (xhrPollingProtocol) writePrelude(w io.Writer) (n int, err error) { return }
