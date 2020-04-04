package sockjs_test

import (
	"net/http"

	"github.com/igm/sockjs-go/v3/sockjs"
)

func ExampleNewHandler_simple() {
	handler := sockjs.NewHandler("/echo", sockjs.DefaultOptions, func(session sockjs.Session) {
		for {
			if msg, err := session.Recv(); err == nil {
				if session.Send(msg) != nil {
					break
				}
			} else {
				break
			}
		}
	})
	_ = http.ListenAndServe(":8080", handler)
}

func ExampleNewHandler_defaultMux() {
	handler := sockjs.NewHandler("/echo", sockjs.DefaultOptions, func(session sockjs.Session) {
		for {
			if msg, err := session.Recv(); err == nil {
				if session.Send(msg) != nil {
					break
				}
			} else {
				break
			}
		}
	})
	// need to provide path prefix for http.Mux
	http.Handle("/echo/", handler)
	_ = http.ListenAndServe(":8080", nil)
}
