package sockjs

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

//websocket specific connection
type websocketProtocol struct{}

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
	// ****** following code was taken from https://github.com/mrlauer/gosockjs
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
	// ****** end
	proto := websocketProtocol{}
	wsh := websocket.Handler(func(net_conn *websocket.Conn) {
		proto.writeOpenFrame(net_conn)
		conn := newConn(this)

		go this.HandlerFunc(conn)

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
					conn.input_channel <- frame
				}
			}
		}()

		for {
			select {
			case frame, ok := <-conn.output_channel:
				if !ok {
					proto.writeClose(net_conn, 3000, "Go away!")
					return
				}
				proto.writeData(net_conn, frame)
			case <-conn_interrupted:
				conn.Close()
				return
			}
		}

	})
	wsh.ServeHTTP(rw, req)
}

func (websocketProtocol) isStreaming() bool   { return true }
func (websocketProtocol) contentType() string { return "" }
func (websocketProtocol) writeOpenFrame(w io.Writer) (int, error) {
	return fmt.Fprint(w, "o")
}
func (websocketProtocol) writeHeartbeat(w io.Writer) (int, error) {
	return fmt.Fprint(w, "h")
}
func (websocketProtocol) writePrelude(w io.Writer) (int, error) {
	return 0, nil
}
func (websocketProtocol) writeClose(w io.Writer, code int, msg string) (int, error) {
	return fmt.Fprintf(w, "c[%d,\"%s\"]", code, msg)
}
func (websocketProtocol) writeData(w io.Writer, frames ...[]byte) (int, error) {
	return w.Write(createDataFrame(frames...))
}
