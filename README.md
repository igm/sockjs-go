What is SockJS?
===============

SockJS is a JavaScript library (for browsers) that provides a WebSocket-like
object. SockJS gives you a coherent, cross-browser, Javascript API
which creates a low latency, full duplex, cross-domain communication
channel between the browser and the web server, with WebSockets or without.
This necessitates the use of a server, which this is one version of, for GO.


SockJS-Go server
================

SockJS-Go is a Node.js server side counterpart of
[SockJS-client browser library](https://github.com/sockjs/sockjs-client)
written in CoffeeScript.

To install `sockjs-go` run:

    go get github.com/igm/sockjs-go/sockjs

A simplified echo SockJS server could look more or less like:    


```go
package main

import (
	"github.com/igm/sockjs-go/sockjs"
	"log"
	"net/http"
)

func main() {
    sockjs.Install("/echo", EchoHandler, sockjs.DefaultConfig)
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}

func EchoHandler(conn sockjs.Conn) {
	for {
		if msg, err := conn.ReadMessage(); err == nil {
			go conn.WriteMessage(msg)
		} else {
			return
		}
	}
}
```

Important
---------
This library is not production ready and use is not recommended.
