package sockjs

import (
	"github.com/igm/sockjs-go-3/sockjs"
	"log"
	"net/http"
)

// This example install echo sockjs server on http.DefaultServeMux using default configuration
func ExampleInstall() {
	echo_handler := func(conn sockjs.Conn) {
		for {
			if msg, err := conn.ReadMessage(); err != nil {
				return
			} else {
				conn.WriteMessage(msg)
			}
		}
	}
	sockjs.Install("/echo", echo_handler, sockjs.DefaultConfig)
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}
