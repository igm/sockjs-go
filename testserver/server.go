package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/igm/sockjs-go/sockjs"
)

func main() {
	// prepare various options for tests
	var echoOptions = sockjs.DefaultOptions
	var disabledWebsocketOptions = sockjs.DefaultOptions
	var cookieNeededOptions = sockjs.DefaultOptions
	echoOptions.ResponseLimit = 4096
	disabledWebsocketOptions.Websocket = false
	cookieNeededOptions.CookieNeeded = true
	// start test handler
	log.Fatal(
		http.ListenAndServe(":8081",
			&testHandler{[]sockjs.Handler{
				sockjs.NewHandler("/echo", sockjs.DefaultOptions, echoHandler),
				sockjs.NewHandler("/close", sockjs.DefaultOptions, closeHandler),
				sockjs.NewHandler("/disabled_websocket_echo", disabledWebsocketOptions, nil),
			}}))
}

// simple http Handler for testing purposes (no redirects, no subpaths ,...)
type testHandler struct{ sockjsHandlers []sockjs.Handler }

func (t *testHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, handler := range t.sockjsHandlers {
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
		msg, err := conn.Recv()
		if err != nil {
			break
		}
		if conn.Send(msg) != nil {
			break
		}
	}
	log.Println("Connection closed")
}

func closeHandler(conn sockjs.Conn) { conn.Close(3000, "Go away!") }
