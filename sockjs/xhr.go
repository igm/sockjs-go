package sockjs

import (
	"encoding/json"
	"io"
	"net/http"
)

func xhrHandler(rw http.ResponseWriter, req *http.Request, sessId string, s *SockJSHandler) {
	if sessions[sessId] == nil {
		sockjs := newSockjsSession(sessId)
		go s.Handler(sockjs)
		go startHeartbeat(sessId, s)

		setCors(rw.Header(), req)
		setContentTypeWithoutCache(rw.Header(), "application/javascript; charset=UTF-8")
		sendFrame("o", "\n", rw, nil)
	} else {
		xhrPolling(rw, req, sessId)
	}
}

func xhrHandlerSend(rw http.ResponseWriter, req *http.Request, sessId string, s *SockJSHandler) {
	if sessions[sessId] != nil {
		sockjs := sessions[sessId]
		decoder := json.NewDecoder(req.Body)
		var value []string
		if err := decoder.Decode(&value); err != nil {
			if err == io.EOF {
				rw.WriteHeader(http.StatusInternalServerError)
				rw.Write([]byte("Payload expected."))
				return
			}
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte("Broken JSON encoding."))
			return
		} else {
			queueMessage(value, sockjs.in)
			setCors(rw.Header(), req)
			setContentTypeWithoutCache(rw.Header(), "text/plain; charset=UTF-8")
			rw.WriteHeader(http.StatusNoContent)
		}
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func xhrHandlerOptions(rw http.ResponseWriter, req *http.Request) {
	setCorsAllowedMethods(rw.Header(), req, "OPTIONS, POST")
	setExpires(rw.Header())
	rw.WriteHeader(http.StatusNoContent)
}

func closeXhrSession(sessId string) {
	if sessions[sessId] != nil {
		sockjs := sessions[sessId]
		delete(sessions, sessId)
		sockjs.close()
	}
}
