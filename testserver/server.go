package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	// TODO(igm) change to gopkg.in v2
	"code.google.com/p/go.net/websocket"
	"github.com/igm/sockjs-go/sockjs"
)

type testHandler []sockjs.Handler

func main() {
	// prepare various options for tests
	echoOptions := sockjs.DefaultOptions
	echoOptions.ResponseLimit = 4096

	disabledWebsocketOptions := sockjs.DefaultOptions
	disabledWebsocketOptions.Websocket = false

	cookieNeededOptions := sockjs.DefaultOptions
	cookieNeededOptions.CookieNeeded = true
	// register various test handlers
	var handlers = []sockjs.Handler{
		sockjs.NewHandler("/echo", echoOptions, echoHandler),
		sockjs.NewHandler("/cookie_needed_echo", cookieNeededOptions, echoHandler),
		sockjs.NewHandler("/close", sockjs.DefaultOptions, closeHandler),
		sockjs.NewHandler("/disabled_websocket_echo", disabledWebsocketOptions, echoHandler),
	}
	log.Fatal(http.ListenAndServe(":8081", testHandler(handlers)))
}

func echoServer(ws *websocket.Conn)  { io.Copy(ws, ws) }
func closeServer(ws *websocket.Conn) { ws.Close() }

func (t testHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/echo/websocket") {
		websocket.Handler(echoServer).ServeHTTP(rw, req)
		return
	}
	if strings.HasPrefix(req.URL.Path, "/close/websocket") {
		websocket.Handler(closeServer).ServeHTTP(rw, req)
		return
	}
	for _, handler := range t {
		if strings.HasPrefix(req.URL.Path, handler.Prefix()) {
			handler.ServeHTTP(rw, req)
			return
		}
	}
	http.NotFound(rw, req)
}

func echoHandler(conn sockjs.Conn) {
	log.Println("New connection created")
	for {
		if msg, err := conn.Recv(); err != nil {
			break
		} else {
			if err := conn.Send(msg); err != nil {
				break
			}
		}
	}
	log.Println("Connection closed")
}

func closeHandler(conn sockjs.Conn) {
	conn.Close(3000, "Go away!")
}
