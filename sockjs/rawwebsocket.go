package sockjs

import (
	"code.google.com/p/go.net/websocket"
	"net/http"
)

func (ctx *context) RawWebSocketHandler(rw http.ResponseWriter, req *http.Request) {
	wsh := websocket.Handler(func(net_conn *websocket.Conn) {
		defer net_conn.Close()
		conn := newConn(ctx)
		go ctx.HandlerFunc(conn)

		conn_interrupted := make(chan bool, 1)
		go func() {
			data := make([]byte, 32768) // TODO
			for {
				n, err := net_conn.Read(data)

				if err != nil {
					conn_interrupted <- true
					return
				}
				frame := make([]byte, n+2)
				copy(frame[1:], data[:n])
				conn.input_channel <- frame
			}
		}()

		for {
			select {
			case frame, ok := <-conn.output_channel:
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
