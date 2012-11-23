// http://golang.org/pkg/testing

package sockjs_test

import (
	"github.com/igm/sockjs-go/sockjs"
	"net/http"
)

func ExampleSockJsConn_Read() {
	config := sockjs.Config{
		SockjsUrl: "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
		Websocket: true,
		Prefix:    "/echo",
	}

	http.HandleFunc("/echo/", sockjs.SockJSHandler{
		Config:  handler,
		Handler: SockJSHandler,
	})

	handler = func(conn *sockjs.SockJsConn) {
		for {
			if msg, err := conn.Read(); err != nil {
				return
			}
			go conn.Write(msg)
		}
	}
}

func ExampleSockJsConn_ReadObject() {
	config := sockjs.Config{
		SockjsUrl: "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
		Websocket: true,
		Prefix:    "/echo",
	}

	http.HandleFunc("/echo/", sockjs.SockJSHandler{
		Config:  handler,
		Handler: SockJSHandler,
	})

	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	handler = func(conn *sockjs.SockJsConn) {
		for {
			var p Person
			if err := conn.ReadObject(&p); err != nil {
				return
			}
			go conn.WriteObject(msg)
		}
	}
}
