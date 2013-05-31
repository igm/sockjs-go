package sockjs

import (
	"bytes"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type xhrStreamingProtocol struct{}

func (ctx *context) XhrStreamingHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessid := vars["sessionid"]

	httpTx := &httpTransaction{
		protocolHelper: xhrStreamingProtocol{},
		req:            req,
		rw:             rw,
		sessionId:      sessid,
		done:           make(chan bool),
	}
	ctx.baseHandler(httpTx)
}

func (xhrStreamingProtocol) isStreaming() bool   { return true }
func (xhrStreamingProtocol) contentType() string { return "application/javascript; charset=UTF-8" }

func (xhrStreamingProtocol) writeOpenFrame(w io.Writer) (int, error) {
	return fmt.Fprintln(w, "o")
}
func (xhrStreamingProtocol) writeHeartbeat(w io.Writer) (int, error) {
	return fmt.Fprintln(w, "h")
}
func (xhrStreamingProtocol) writePrelude(w io.Writer) (int, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s\n", strings.Repeat("h", 2048))
	n, err := b.WriteTo(w)
	return int(n), err
}
func (xhrStreamingProtocol) writeClose(w io.Writer, code int, msg string) (int, error) {
	return fmt.Fprintf(w, "c[%d,\"%s\"]\n", code, msg)
}

func (xhrStreamingProtocol) writeData(w io.Writer, frames ...[]byte) (int, error) {
	frame := createDataFrame(frames...)
	b := &bytes.Buffer{}
	b.Write(frame)
	fmt.Fprintf(b, "\n")
	n, err := b.WriteTo(w)
	return int(n), err
}

// author: https://github.com/mrlauer/
var re = regexp.MustCompile("[\x00-\x1f\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufff0-\uffff]")

func createDataFrame(frames ...[]byte) []byte {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "a[")
	for n, frame := range frames {
		if n > 0 {
			b.Write([]byte(","))
		}
		// author: https://github.com/mrlauer/
		sesc := re.ReplaceAllFunc(frame, func(s []byte) []byte {
			return []byte(fmt.Sprintf(`\u%04x`, []rune(string(s))[0]))
		})
		b.Write(sesc)
	}
	fmt.Fprintf(b, "]")
	return b.Bytes()
}
