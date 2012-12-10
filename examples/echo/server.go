package main

import (
	"github.com/igm/sockjs-go/sockjs"
	"log"
	"net/http"
)

func main() {
	log.Println("server started")

	sockjs.Install("/echo", SockJSHandler, sockjs.DefaultConfig)
	http.Handle("/", http.FileServer(http.Dir("./www")))
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}

func SockJSHandler(session sockjs.Conn) {
	log.Println("Session created")
	for {
		val, err := session.ReadMessage()
		if err != nil {
			break
		}
		go func() { session.WriteMessage(val) }()
	}

	log.Println("session closed")
}
