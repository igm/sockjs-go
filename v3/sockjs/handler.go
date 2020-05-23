package sockjs

import (
	"net/http"
	"strings"
	"sync"

	"github.com/julienschmidt/httprouter"
)

type Handler struct {
	options     Options
	handlerFunc func(*session)
	router      http.Handler

	sessionsMux sync.Mutex
	sessions    map[string]*session

	infoOptions   http.Handler
	infoGet       http.Handler
	iframeHandler http.Handler

	welcomePath   string
	infoPath      string
	iframePath    string
	websocketPath string
}

const sessionPrefix = "/:server/:session"

func toMW(f func(rw http.ResponseWriter, req *http.Request)) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f(w, r)
			// Call the next handler, which can be another middleware in the chain, or the final handler.
			next.ServeHTTP(w, r)
		})
	}
}

// helper to build common middleware chains.
type middlewareBuilder struct {
	opts Options
}

func (mb *middlewareBuilder) cookieCorsNoCache(f func(rw http.ResponseWriter, req *http.Request)) http.Handler {
	xhrCorsMW := toMW(xhrCorsFactory(mb.opts))
	cookieMW := toMW(mb.opts.cookie)
	noCacheMW := toMW(noCache)
	return cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(f))))
}

func (mb *middlewareBuilder) cookieCorsCacheFor(f func(rw http.ResponseWriter, req *http.Request)) http.Handler {
	xhrCorsMW := toMW(xhrCorsFactory(mb.opts))
	cookieMW := toMW(mb.opts.cookie)
	cacheForMW := toMW(cacheFor)
	return cookieMW(xhrCorsMW(cacheForMW(http.HandlerFunc(f))))
}

// NewHandler creates new HTTP handler that conforms to the basic net/http.Handler interface.
// It takes path prefix, options and sockjs handler function as parameters
func NewHandler(prefix string, opts Options, handlerFunc func(Session)) *Handler {
	if handlerFunc == nil {
		handlerFunc = func(s Session) {}
	}
	h := &Handler{
		options:     opts,
		handlerFunc: handlerFunc,
		sessions:    make(map[string]*session),
	}

	r := httprouter.New()

	mb := middlewareBuilder{opts: opts}
	h.infoOptions = mb.cookieCorsCacheFor(opts.info)
	h.infoGet = mb.cookieCorsNoCache(opts.info)
	h.iframeHandler = toMW(cacheFor)(http.HandlerFunc(h.iframe))

	r.Handler(http.MethodPost, sessionPrefix+"/xhr_send", mb.cookieCorsNoCache(h.xhrSend))
	r.Handler(http.MethodOptions, sessionPrefix+"/xhr_send", mb.cookieCorsCacheFor(xhrOptions))
	r.Handler(http.MethodPost, sessionPrefix+"/xhr", mb.cookieCorsNoCache(h.xhrPoll))
	r.Handler(http.MethodOptions, sessionPrefix+"/xhr", mb.cookieCorsCacheFor(xhrOptions))
	r.Handler(http.MethodPost, sessionPrefix+"/xhr_streaming", mb.cookieCorsNoCache(h.xhrStreaming))
	r.Handler(http.MethodOptions, sessionPrefix+"/xhr_streaming", mb.cookieCorsCacheFor(xhrOptions))

	r.Handler(http.MethodGet, sessionPrefix+"/eventsource", mb.cookieCorsNoCache(h.eventSource))

	r.Handler(http.MethodGet, sessionPrefix+"/htmlfile", mb.cookieCorsNoCache(h.htmlFile))

	r.Handler(http.MethodGet, sessionPrefix+"/jsonp", mb.cookieCorsNoCache(h.jsonp))
	r.Handler(http.MethodOptions, sessionPrefix+"/jsonp", mb.cookieCorsCacheFor(xhrOptions))
	r.Handler(http.MethodPost, sessionPrefix+"/jsonp_send", mb.cookieCorsNoCache(h.jsonpSend))

	if opts.Websocket {
		r.HandlerFunc(http.MethodGet, sessionPrefix+"/websocket", h.sockjsWebsocket)
	}

	h.welcomePath = prefix + "/"
	h.infoPath = prefix + "/info"
	h.iframePath = prefix + "/iframe"
	h.websocketPath = prefix + "/websocket"
	h.router = http.StripPrefix(prefix, r)

	return h
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	if path == h.welcomePath {
		welcomeHandler(rw, req)
		return
	} else if path == h.infoPath {
		if req.Method == http.MethodGet {
			h.infoGet.ServeHTTP(rw, req)
			return
		} else if req.Method == http.MethodOptions {
			h.infoOptions.ServeHTTP(rw, req)
			return
		}
	} else if path == h.websocketPath && h.options.RawWebsocket {
		h.rawWebsocket(rw, req)
		return
	} else if strings.HasPrefix(path, h.iframePath) && req.Method == http.MethodGet {
		h.iframeHandler.ServeHTTP(rw, req)
		return
	}
	h.router.ServeHTTP(rw, req)
}

func (h *Handler) parseSessionID(req *http.Request) (string, error) {
	params := httprouter.ParamsFromContext(req.Context())
	sessionID := params.ByName("session")
	if sessionID == "" {
		return "", errSessionParse
	}
	return sessionID, nil
}

func (h *Handler) sessionByRequest(req *http.Request) (*session, error) {
	h.sessionsMux.Lock()
	defer h.sessionsMux.Unlock()
	sessionID, err := h.parseSessionID(req)
	if err != nil {
		return nil, err
	}
	sess, exists := h.sessions[sessionID]
	if !exists {
		sess = newSession(req, sessionID, h.options.DisconnectDelay, h.options.HeartbeatDelay)
		h.sessions[sessionID] = sess
		go func() {
			<-sess.closeCh
			h.sessionsMux.Lock()
			delete(h.sessions, sessionID)
			h.sessionsMux.Unlock()
		}()
	}
	sess.setCurrentRequest(req)
	return sess, nil
}
