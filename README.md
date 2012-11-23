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

	http.Handle("/echo/", sockjs.SockJSHandler{
		Handler: SockJSHandler,
		Config: sockjs.Config{
			SockjsUrl:     "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:     true,
			ResponseLimit: 4096,
			Prefix:        "/echo",
		},
	})
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
	
}

func SockJSHandler(session *sockjs.SockJsConn) {
	for {
		val, err := session.Read()
		if err != nil {
			break
		}
		go func() { session.Write(val) }()
	}
}
```

Important
---------
This library is not production ready and use is not recommended.
