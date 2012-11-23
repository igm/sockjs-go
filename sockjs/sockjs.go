package sockjs

import (
	"net/http"
	"regexp"
)

/************************* ROUTER ********************************************/

var re_info = regexp.MustCompile("/info$")
var re_sessionUrl = regexp.MustCompile(`/(?:[\w- ]+)/([\w- ]+)/(xhr|xhr_send|xhr_streaming|eventsource|websocket|jsonp|jsonp_send)$`)
var re_iframe = regexp.MustCompile(`/iframe[\w\d-\. ]*\.html$`)

func (s SockJSHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if s.Config.HeartbeatDelay == 0 {
		s.Config.HeartbeatDelay = 25000
	}
	if s.Config.ResponseLimit == 0 {
		s.Config.ResponseLimit = 128 * 1024
	}

	path := req.URL.Path
	method := req.Method

	// log.Println(path, method)

	switch {
	case re_info.MatchString(path) && method == "GET":
		infoHandler(rw, req, &s)

	case re_iframe.MatchString(path) && method == "GET":
		iframeHandler(rw, req, &s)

	case re_sessionUrl.MatchString(path) && method == "GET":
		matches := re_sessionUrl.FindStringSubmatch(path)
		sessId := matches[1]
		service := matches[2]
		switch service {
		case "eventsource":
			eventSourceHandler(rw, req, sessId, &s)
		case "websocket":
			websocketHandler(rw, req, sessId, &s)
		case "jsonp":
			jsonpHandler(rw, req, sessId, &s)
		}

	case re_info.MatchString(path) && method == "OPTIONS":
		infoOptionsHandler(rw, req)

	case re_sessionUrl.MatchString(path) && method == "POST":
		matches := re_sessionUrl.FindStringSubmatch(path)
		sessId := matches[1]
		service := matches[2]
		switch service {
		case "xhr":
			xhrHandler(rw, req, sessId, &s)
		case "xhr_streaming":
			xhrStreamingHandler(rw, req, sessId, &s)
		case "xhr_send":
			xhrHandlerSend(rw, req, sessId, &s)
		case "jsonp_send":
			jsonpHandlerSend(rw, req, sessId, &s)
		}

	case re_sessionUrl.MatchString(path) && method == "OPTIONS":
		xhrHandlerOptions(rw, req)

	default:
		rw.WriteHeader(http.StatusNotFound)
	}

}
