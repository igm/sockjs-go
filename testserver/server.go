package main

import (
	"github.com/igm/sockjs-go/sockjs"
	"log"
	"net/http"
)

func main() {
	log.Println("server started")

	http.Handle("/echo/", sockjs.SockJSHandler{
		Handler: SockJSHandler,
		Config: sockjs.Config{
			SockjsUrl:     "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:     true,
			ResponseLimit: 4096,
			Prefix:        "/echo",
		},
	})

	http.Handle("/disabled_websocket_echo/", sockjs.SockJSHandler{
		Handler: SockJSHandler,
		Config: sockjs.Config{
			SockjsUrl:     "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:     false,
			ResponseLimit: 4096,
			Prefix:        "/disabled_websocket_echo",
		},
	})

	http.Handle("/close/", sockjs.SockJSHandler{
		Handler: SockJSCloseHandler,
		Config: sockjs.Config{
			SockjsUrl:      "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:      false,
			Prefix:         "/close",
			HeartbeatDelay: 5000,
		},
	})

	http.Handle("/", http.FileServer(http.Dir("./www")))
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}

func SockJSCloseHandler(session *sockjs.SockJsConn) {
	session.Close()
}

func SockJSHandler(session *sockjs.SockJsConn) {
	log.Println("Session created")
	for {
		val, err := session.Read()
		if err != nil {
			break
		}
		go func() { session.Write(val) }()
	}

	log.Println("session closed")
}
