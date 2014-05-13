package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/igm/sockjs-go/sockjs"
)

var (
	websocket = flag.Bool("websocket", true, "enable/disable websocket protocol")
)

func init() {
	flag.Parse()
}

func main() {
	opts := sockjs.DefaultOptions
	opts.Websocket = *websocket
	handler := sockjs.NewHandler("/echo", opts, echoHandler)
	http.Handle("/echo/", handler)
	http.Handle("/", http.FileServer(http.Dir("web/")))
	log.Println("Server started")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func echoHandler(conn sockjs.Conn) {
	log.Println("new sockjs connection established")
	for {
		if msg, err := conn.Recv(); err == nil {
			conn.Send(msg)
			continue
		}
		break
	}
	log.Println("sockjs connection closed")
}
