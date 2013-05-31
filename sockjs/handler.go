package sockjs

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"time"
)

type (
	protocolHelper interface {
		contentType() string
		writePrelude(io.Writer) (int, error)
		writeOpenFrame(io.Writer) (int, error)
		writeHeartbeat(io.Writer) (int, error)

		writeData(io.Writer, ...[]byte) (int, error)
		writeClose(io.Writer, int, string) (int, error)
		isStreaming() bool
	}

	httpTransaction struct {
		protocolHelper
		req       *http.Request
		rw        http.ResponseWriter
		sessionId string
		done      chan bool
	}
)

const session_cookie = "JSESSIONID"

var test_cookie = &http.Cookie{
	Name:  session_cookie,
	Value: "dummy",
}

func (this *context) baseHandler(httpTx *httpTransaction) {
	sessid := httpTx.sessionId

	conn, _ := this.getOrCreate(sessid, func() *conn {
		sockjsConnection := newConn(this)
		go sockjsConnection.run(func() { this.delete(sessid) })
		go this.HandlerFunc(sockjsConnection)
		return sockjsConnection
	})

	// proper HTTP header
	header := httpTx.rw.Header()
	setCors(header, httpTx.req)
	setContentType(header, httpTx.contentType())
	disableCache(header)
	conn.handleCookie(httpTx.rw, httpTx.req)
	httpTx.rw.WriteHeader(http.StatusOK)

	conn.httpTransactions <- httpTx
	<-httpTx.done
	// log.Printf("request processed with protocol: %#v:\n", httpTx.protocolHelper)
}

func (conn *conn) handleCookie(rw http.ResponseWriter, req *http.Request) {
	header := rw.Header()
	if conn.CookieNeeded { // cookie is needed
		cookie, err := req.Cookie(session_cookie)
		if err == http.ErrNoCookie {
			cookie = test_cookie
		}
		cookie.Path = "/"
		header.Add("set-cookie", cookie.String())
	}
}

func openConnectionState(c *conn) connectionStateFn {
	select {
	case <-time.After(c.DisconnectDelay): // timout connection
		// log.Println("timeout in open:", c)
		return nil
	case httpTx := <-c.httpTransactions:

		writer := httpTx.rw
		httpTx.writePrelude(writer)
		writer.(http.Flusher).Flush()
		httpTx.writeOpenFrame(writer)
		writer.(http.Flusher).Flush()

		if httpTx.isStreaming() {
			go func() { c.httpTransactions <- httpTx }()
		} else {
			httpTx.done <- true // let baseHandler finish
		}
		return activeConnectionState
	}
}

func activeConnectionState(c *conn) connectionStateFn {
	select {
	case <-time.After(c.DisconnectDelay): // timout connection
		// log.Println("timeout in active:", c)
		return nil
	case httpTx := <-c.httpTransactions:
		writer := httpTx.rw
		// continue with protocol handling with hijacked connection

		conn, err := hijack(writer)
		if err != nil {
			// TODO
			log.Fatal(err)
		}

		httpTx.done <- true // let baseHandler finish
		chunked := httputil.NewChunkedWriter(conn)
		defer func() {
			chunked.Close()
			conn.Write([]byte("\r\n")) // close chunked data
			conn.Close()
		}()

		// start protocol handling
		conn_closed := make(chan bool, 1)
		defer func() { conn_closed <- true }()
		go c.activeConnectionGuard(conn_closed)

		conn_interrupted := make(chan bool, 1)
		go connectionClosedGuard(conn, conn_interrupted)

		bytes_sent := 0
		for loop := true; loop; {

			select {
			case frame, ok := <-c.output_channel:
				if !ok {
					httpTx.writeClose(chunked, 3000, "Go away!")
					return closedConnectionState
				}
				frames := [][]byte{frame}
				for drain := true; drain; {
					select {
					case frame, ok = <-c.output_channel:
						frames = append(frames, frame)
					default:
						drain = false
					}
				}
				n, _ := httpTx.writeData(chunked, frames...)
				bytes_sent = bytes_sent + n
			case <-time.After(c.HeartbeatDelay):
				httpTx.writeHeartbeat(chunked)
			case <-conn_interrupted:
				c.Close()
				return nil
			}

			if httpTx.isStreaming() {
				if bytes_sent > c.ResponseLimit {
					loop = false
				}
			} else {
				loop = false
			}
		}
		return activeConnectionState
	}
}

func closedConnectionState(c *conn) connectionStateFn {
	select {
	case httpTx := <-c.httpTransactions:
		httpTx.writePrelude(httpTx.rw)
		httpTx.writeClose(httpTx.rw, 3000, "Go away!")
		httpTx.done <- true
		return closedConnectionState
	case <-time.After(c.DisconnectDelay): // timout connection
		// log.Println("timeout in closed:", c)
		return nil
	}
}

//  reject other connectins while this one is active
func (c *conn) activeConnectionGuard(conn_closed <-chan bool) {
	for {
		select {
		case httpTx := <-c.httpTransactions:
			httpTx.writePrelude(httpTx.rw)
			httpTx.writeClose(httpTx.rw, 2010, "Another connection still open")
			httpTx.done <- true
		case <-conn_closed:
			return
		}
	}
}
