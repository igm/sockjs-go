package sockjs

import (
	"bytes"
	"code.google.com/p/gorilla/mux"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//jsonp specific connection
type htmlFileConn struct {
	baseConn
	requests chan htmlFileRequest
}

type htmlFileRequest struct {
	rw       http.ResponseWriter
	req      *http.Request
	done     chan bool
	callback string
}

// state function type definition (for xhr connection states)
type htmlFileConnectionState func(*htmlFileConn) htmlFileConnectionState

// run the state machine
func (this *htmlFileConn) run(ctx *context, sessId string, initState htmlFileConnectionState) {
	for state := initState; state != nil; {
		state = state(this)
	}
	ctx.delete(sessId)
}

func (this *context) HtmlfileHandler(rw http.ResponseWriter, req *http.Request) {
	sessId := mux.Vars(req)["sessionid"]
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

	conn, exists := this.getOrCreate(sessId, func() conn {
		conn := &htmlFileConn{
			baseConn: newBaseConn(this),
			requests: make(chan htmlFileRequest),
		}
		return conn
	})

	htmlFile_conn := conn.(*htmlFileConn)
	if !exists {
		go htmlFile_conn.run(this, sessId, hmlFileNewConnection)
		go this.HandlerFunc(htmlFile_conn)
	}
	done := make(chan bool)

	setCors(rw.Header(), req)
	setContentTypeWithoutCache(rw.Header(), "text/html; charset=UTF-8")
	htmlFile_conn.setCookie(rw.Header(), req)

	go func() {
		htmlFile_conn.requests <- htmlFileRequest{
			rw:       rw,
			req:      req,
			done:     done,
			callback: callback,
		}
	}()
	<-done
}

/**************************************************************************************************/
/********** htmlfile state functions ***********************************************************/
/**************************************************************************************************/
func hmlFileNewConnection(conn *htmlFileConn) htmlFileConnectionState {
	req := <-conn.requests
	defer func() { req.done <- true }()
	conn.sendPrelude(req.rw, req.callback)
	conn.sendOpenFrame(req.rw, req.callback)

	flusher := req.rw.(http.Flusher)
	flusher.Flush()

	for bytes_sent := 0; bytes_sent < conn.ResponseLimit; {
		select {
		case frame, ok := <-conn.output():
			if !ok {
				// conn.sendCloseFrame(req.rw, 3000, "Go away!")
				return nil
			}
			n, _ := conn.sendDataFrame(req.rw, frame)
			flusher.Flush()
			bytes_sent = bytes_sent + int(n)
		case <-time.After(conn.HeartbeatDelay): // heartbeat
			// conn.sendHeartbeatFrame(req.rw)
			// case <-conn_interrupted:
			// 	conn.Close()
			// 	return nil // optionally xhrStreamingInterruptedConnection
		}
	}

	return hmlFileNewConnection
}

func (conn *htmlFileConn) setCookie(header http.Header, req *http.Request) {
	if conn.CookieNeeded { // cookie is needed
		cookie, err := req.Cookie(session_cookie)
		if err == http.ErrNoCookie {
			cookie = test_cookie
		}
		cookie.Path = "/"
		header.Add("set-cookie", cookie.String())
	}
}

func (*htmlFileConn) sendPrelude(w io.Writer, callback string) (int64, error) {
	prelude := fmt.Sprintf(_htmlFile, callback)
	// It must be at least 1024 bytes.
	if len(prelude) < 1024 {
		prelude += strings.Repeat(" ", 1024)
	}
	prelude += "\r\n"
	n, err := io.WriteString(w, prelude)
	// return err

	// n, err := fmt.Fprintf(w, "%s(\"o\");\r\n", callback)
	return int64(n), err
}

func (*htmlFileConn) sendOpenFrame(w io.Writer, callback string) (int64, error) {
	n, err := fmt.Fprintf(w, "<script>\np(\"o\");\n</script>\r\n")
	return int64(n), err
}

func (*htmlFileConn) sendDataFrame(w io.Writer, frames ...[]byte) (int64, error) {
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
	// return b.WriteTo(w)
	a := b.Bytes()
	// a = a[0 : len(a)-1]

	n, err := fmt.Fprintf(w, "<script>\np(\"%s\");\n</script>\r\n", string(a))
	return int64(n), err
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
