package sockjs

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func jsonpHandler(rw http.ResponseWriter, req *http.Request, sessId string, s *SockJSHandler) {
	if sessions[sessId] == nil {
		callback := req.FormValue("c")
		if callback == "" {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte("\"callback\" parameter required"))
			return
		}
		sockjs := newSockjsSession(sessId)
		go s.Handler(sockjs)
		go startHeartbeat(sessId, s)

		setCors(rw.Header(), req)
		setContentTypeWithoutCache(rw.Header(), "application/javascript; charset=UTF-8")
		sendJsonpOpenFrame(rw, callback)
	} else {
		jsonpPolling(rw, req, sessId)
	}
}

func jsonpHandlerSend(rw http.ResponseWriter, req *http.Request, sessId string, s *SockJSHandler) {
	if sessions[sessId] != nil {
		sockjs := sessions[sessId]
		var value []string
		input := req.FormValue("d")
		if input == "" {
			all, _ := ioutil.ReadAll(req.Body)
			input = string(all)
		}
		if err := json.Unmarshal([]byte(input), &value); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			if json_err, ok := err.(*json.SyntaxError); ok {
				if json_err.Offset == 0 {
					rw.Write([]byte("Payload expected."))
					return
				} else {
					rw.Write([]byte("Broken JSON encoding."))
					return
				}
			}
		} else {
			queueMessage(value, sockjs.in)

			setCors(rw.Header(), req)
			setContentTypeWithoutCache(rw.Header(), "text/plain; charset=UTF-8")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("ok"))
		}
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func jsonpPolling(rw http.ResponseWriter, req *http.Request, sessId string) {
	sockjs := sessions[sessId]

	setContentTypeWithoutCache(rw.Header(), "application/javascript; encoding=UTF-8")
	setCors(rw.Header(), req)

	callback := req.FormValue("c")

	if sockjs.closed {
		rw.WriteHeader(http.StatusOK)
		sendJsonpCloseFrame(rw, callback)
		return
	}

	select {
	case val, ok := <-sockjs.out:
		if !ok {
			return
		}
		values := []string{val}
		for loop := true; loop; {
			select {
			case value, ok := <-sockjs.out:
				if !ok {
					return
				}
				values = append(values, value)
			default:
				loop = false
				sendJsonpDataFrame(rw, callback, values)
			}
		}
	case _, ok := <-sockjs.hb:
		if !ok {
			return
		}
		sendJsonpHeartbeatFrame(rw, callback)
		// sendFrame(callback+"(\"h", "\");\r\n", rw, nil)
	case _, ok := <-sockjs.cch:
		if !ok {
			return
		}
		sendJsonpCloseFrame(rw, callback)
		// sendFrame(callback+`("c[3000,\"Go away!\"]");`, "\r\n", rw, nil)
	}
}

func sendJsonpCloseFrame(rw io.Writer, callback string) (int, error) {
	return sendFrame(callback+`("c[3000,\"Go away!\"]");`, "\r\n", rw, nil)
}

func sendJsonpOpenFrame(rw io.Writer, callback string) (int, error) {
	return sendFrame(callback+"(\"o", "\");\r\n", rw, nil)
}

func sendJsonpHeartbeatFrame(rw io.Writer, callback string) (int, error) {
	return sendFrame(callback+"(\"h", "\");\r\n", rw, nil)
}

func sendJsonpDataFrame(rw io.Writer, callback string, values interface{}) (int, error) {
	vals, _ := json.Marshal(values)
	b := bytes.Buffer{}
	b.Write([]byte(callback + `("a`))
	value := strings.Replace(string(vals), `"`, `\"`, -1)
	value = strings.Replace(value, `\\"`, `\\\"`, -1)
	b.Write([]byte(value))
	// b.Write([]byte())
	b.Write([]byte("\");\r\n"))
	return rw.Write(b.Bytes())
}
