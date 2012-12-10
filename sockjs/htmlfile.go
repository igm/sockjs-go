package sockjs

import (
	"bytes"
	"code.google.com/p/gorilla/mux"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type htmlfileProtocol struct{ callback string }

func (this *context) HtmlfileHandler(rw http.ResponseWriter, req *http.Request) {
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
		protocolHelper: htmlfileProtocol{callback},
		req:            req,
		rw:             rw,
		sessionId:      sessid,
		done:           make(chan bool),
	}
	this.baseHandler(httpTx)
}

func (htmlfileProtocol) isStreaming() bool   { return true }
func (htmlfileProtocol) contentType() string { return "text/html; charset=UTF-8" }

func (htmlfileProtocol) writeOpenFrame(w io.Writer) (int, error) {
	return fmt.Fprintf(w, "<script>\np(\"o\");\n</script>\r\n")
}
func (htmlfileProtocol) writeHeartbeat(w io.Writer) (int, error) {
	return fmt.Fprintf(w, "<script>\np(\"h\");\n</script>\r\n")
}
func (this htmlfileProtocol) writePrelude(w io.Writer) (int, error) {
	prelude := fmt.Sprintf(_htmlFile, this.callback)
	// It must be at least 1024 bytes.
	if len(prelude) < 1024 {
		prelude += strings.Repeat(" ", 1024)
	}
	prelude += "\r\n"
	return io.WriteString(w, prelude)
}
func (htmlfileProtocol) writeClose(w io.Writer, code int, msg string) (int, error) {
	// TODO check close frame structure with htmlfile protocol
	return fmt.Fprintf(w, "<script>\np(\"c[%d,\"%s\"]\");\n</script>\r\n", code, msg)
}

func (htmlfileProtocol) writeData(w io.Writer, frames ...[]byte) (int, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "a[")
	for n, frame := range frames {
		if n > 0 {
			b.Write([]byte(","))
		}
		sesc := re.ReplaceAllFunc(frame, func(s []byte) []byte {
			return []byte(fmt.Sprintf(`\u%04x`, []rune(string(s))[0]))
		})
		d, _ := json.Marshal(string(sesc))
		b.Write(d[1 : len(d)-1])
	}
	fmt.Fprintf(b, "]")
	a := b.Bytes()
	return fmt.Fprintf(w, "<script>\np(\"%s\");\n</script>\r\n", string(a))
}

var _htmlFile string = `<!doctype html>
<html><head>
  <meta http-equiv="X-UA-Compatible" content="IE=edge" />
  <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
</head><body><h2>Don't panic!</h2>
  <script>
    document.domain = document.domain;
    var c = parent.%s;
    c.start();
    function p(d) {c.message(d);};
    window.onload = function() {c.stop();};
  </script>
`
