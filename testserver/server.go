package main

import (
	"github.com/igm/sockjs-go/sockjs"
	"log"
	"net/http"
	"path"
)

func main() {
	log.Println("server started")

	http.Handle("/echo/", sockjs.SockJSHandler{
		Handler: SockJSHandler,
		Config: sockjs.Config{
			SockjsUrl:     "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:     true,
			ResponseLimit: 4096,
			Prefix:        "/echo",
		},
	})
	http.Handle("/disabled_websocket_echo/", sockjs.SockJSHandler{
		Handler: SockJSHandler,
		Config: sockjs.Config{
			SockjsUrl:     "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:     false,
			ResponseLimit: 4096,
			Prefix:        "/disabled_websocket_echo",
		},
	})

	http.Handle("/close/", sockjs.SockJSHandler{
		Handler: SockJSCloseHandler,
		Config: sockjs.Config{
			SockjsUrl:      "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
			Websocket:      false,
			Prefix:         "/close",
			HeartbeatDelay: 5000,
		},
	})

	http.Handle("/", http.FileServer(http.Dir("./www")))
	err := http.ListenAndServe(":8080", &TestServer{})
	log.Fatal(err)
}

func SockJSCloseHandler(session *sockjs.SockJsConn) {
	session.Close()
}

func SockJSHandler(session *sockjs.SockJsConn) {
	log.Println("Session created")
	for {
		val, err := session.Read()
		if err != nil {
			break
		}
		go func() { session.Write(val) }()
	}

	log.Println("session closed")
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

type TestServer struct {
	*http.ServeMux
}

func (m *TestServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// To get the sockjs-protocol tests to work, barf if the path is not already clean.
	if req.URL.Path != cleanPath(req.URL.Path) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	http.DefaultServeMux.ServeHTTP(w, req)
}
