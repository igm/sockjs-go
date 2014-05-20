package sockjs_test

import (
	"net/http"

	"github.com/igm/sockjs-go/sockjs"
)

func ExampleNewHandler_simple() {
	handler := sockjs.NewHandler("/echo", sockjs.DefaultOptions, func(con sockjs.Session) {
		var msg string
		var err error
		for {
			if msg, err = con.Recv(); err != nil {
				break
			}
			if err = con.Send(msg); err != nil {
				break
			}
		}
	})
	http.ListenAndServe(":8080", handler)
}

func ExampleNewHandler_defaultMux() {
	handler := sockjs.NewHandler("/echo", sockjs.DefaultOptions, func(con sockjs.Session) {
		var msg string
		var err error
		for {
			if msg, err = con.Recv(); err != nil {
				break
			}
			if err = con.Send(msg); err != nil {
				break
			}
		}
	})
	// need to provide path prefix for http.Mux
	http.Handle("/echo/", handler)
	http.ListenAndServe(":8080", nil)
}
