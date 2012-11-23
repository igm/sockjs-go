package sockjs

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
)

func eventSourceHandler(rw http.ResponseWriter, req *http.Request, sessId string, s *SockJSHandler) {
	isNew := false
	if sessions[sessId] == nil {
		isNew = true
		sockjs := newSockjsSession(sessId)
		go s.Handler(sockjs)
		go startHeartbeat(sessId, s)
	}

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

	header := http.Header{}
	setCors(header, req)
	setContentTypeWithoutCache(header, "text/event-stream; charset=UTF-8")
	header.Add("Transfer-Encoding", "chunked")

	bufrw.Write([]byte("HTTP/1.1 200 OK\n"))
	header.Write(bufrw)
	bufrw.Write([]byte("\n"))

	chunked := httputil.NewChunkedWriter(bufrw)

	chunked.Write([]byte("\r\n")) // 2048h frame
	bufrw.Flush()
	if isNew {
		sendFrame("data: o", "\r\n\r\n", chunked, nil)
		bufrw.Flush()
	}
	eventSource(conn, bufrw, chunked, sessId, s)
}

func eventSource(conn net.Conn, bufrw *bufio.ReadWriter, chunkedWriter io.WriteCloser, sessId string, s *SockJSHandler) {
	defer func() {
		chunkedWriter.Close()
		sendFrame("", "\r\n", bufrw, nil)
		bufrw.Flush()
	}()

	sockjs := sessions[sessId]
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
					n, err := sendFrame("data: a", "\r\n\r\n", chunkedWriter, values)
					if err != nil {
						return
					}
					sent = sent + n
					bufrw.Flush()
				}
			}
		case <-sockjs.hb:
			n, err := sendFrame("data: h", "\r\n\r\n", chunkedWriter, nil)
			if err != nil {
				return
			}
			sent = sent + n
			bufrw.Flush()
		}
	}
}
