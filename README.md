What is SockJS?
===============

SockJS is a JavaScript library (for browsers) that provides a WebSocket-like
object. SockJS gives you a coherent, cross-browser, Javascript API
which creates a low latency, full duplex, cross-domain communication
channel between the browser and the web server, with WebSockets or without.
This necessitates the use of a server, which this is one version of, for GO.


SockJS-Go server
================

SockJS-Go is a [SockJS](https://github.com/sockjs/sockjs-client) server written in Go.

To install `sockjs-go` run:

    go get gopkg.in/igm/sockjs-go.v0/sockjs


Versioning
==========

SockJS-Go project adopted [gopkg.in](http://gopkg.in) approach for versioning. Current version is v0 which "is equivalent to labeling the package as alpha or beta quality, and as such the use of these packages as dependencies of stable packages and applications is discouraged".

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

SockJS Protocol Tests Status
----------------------------
```
ERROR: test_haproxy (__main__.WebsocketHixie76)
ERROR: test_firefox_602_connection_header (__main__.WebsocketHybi10)
ERROR: test_headersSanity (__main__.WebsocketHybi10)
FAIL: test_streaming (__main__.Http10)

Ran 68 tests in 1.441s
FAILED (failures=1, errors=3)
```

Important
---------
This library is not production ready and use is not recommended.
