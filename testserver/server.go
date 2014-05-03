package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	// TODO(igm) change to gopkg.in v2
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
		sockjs.NewHandler("/close", sockjs.DefaultOptions, closeHandler),
		sockjs.NewHandler("/disabled_websocket_echo", disabledWebsocketOptions, nil),
	}
	// start test handler
	log.Fatal(http.ListenAndServe(":8081", testHandler(handlers)))
}

func (t testHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
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
			fmt.Println("recv err:", err)
			break
		} else {
			fmt.Println(msg)
			if err := conn.Send(msg); err != nil {
				fmt.Println("send err:", err)
				break
			}
		}
	}
	log.Println("Connection closed")
}

func closeHandler(conn sockjs.Conn) {
	conn.Close(3000, "Go away!")
}
