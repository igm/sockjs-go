package sockjs

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

type Handler struct {
	options     Options
	handlerFunc func(*session)
	router      *mux.Router

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

// NewHandler creates new HTTP handler that conforms to the basic net/http.Handler interface.
// It takes path prefix, options and sockjs handler function as parameters
func NewHandler(opts Options, handlerFunc func(Session)) *Handler {
	if handlerFunc == nil {
		handlerFunc = func(s Session) {}
	}
	h := &Handler{
		options:     opts,
		handlerFunc: handlerFunc,
		sessions:    make(map[string]*session),
	}

	xhrCorsMW := toMW(xhrCorsFactory(opts))
	cookieMW := toMW(opts.cookie)
	cacheForMW := toMW(cacheFor)
	noCacheMW := toMW(noCache)

	r := mux.NewRouter()
	r.HandleFunc("/", welcomeHandler).Methods(http.MethodGet)
	r.Handle("/info", cookieMW(xhrCorsMW(cacheForMW(http.HandlerFunc(opts.info))))).Methods(http.MethodOptions)
	r.Handle("/info", xhrCorsMW(noCacheMW(http.HandlerFunc(opts.info)))).Methods(http.MethodGet)

	r.Handle(sessionPrefix+"/xhr_send", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.xhrSend))))).Methods(http.MethodPost)
	r.Handle(sessionPrefix+"/xhr_send$", cookieMW(xhrCorsMW(cacheForMW(http.HandlerFunc(xhrOptions))))).Methods(http.MethodOptions)
	r.Handle(sessionPrefix+"/xhr", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.xhrPoll))))).Methods(http.MethodPost)
	r.Handle(sessionPrefix+"/xhr", cookieMW(xhrCorsMW(cacheForMW(http.HandlerFunc(xhrOptions))))).Methods(http.MethodOptions)
	r.Handle(sessionPrefix+"/xhr_streaming", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.xhrStreaming))))).Methods(http.MethodPost)
	r.Handle(sessionPrefix+"/xhr_streaming", cookieMW(xhrCorsMW(cacheForMW(http.HandlerFunc(xhrOptions))))).Methods(http.MethodOptions)

	r.Handle(sessionPrefix+"/eventsource", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.eventSource))))).Methods(http.MethodGet)

	r.Handle(sessionPrefix+"/htmlfile", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.htmlFile))))).Methods(http.MethodGet)

	r.Handle(sessionPrefix+"/jsonp", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.jsonp))))).Methods(http.MethodGet)
	r.Handle(sessionPrefix+"/jsonp", cookieMW(xhrCorsMW(cacheForMW(http.HandlerFunc(xhrOptions))))).Methods(http.MethodOptions)
	r.Handle(sessionPrefix+"/jsonp_send", cookieMW(xhrCorsMW(noCacheMW(http.HandlerFunc(h.jsonpSend))))).Methods(http.MethodPost)

	r.Handle("/iframe[0-9-.a-z_]*.html", cacheForMW(http.HandlerFunc(h.iframe))).Methods(http.MethodGet)

	if opts.Websocket {
		r.HandleFunc(sessionPrefix+"/websocket", h.sockjsWebsocket).Methods(http.MethodGet)
	}
	if opts.RawWebsocket {
		r.HandleFunc("/websocket", h.rawWebsocket).Methods(http.MethodGet)
	}

	h.router = r
	return h
}

func (h *Handler) Prefix() string { return "" }

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
