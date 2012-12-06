package sockjs

import (
	"code.google.com/p/go.net/websocket"
	"net/http"
)

func (this *context) RawWebSocketHandler(rw http.ResponseWriter, req *http.Request) {
	wsh := websocket.Handler(func(net_conn *websocket.Conn) {
		defer net_conn.Close()
		conn := newBaseConn(this)
		go this.HandlerFunc(&conn)
		// conn.sendOpenFrame(net_conn)

		conn_interrupted := make(chan bool)
		go func() {
			data := make([]byte, 32768)
			for {
				n, err := net_conn.Read(data)

				if err != nil {
					conn_interrupted <- true
					return
				}
				frame := make([]byte, n+2)
				copy(frame[1:], data[:n])
				conn.input() <- frame
			}
		}()

		for {
			select {
			case frame, ok := <-conn.output():
				if !ok {
					return
				}
				net_conn.Write(frame)
			case <-conn_interrupted:
				conn.Close()
				return
			}
		}
	})
	wsh.ServeHTTP(rw, req)
}
