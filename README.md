What is SockJS?
=

SockJS is a JavaScript library (for browsers) that provides a WebSocket-like
object. SockJS gives you a coherent, cross-browser, Javascript API
which creates a low latency, full duplex, cross-domain communication
channel between the browser and the web server, with WebSockets or without.
This necessitates the use of a server, which this is one version of, for GO.


SockJS-Go server
=

SockJS-Go is a [SockJS](https://github.com/sockjs/sockjs-client) server written in Go.

To install **development** version of `sockjs-go` run:

    go get -u gopkg.in/igm/sockjs-go.v0/sockjs

To install **stable** version of `sockjs-go` run (currently v2):

    go get gopkg.in/igm/sockjs-go.v2/sockjs

To install **previoud stable**(deprecated) version of `sockjs-go` run:

    go get gopkg.in/igm/sockjs-go.v1/sockjs


Versioning
-

SockJS-Go project adopted [gopkg.in](http://gopkg.in) approach for versioning. Current development version is v0 which "is equivalent to labeling the package as alpha or beta quality, and as such the use of these packages as dependencies of stable packages and applications is discouraged". This means that version 0 denotes "master" and various API changes might and will be introduced here. 

For **stable** version use v1, which will not break API:

    go get gopkg.in/igm/sockjs-go.v1/sockjs


Example
-

A simplified echo SockJS server could look more or less like:    


```go
package main

import (
	"log"
	"net/http"

	"gopkg.in/igm/sockjs-go.v2/sockjs"
)

func main() {
	// TODO(igm) add simple echo sockjs handler example
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
-

```
ERROR: test_transport (__main__.XhrPolling)
 - this test does not pass dut to a feature in net/http that removes content-type header in case of StatusNoContent response code
 
```

