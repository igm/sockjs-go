package sockjs

import (
	"code.google.com/p/gorilla/mux"
	"net/http"
	"net/http/httputil"
	"time"
)

/* POST handler */
func (this *context) XhrPollingHandler(rw http.ResponseWriter, req *http.Request) {
	sessId := mux.Vars(req)["sessionid"]
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
	if !exists {
		go xhr_conn.run(this, sessId, xhrPollingNewConnection)
		go this.HandlerFunc(xhr_conn)
	}
	go func() {
		xhr_conn.requests <- clientRequest{conn: net_conn, req: req}
	}()
}

func xhrPollingNewConnection(conn *xhrStreamConn) xhrConnectionState {
	req := <-conn.requests
	chunked := httputil.NewChunkedWriter(req.conn)
	conn.writeHttpHeader(req.conn, req.req)
	conn.sendOpenFrame(chunked)

	chunked.Close()
	req.conn.Write([]byte("\r\n")) // close chunked data
	req.conn.Close()
	return xhrPollingOpenConnection
}

func xhrPollingOpenConnection(conn *xhrStreamConn) xhrConnectionState {
	req := <-conn.requests

	chunked := httputil.NewChunkedWriter(req.conn)
	defer func() {
		chunked.Close()
		req.conn.Write([]byte("\r\n")) // close chunked data
		req.conn.Close()
	}()

	conn.writeHttpHeader(req.conn, req.req)

	conn_closed := make(chan bool)
	defer func() { conn_closed <- true }()
	go conn.activePollingConnectionGuard(conn_closed)

	conn_interrupted := make(chan bool)
	go connectionClosedGuard(req.conn, conn_interrupted)

	select {
	case frame, ok := <-conn.output():
		if !ok {
			conn.sendCloseFrame(chunked, 3000, "Go away!")
			return xhrPollingClosedConnection
		}
		frames := [][]byte{frame}
		for drain := true; drain; {
			select {
			case frame, ok = <-conn.output():
				frames = append(frames, frame)
			default:
				drain = false
			}
		}

		conn.sendDataFrame(chunked, frames...)
		return xhrPollingOpenConnection
	case <-time.After(conn.HeartbeatDelay): // heartbeat
		conn.sendHeartbeatFrame(chunked)
		return xhrPollingOpenConnection
	case <-conn_interrupted:
		conn.Close()
		return nil // final state
	}
	panic("unreachable")
}

func xhrPollingClosedConnection(conn *xhrStreamConn) xhrConnectionState {
	select {
	case req := <-conn.requests:
		chunked := httputil.NewChunkedWriter(req.conn)

		defer func() {
			chunked.Close()
			req.conn.Write([]byte("\r\n")) // close chunked data
			req.conn.Close()
		}()

		conn.writeHttpHeader(req.conn, req.req)
		conn.sendCloseFrame(chunked, 3000, "Go away!")
		return xhrStreamingClosedConnection
	case <-time.After(conn.DisconnectDelay): // timout connection
		return nil
	}
	panic("unreachable")
}

//  reject other connectins while this one is active
func (conn *xhrStreamConn) activePollingConnectionGuard(conn_closed <-chan bool) {
	for {
		select {
		case req := <-conn.requests:
			chunked := httputil.NewChunkedWriter(req.conn)

			conn.writeHttpHeader(req.conn, req.req)
			conn.sendCloseFrame(chunked, 2010, "Another connection still open")

			chunked.Close()
			req.conn.Write([]byte("\r\n")) // close chunked data
			req.conn.Close()
		case <-conn_closed:
			return
		}
	}
}
