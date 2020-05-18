package sockjs

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

type Handler struct {
	options     Options
	handlerFunc func(*session)
	router      http.Handler

	sessionsMux sync.Mutex
	sessions    map[string]*session
}

const sessionPrefix = "/{server:[^/.]+}/{session:[^/.]+}"

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

	r := mux.NewRouter()

	mb := middlewareBuilder{opts: opts}

	r.HandleFunc("/", welcomeHandler).Methods(http.MethodGet)
	r.Handle("/info", mb.cookieCorsCacheFor(opts.info)).Methods(http.MethodOptions)
	r.Handle("/info", mb.cookieCorsNoCache(opts.info)).Methods(http.MethodGet)

	r.Handle(sessionPrefix+"/xhr_send", mb.cookieCorsNoCache(h.xhrSend)).Methods(http.MethodPost)
	r.Handle(sessionPrefix+"/xhr_send$", mb.cookieCorsCacheFor(xhrOptions)).Methods(http.MethodOptions)
	r.Handle(sessionPrefix+"/xhr", mb.cookieCorsNoCache(h.xhrPoll)).Methods(http.MethodPost)
	r.Handle(sessionPrefix+"/xhr", mb.cookieCorsCacheFor(xhrOptions)).Methods(http.MethodOptions)
	r.Handle(sessionPrefix+"/xhr_streaming", mb.cookieCorsNoCache(h.xhrStreaming)).Methods(http.MethodPost)
	r.Handle(sessionPrefix+"/xhr_streaming", mb.cookieCorsCacheFor(xhrOptions)).Methods(http.MethodOptions)

	r.Handle(sessionPrefix+"/eventsource", mb.cookieCorsNoCache(h.eventSource)).Methods(http.MethodGet)

	r.Handle(sessionPrefix+"/htmlfile", mb.cookieCorsNoCache(h.htmlFile)).Methods(http.MethodGet)

	r.Handle(sessionPrefix+"/jsonp", mb.cookieCorsNoCache(h.jsonp)).Methods(http.MethodGet)
	r.Handle(sessionPrefix+"/jsonp", mb.cookieCorsCacheFor(xhrOptions)).Methods(http.MethodOptions)
	r.Handle(sessionPrefix+"/jsonp_send", mb.cookieCorsNoCache(h.jsonpSend)).Methods(http.MethodPost)

	r.Handle("/iframe[0-9-.a-z_]*.html", toMW(cacheFor)(http.HandlerFunc(h.iframe))).Methods(http.MethodGet)

	if opts.Websocket {
		r.HandleFunc(sessionPrefix+"/websocket", h.sockjsWebsocket).Methods(http.MethodGet)
	}
	if opts.RawWebsocket {
		r.HandleFunc("/websocket", h.rawWebsocket).Methods(http.MethodGet)
	}

	h.router = http.StripPrefix(prefix, r)
	return h
}

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.router.ServeHTTP(rw, req)
}

func (h *Handler) parseSessionID(req *http.Request) (string, error) {
	vars := mux.Vars(req)
	sessionID := vars["session"]
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
