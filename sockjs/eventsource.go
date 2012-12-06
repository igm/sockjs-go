package sockjs

import (
	"bytes"
	"code.google.com/p/gorilla/mux"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

//eventSource specific connection
type eventSourceConn struct {
	baseConn
	requests chan clientRequest
}

// state function type definition (for xhr connection states)
type esConnectionState func(*eventSourceConn) esConnectionState

// run the state machine
func (this *eventSourceConn) run(ctx *context, sessId string, initState esConnectionState) {
	for state := initState; state != nil; {
		state = state(this)
	}
	ctx.delete(sessId)
}

func (this *context) EventSourceHandler(rw http.ResponseWriter, req *http.Request) {

	sessId := mux.Vars(req)["sessionid"]
	net_conn, err := hijack(rw)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	conn, exists := this.getOrCreate(sessId, func() conn {
		conn := &eventSourceConn{
			baseConn: newBaseConn(this),
			requests: make(chan clientRequest),
		}
		return conn
	})

	es_conn := conn.(*eventSourceConn)
	if !exists { // create new connection with initial state
		go es_conn.run(this, sessId, eventSourceNewConnection)
		go this.HandlerFunc(es_conn)
	}
	go func() {
		es_conn.requests <- clientRequest{conn: net_conn, req: req}
	}()
}

/**************************************************************************************************/
/********** EventSource state functions ***********************************************************/
/**************************************************************************************************/
func eventSourceNewConnection(conn *eventSourceConn) esConnectionState {
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
	go conn.activeEventSourceConnectionGuard(conn_closed)

	conn_interrupted := make(chan bool)
	go connectionClosedGuard(req.conn, conn_interrupted)

	for bytes_sent := 0; bytes_sent < conn.ResponseLimit; {
		select {
		case frame, ok := <-conn.output():
			if !ok {
				conn.sendCloseFrame(chunked, 3000, "Go away!")
				return nil
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
	return eventSourceNewConnection
}

//  reject other connectins while this one is active
func (conn *eventSourceConn) activeEventSourceConnectionGuard(conn_closed <-chan bool) {
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

/**************************************************************************************************/
/********** EventSource writers *******************************************************************/
/**************************************************************************************************/
func (conn *eventSourceConn) writeHttpHeader(w io.Writer, req *http.Request) (int64, error) {
	b := &bytes.Buffer{}
	fmt.Fprintln(b, "HTTP/1.1", "200 OK")
	header := http.Header{}
	header.Add("content-type", "text/event-stream; charset=UTF-8")
	header.Add("cache-control", "no-store, no-cache, must-revalidate, max-age=0")
	header.Add("transfer-encoding", "chunked")
	header.Add("access-control-allow-credentials", "true")
	header.Add("access-control-allow-origin", getOriginHeader(req))

	if conn.CookieNeeded { // cookie is needed
		cookie, err := req.Cookie(session_cookie)
		if err == http.ErrNoCookie {
			cookie = test_cookie
		}
		cookie.Path = "/"
		header.Add("set-cookie", cookie.String())
	}

	setCors(header, req)
	header.Write(b)
	fmt.Fprintln(b)

	return b.WriteTo(w)
}

func (*eventSourceConn) sendPrelude(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "\r\n")
	return int64(n), err
}

func (*eventSourceConn) sendOpenFrame(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "data: o\r\n\r\n")
	return int64(n), err
}

func (*eventSourceConn) sendDataFrame(w io.Writer, frames ...[]byte) (int64, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "data: a[")
	for n, frame := range frames {
		if n > 0 {
			b.Write([]byte(","))
		}
		sesc := re.ReplaceAllFunc(frame, func(s []byte) []byte {
			return []byte(fmt.Sprintf(`\u%04x`, []rune(string(s))[0]))
		})
		b.Write(sesc)
	}
	fmt.Fprintf(b, "]\r\n\r\n")
	return b.WriteTo(w)
}

func (*eventSourceConn) sendCloseFrame(w io.Writer, code int, msg string) (int64, error) {
	n, err := fmt.Fprintf(w, "data: c[%d,\"%s\"]\r\n\r\n", code, msg)
	return int64(n), err
}

func (*eventSourceConn) sendHeartbeatFrame(w io.Writer) (int64, error) {
	n, err := fmt.Fprintln(w, "data: h\r\n\r\n")
	return int64(n), err
}
