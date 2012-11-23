package sockjs

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
)

func websocketHandler(rw http.ResponseWriter, req *http.Request, _ string, s *SockJSHandler) {
	wsh := websocket.Handler(func(conn *websocket.Conn) {
		defer conn.Close()
		sockjs := newSockJSCon()
		sendFrame("o", "", conn, nil)
		go s.Handler(sockjs)
		go func() {
			msg := []string{}
			for {
				if err := websocket.JSON.Receive(conn, &msg); err != nil {
					if shouldClose(err) {
						sockjs.close()
						return
					}
				} else {
					go queueMessage(msg, sockjs.in)
				}
			}
		}()
		for loop := true; loop; {
			select {
			case msg, ok := <-sockjs.out:
				if !ok {
					return
				}
				if _, err := sendFrame("a", "", conn, []string{msg}); err != nil {
					sockjs.close()
				}
			case _, ok := <-sockjs.cch:
				if !ok {
					return
				}
				// log.Println("Closing session")
				sendFrame(`c[3000,"Go away!"]`, "", conn, nil)
				sockjs.close()
				loop = false
			}
		}
	})
	wsh.ServeHTTP(rw, req)
}

func shouldClose(err error) bool {
	if _, ok := err.(*net.OpError); ok {
		return true
	}
	if err == io.EOF {
		return true
	}
	if json_err, ok := err.(*json.SyntaxError); ok {
		if json_err.Offset == 0 {
			// ll.Printf("empty json (ignore): %#v", err)
			return false
		} else {
			// ll.Printf("invalid json (closing connection): %#v", err)
			return true
		}
	}
	if err.Error() == "unexpected EOF" {
		// ll.Printf("unexpected EOF: %#v", err)
		return true
	}
	// unknown error -> FATAL
	// ll.Printf("unknown error: %#v", err)
	log.Fatal(err)
	return true
}
