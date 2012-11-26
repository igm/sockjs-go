package sockjs

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
)

func xhrStreamingHandler(rw http.ResponseWriter, req *http.Request, sessId string, s *SockJSHandler) {
	isNew := false

	if sockjs, new := sessions.GetOrCreate(sessId); new {
		// if sessions[sessId] == nil {
		isNew = true
		// sockjs := newSockjsSession(sessId)
		go s.Handler(sockjs)
		go startHeartbeat(sessId, s)
	}

	setCors(rw.Header(), req)
	setContentTypeWithoutCache(rw.Header(), "application/javascript; charset=UTF-8")
	rw.WriteHeader(http.StatusOK)

	hj, ok := rw.(http.Hijacker)
	if !ok {
		http.Error(rw, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	bufrw.Flush()

	chunked := httputil.NewChunkedWriter(bufrw)

	prelude := make([]byte, 2049)
	for i := range prelude {
		prelude[i] = byte('h')
	}
	prelude[len(prelude)-1] = byte('\n')
	chunked.Write(prelude) // 2048h frame
	bufrw.Flush()
	if isNew {
		chunked.Write([]byte("o\n")) // o frame
		bufrw.Flush()
	}
	xhrStreaming(conn, bufrw, chunked, sessId, s)
}

func xhrStreaming(conn net.Conn, bufrw *bufio.ReadWriter, chunkedWriter io.WriteCloser, sessId string, s *SockJSHandler) {
	defer func() {
		chunkedWriter.Close()
		sendFrame("", "\r\n", bufrw, nil)
		bufrw.Flush()
	}()

	sockjs := sessions.Get(sessId)

	if sockjs.closed {
		sendFrame(`c[3000,"Go away!"]`, "\n", chunkedWriter, nil)
		return
	}

	for sent := 0; sent < s.Config.ResponseLimit; {
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
					n, err := sendFrame("a", "\n", chunkedWriter, values)
					if err != nil {
						return
					}
					sent = sent + n
					bufrw.Flush()
				}
			}
		case _, ok := <-sockjs.hb:
			if !ok {
				return
			}
			n, err := sendFrame("h", "\n", chunkedWriter, nil)
			if err != nil {
				return
			}
			sent = sent + n
			bufrw.Flush()
		case _, ok := <-sockjs.cch:
			if !ok {
				return
			}
			sendFrame(`c[3000,"Go away!"]`, "\n", chunkedWriter, nil)
			return
		}
	}
}
