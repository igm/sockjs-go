package sockjs

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

type Handler struct {
	prefix      string
	options     Options
	handlerFunc func(Session)
	mappings    []*mapping

	sessionsMux sync.Mutex
	sessions    map[string]*session
}

const sessionPrefix = "^/([^/.]+)/([^/.]+)"

var sessionRegExp = regexp.MustCompile(sessionPrefix)

// NewHandler creates new HTTP handler that conforms to the basic net/http.Handler interface.
// It takes path prefix, options and sockjs handler function as parameters
func NewHandler(prefix string, opts Options, handlerFunc func(Session)) *Handler {
	if handlerFunc == nil {
		handlerFunc = func(s Session) {}
	}
	h := &Handler{
		prefix:      prefix,
		options:     opts,
		handlerFunc: handlerFunc,
		sessions:    make(map[string]*session),
	}

	h.fillMappingsWithAllowedMethods()

	if opts.Websocket {
		h.mappings = append(h.mappings, newMapping("GET", sessionPrefix+"/websocket$", h.sockjsWebsocket))
	}
	if opts.RawWebsocket {
		h.mappings = append(h.mappings, newMapping("GET", "^/websocket$", h.rawWebsocket))
	}
	return h
}

func (h *Handler) Prefix() string { return h.prefix }

func (h *Handler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// iterate over mappings
	http.StripPrefix(h.prefix, http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		var allowedMethods []string
		for _, mapping := range h.mappings {
			if match, method := mapping.matches(req); match == fullMatch {
				for _, hf := range mapping.chain {
					hf(rw, req)
				}
				return
			} else if match == pathMatch {
				allowedMethods = append(allowedMethods, method)
			}
		}
		if len(allowedMethods) > 0 {
			rw.Header().Set("allow", strings.Join(allowedMethods, ", "))
			rw.Header().Set("Content-Type", "")
			rw.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		http.NotFound(rw, req)
	})).ServeHTTP(rw, req)
}

func (h *Handler) parseSessionID(url *url.URL) (string, error) {
	matches := sessionRegExp.FindStringSubmatch(url.Path)
	if len(matches) == 3 {
		return matches[2], nil
	}
	return "", errSessionParse
}

func (h *Handler) sessionByRequest(req *http.Request) (*session, error) {
	h.sessionsMux.Lock()
	defer h.sessionsMux.Unlock()
	sessionID, err := h.parseSessionID(req.URL)
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

// fillMappingsWithAllowedMethods adds only allowed methods to handler.mappings, by if method is not disabled
func (h *Handler) fillMappingsWithAllowedMethods() {

	xhrCors := xhrCorsFactory(h.options)

	// Default Methods
	h.mappings = []*mapping{
		newMapping("GET", "^[/]?$", welcomeHandler),
		newMapping("OPTIONS", "^/info$", h.options.cookie, xhrCors, cacheFor, h.options.info),
		newMapping("GET", "^/info$", h.options.cookie, xhrCors, noCache, h.options.info),
		// IFrame
		newMapping("GET", "^/iframe[0-9-.a-z_]*.html$", cacheFor, h.iframe),
	}

	// Adding XHR to mapping
	if !h.options.DisableXHR {
		h.mappings = append(h.mappings,
			newMapping("POST", sessionPrefix+"/xhr$", h.options.cookie, xhrCors, noCache, h.xhrPoll),
			newMapping("OPTIONS", sessionPrefix+"/xhr$", h.options.cookie, xhrCors, cacheFor, xhrOptions),
		)
	}

	// Adding XHRStreaming to mapping
	if !h.options.DisableXHRStreaming {
		h.mappings = append(h.mappings,
			newMapping("POST", sessionPrefix+"/xhr_streaming$", h.options.cookie, xhrCors, noCache, h.xhrStreaming),
			newMapping("OPTIONS", sessionPrefix+"/xhr_streaming$", h.options.cookie, xhrCors, cacheFor, xhrOptions),
		)
	}

	// Adding EventSource to mapping
	if !h.options.DisableEventSource {
		h.mappings = append(h.mappings,
			newMapping("GET", sessionPrefix+"/eventsource$", h.options.cookie, xhrCors, noCache, h.eventSource),
		)
	}

	// Adding HtmlFile to mapping
	if !h.options.DisableHtmlFile {
		h.mappings = append(h.mappings,
			newMapping("GET", sessionPrefix+"/htmlfile$", h.options.cookie, xhrCors, noCache, h.htmlFile),
		)
	}

	// Adding JSONP to mapping
	if !h.options.DisableJSONP {
		h.mappings = append(h.mappings,
			newMapping("GET", sessionPrefix+"/jsonp$", h.options.cookie, xhrCors, noCache, h.jsonp),
			newMapping("OPTIONS", sessionPrefix+"/jsonp$", h.options.cookie, xhrCors, cacheFor, xhrOptions),
			newMapping("POST", sessionPrefix+"/jsonp_send$", h.options.cookie, xhrCors, noCache, h.jsonpSend),
		)
	}

	// when adding XHRPoll or/and XHRStreaming xhr_send must be added too (only once)
	if !h.options.DisableXHR || !h.options.DisableXHRStreaming {
		h.mappings = append(h.mappings,
			newMapping("POST", sessionPrefix+"/xhr_send$", h.options.cookie, xhrCors, noCache, h.xhrSend),
			newMapping("OPTIONS", sessionPrefix+"/xhr_send$", h.options.cookie, xhrCors, cacheFor, xhrOptions),
		)
	}
}
