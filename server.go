package main

import (
	"github.com/igm/sockjs-go-3/sockjs"
	"log"
	"net/http"
	"path"
)

type NoRedirectServer struct {
	*http.ServeMux
}

// Stolen from http package
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	np := path.Clean(p)
	// path.Clean removes trailing slash except for root;
	// put the trailing slash back if necessary.
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}

func (m *NoRedirectServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// To get the sockjs-protocol tests to work, barf if the path is not already clean.
	if req.URL.Path != cleanPath(req.URL.Path) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	http.DefaultServeMux.ServeHTTP(w, req)
}

func main() {
	log.Println("server started")

	cfg_ws_off := sockjs.DefaultConfig
	cfg_ws_off.Websocket = false

	cfg_4096_limit := sockjs.DefaultConfig
	cfg_4096_limit.ResponseLimit = 4096

	cfg_cookie_needed := cfg_4096_limit
	cfg_cookie_needed.CookieNeeded = true

	sockjs.Install("/echo", EchoHandler, cfg_4096_limit)
	sockjs.Install("/close", CloseHandler, sockjs.DefaultConfig)
	sockjs.Install("/cookie_needed_echo", EchoHandler, cfg_cookie_needed)
	sockjs.Install("/disabled_websocket_echo", EchoHandler, cfg_ws_off)

	err := http.ListenAndServe(":8080", new(NoRedirectServer))
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

func CloseHandler(conn sockjs.Conn) {
	conn.Close()
}
