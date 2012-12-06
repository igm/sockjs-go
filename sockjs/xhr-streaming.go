package sockjs

import (
	"bytes"
	"code.google.com/p/gorilla/mux"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"
	"time"
)

/* POST handler */
func (this *context) XhrStreamingHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessId := vars["sessionid"]

	net_conn, err := hijack(rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	conn, exists := this.getOrCreate(sessId, func() conn {
		conn := &xhrStreamConn{
			baseConn: newBaseConn(this),
			requests: make(chan clientRequest),
		}
		return conn
	})

	xhr_conn := conn.(*xhrStreamConn)
	if !exists { // create new connection with initial state
		go xhr_conn.run(this, sessId, xhrStreamingNewConnection)
		go this.HandlerFunc(xhr_conn)
	}
	go func() {
		xhr_conn.requests <- clientRequest{conn: net_conn, req: req}
	}()
}

/* OPTIONS handler */
func xhrOptions(rw http.ResponseWriter, req *http.Request) {
	setCors(rw.Header(), req)
	setCorsAllowedMethods(rw.Header(), req, "OPTIONS, POST")
	setExpires(rw.Header())
	rw.WriteHeader(http.StatusNoContent)
}

/*************************************/
/** Connection State Functions *******/
/*************************************/
func xhrStreamingNewConnection(conn *xhrStreamConn) xhrConnectionState {
	req := <-conn.requests
	chunked := httputil.NewChunkedWriter(req.conn)

	defer func() {
		chunked.Close()
		req.conn.Write([]byte("\r\n")) // close chunked data
		req.conn.Close()
	}()

	conn.writeHttpHeader(req.conn, req.req)
	conn.sendPrelude(chunked)
	conn.sendOpenFrame(chunked)

	conn_closed := make(chan bool)
	defer func() { conn_closed <- true }()
	go conn.activeStreamingConnectionGuard(conn_closed)

	conn_interrupted := make(chan bool)
	go connectionClosedGuard(req.conn, conn_interrupted)

	for bytes_sent := 0; bytes_sent < conn.ResponseLimit; {
		select {
		case frame, ok := <-conn.output():
			if !ok {
				conn.sendCloseFrame(chunked, 3000, "Go away!")
				return xhrStreamingClosedConnection
			}
			n, _ := conn.sendDataFrame(chunked, frame)
			bytes_sent = bytes_sent + int(n)
		case <-time.After(conn.HeartbeatDelay): // heartbeat
			conn.sendHeartbeatFrame(chunked)
		case <-conn_interrupted:
			conn.Close()
			return nil // optionally xhrStreamingInterruptedConnection
		}
	}
	return xhrStreamingNewConnection
}

func xhrStreamingClosedConnection(conn *xhrStreamConn) xhrConnectionState {
	select {
	case req := <-conn.requests:
		chunked := httputil.NewChunkedWriter(req.conn)

		defer func() {
			chunked.Close()
			req.conn.Write([]byte("\r\n")) // close chunked data
			req.conn.Close()
		}()

		conn.writeHttpHeader(req.conn, req.req)
		conn.sendPrelude(chunked)
		conn.sendCloseFrame(chunked, 3000, "Go away!")
		return xhrStreamingClosedConnection
	case <-time.After(conn.DisconnectDelay): // timout connection
		return nil
	}
	panic("unreachable")
}

func xhrStreamingInterruptedConnection(conn *xhrStreamConn) xhrConnectionState {
	select {
	case req := <-conn.requests:
		chunked := httputil.NewChunkedWriter(req.conn)
		conn.writeHttpHeader(req.conn, req.req)
		conn.sendPrelude(chunked)
		conn.sendCloseFrame(chunked, 1002, "Connection interrupted!")
		chunked.Close()
		req.conn.Write([]byte("\r\n")) // close chunked data
		req.conn.Close()
		return xhrStreamingInterruptedConnection
	case <-time.After(conn.DisconnectDelay): // timout connection
		return nil
	}
	panic("unreachable")
}

//  reject other connectins while this one is active
func (conn *xhrStreamConn) activeStreamingConnectionGuard(conn_closed <-chan bool) {
	for {
		select {
		case req := <-conn.requests:
			chunked := httputil.NewChunkedWriter(req.conn)

			conn.writeHttpHeader(req.conn, req.req)
			conn.sendPrelude(chunked)
			conn.sendCloseFrame(chunked, 2010, "Another connection still open")

			chunked.Close()
			req.conn.Write([]byte("\r\n")) // close chunked data
			req.conn.Close()
		case <-conn_closed:
			return
		}
	}
}

/*************************************************************************************************/
/* sockjs protocol writer xhr-streaming function
/*************************************************************************************************/
const session_cookie = "JSESSIONID"

var test_cookie = &http.Cookie{
	Name:  session_cookie,
	Value: "dummy",
}

func (conn *baseConn) writeHttpHeader(w io.Writer, req *http.Request) (int64, error) {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "HTTP/1.1", "200 OK")
	header := http.Header{}

	setCors(header, req)
	setContentTypeWithoutCache(header, "application/javascript; charset=UTF-8")
	header.Add("transfer-encoding", "chunked")
	// header.Add("content-type", "application/javascript; charset=UTF-8")
	// header.Add("cache-control", "no-store, no-cache, must-revalidate, max-age=0")
	//	header.Add("access-control-allow-credentials", "true")
	//	header.Add("access-control-allow-origin", getOriginHeader(req))

	if conn.CookieNeeded { // cookie is needed
		cookie, err := req.Cookie(session_cookie)
		if err == http.ErrNoCookie {
			cookie = test_cookie
		}
		cookie.Path = "/"
		header.Add("set-cookie", cookie.String())
	}

	// setCors(header, req)
	header.Write(b)
	fmt.Fprintln(b)

	return b.WriteTo(w)
}

func (*xhrStreamConn) sendPrelude(w io.Writer) (int64, error) {
	b := &bytes.Buffer{}
	prelude := strings.Repeat("h", 2048)
	fmt.Fprintf(b, prelude)
	// for i := 0; i < 32; i++ { // prelude 2048*'h'
	// 	fmt.Fprint(b, "hhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhhh")
	// }
	fmt.Fprintln(b)
	return b.WriteTo(w)
}

func (*xhrStreamConn) sendOpenFrame(w io.Writer) (int64, error) {
	n, err := fmt.Fprintln(w, "o")
	return int64(n), err
}

var re = regexp.MustCompile("[\x00-\x1f\u200c-\u200f\u2028-\u202f\u2060-\u206f\ufff0-\uffff]")

func (*xhrStreamConn) sendDataFrame(w io.Writer, frames ...[]byte) (int64, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "a[")
	for n, frame := range frames {
		if n > 0 {
			b.Write([]byte(","))
		}

		sesc := re.ReplaceAllFunc(frame, func(s []byte) []byte {
			return []byte(fmt.Sprintf(`\u%04x`, []rune(string(s))[0]))
		})

		b.Write(sesc)
	}
	fmt.Fprintf(b, "]\n")
	return b.WriteTo(w)
}

func (*xhrStreamConn) sendHeartbeatFrame(w io.Writer) (int64, error) {
	n, err := fmt.Fprintln(w, "h")
	return int64(n), err
}

func (*xhrStreamConn) sendCloseFrame(w io.Writer, code int, msg string) (int64, error) {
	n, err := fmt.Fprintf(w, "c[%d,\"%s\"]\n", code, msg)
	return int64(n), err
}
