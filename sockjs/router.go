package sockjs

import (
	"github.com/gorilla/mux"
	"net/http"
)

// Creates new http.Handler that can be used in http.ServeMux (e.g. http.DefaultServeMux)
func NewRouter(baseUrl string, h HandlerFunc, cfg Config) http.Handler {
	router := mux.NewRouter()
	ctx := &context{
		Config:      cfg,
		HandlerFunc: h,
		connections: newConnections(),
	}
	sub := router.PathPrefix(baseUrl).Subrouter()
	sub.HandleFunc("/info", ctx.infoHandler).Methods("GET")
	sub.HandleFunc("/info", infoOptionsHandler).Methods("OPTIONS")
	ss := sub.PathPrefix("/{serverid:[^./]+}/{sessionid:[^./]+}").Subrouter()
	ss.HandleFunc("/xhr_streaming", ctx.XhrStreamingHandler).Methods("POST")
	ss.HandleFunc("/xhr_send", ctx.XhrSendHandler).Methods("POST")
	ss.HandleFunc("/xhr_send", xhrOptions).Methods("OPTIONS")
	ss.HandleFunc("/xhr_streaming", xhrOptions).Methods("OPTIONS")
	ss.HandleFunc("/xhr", ctx.XhrPollingHandler).Methods("POST")
	ss.HandleFunc("/xhr", xhrOptions).Methods("OPTIONS")
	ss.HandleFunc("/eventsource", ctx.EventSourceHandler).Methods("GET")
	ss.HandleFunc("/jsonp", ctx.JsonpHandler).Methods("GET")
	ss.HandleFunc("/jsonp_send", ctx.JsonpSendHandler).Methods("POST")
	ss.HandleFunc("/htmlfile", ctx.HtmlfileHandler).Methods("GET")
	ss.HandleFunc("/websocket", webSocketPostHandler).Methods("POST")
	ss.HandleFunc("/websocket", ctx.WebSocketHandler).Methods("GET")

	sub.HandleFunc("/iframe.html", ctx.iframeHandler).Methods("GET")
	sub.HandleFunc("/iframe-.html", ctx.iframeHandler).Methods("GET")
	sub.HandleFunc("/iframe-{ver}.html", ctx.iframeHandler).Methods("GET")
	sub.HandleFunc("/", welcomeHandler).Methods("GET")
	sub.HandleFunc("/websocket", ctx.RawWebSocketHandler).Methods("GET")
	return router
}

func Install(baseUrl string, h HandlerFunc, cfg Config) http.Handler {
	handler := NewRouter(baseUrl, h, cfg)
	http.Handle(baseUrl+"/", handler)
	http.HandleFunc(baseUrl, welcomeHandler)
	return handler
}

type ctxHandler func(*context, http.ResponseWriter, *http.Request)

// func (this *context) wrap(f ctxHandler) func(w http.ResponseWriter, req *http.Request) {
// 	return func(w http.ResponseWriter, req *http.Request) {
// 		f(this, w, req)
// 	}
// }

func welcomeHandler(rw http.ResponseWriter, req *http.Request) {
	setContentType(rw.Header(), "text/plain; charset=UTF-8")
	// disableCache(rw.Header())
	rw.Write([]byte("Welcome to SockJS!\n"))
}
