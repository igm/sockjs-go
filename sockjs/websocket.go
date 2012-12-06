package sockjs

import (
	"code.google.com/p/go.net/websocket"
	// "io"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

//websocket specific connection
type websocketConn struct {
	baseConn
}

func webSocketPostHandler(w http.ResponseWriter, req *http.Request) {
	rwc, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic("Hijack failed: " + err.Error())
	}
	defer rwc.Close()
	code := http.StatusMethodNotAllowed
	fmt.Fprintf(buf, "HTTP/1.1 %d %s\r\n", code, http.StatusText(code))
	fmt.Fprint(buf, "Content-Length: 0\r\n")
	fmt.Fprint(buf, "Allow: GET\r\n")
	fmt.Fprint(buf, "\r\n")
	buf.Flush()
	return
}

func (this *context) WebSocketHandler(rw http.ResponseWriter, req *http.Request) {
	// I think there is a bug in SockJS. Hybi v13 wants "Origin", not "Sec-WebSocket-Origin"
	if req.Header.Get("Sec-WebSocket-Version") == "13" && req.Header.Get("Origin") == "" {
		req.Header.Set("Origin", req.Header.Get("Sec-WebSocket-Origin"))
	}
	if strings.ToLower(req.Header.Get("Upgrade")) != "websocket" {
		http.Error(rw, `Can "Upgrade" only to "WebSocket".`, http.StatusBadRequest)
		return
	}
	conn := strings.ToLower(req.Header.Get("Connection"))
	// Silly firefox...
	if conn == "keep-alive, upgrade" {
		req.Header.Set("Connection", "Upgrade")
	} else if conn != "upgrade" {
		http.Error(rw, `"Connection" must be "Upgrade".`, http.StatusBadRequest)
		return
	}
	wsh := websocket.Handler(func(net_conn *websocket.Conn) {
		defer net_conn.Close()
		conn := &websocketConn{newBaseConn(this)}
		go this.HandlerFunc(conn)
		conn.sendOpenFrame(net_conn)

		conn_interrupted := make(chan bool)
		go func() {
			data := make([]byte, 32768)
			for {
				n, err := net_conn.Read(data)

				if err != nil {
					conn_interrupted <- true
					return
				}
				if n > 0 { // ignore empty frames
					frame := make([]byte, n)
					copy(frame, data[:n])
					var a []interface{}
					if json.Unmarshal(frame, &a) != nil {
						conn_interrupted <- true
						return
					}
					conn.input() <- frame
				}
			}
		}()

		for {
			select {
			case frame, ok := <-conn.output():
				if !ok {
					conn.sendCloseFrame(net_conn, 3000, "Go away!")
					return
				}
				conn.sendDataFrame(net_conn, frame)
			case <-conn_interrupted:
				conn.Close()
				return
			}
		}

	})
	wsh.ServeHTTP(rw, req)
}

func (*websocketConn) sendOpenFrame(w io.Writer) (int64, error) {
	n, err := w.Write([]byte("o"))
	return int64(n), err
}

func (*websocketConn) sendDataFrame(w io.Writer, frames ...[]byte) (int64, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "a[")
	for n, frame := range frames {
		if n > 0 {
			b.Write([]byte(","))
		}

		b.Write(frame)
	}
	fmt.Fprintf(b, "]")
	return b.WriteTo(w)
}

func (*websocketConn) sendCloseFrame(w io.Writer, code int, msg string) (int64, error) {
	n, err := fmt.Fprintf(w, "c[%d,\"%s\"]", code, msg)
	return int64(n), err
}
