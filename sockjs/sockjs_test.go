package sockjs_test

import (
	"github.com/igm/sockjs-go/sockjs"
	"net/http"
	"testing"
)

func Test_Install(t *testing.T) {
	t.Log("test started")

}

func ExampleInstall() {
	// Echo Handler
	var handler = func(c sockjs.Conn) {
		for {
			msg, err := c.ReadMessage()
			if err == sockjs.ErrConnectionClosed {
				return
			}
			c.WriteMessage(msg)
		}
	}
	// install echo sockjs in default http handler
	sockjs.Install("/echo", handler, sockjs.DefaultConfig)
	http.ListenAndServe(":8080", nil)
}

func ExampleNewRouter() {
	// Echo Handler
	var handler = func(c sockjs.Conn) {
		for {
			msg, err := c.ReadMessage()
			if err == sockjs.ErrConnectionClosed {
				return
			}
			c.WriteMessage(msg)
		}
	}
	router := sockjs.NewRouter("/echo", handler, sockjs.DefaultConfig)
	http.Handle("/echo", router)
	http.ListenAndServe(":8080", nil)
}
