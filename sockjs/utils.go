package sockjs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func queueMessage(msg []string, ch chan<- string) {
	for _, m := range msg {
		if len(m) > 0 {
			ch <- m
		}
	}
}

func sendFrame(prefix string, suffix string, writer io.Writer, values []string) (n int, err error) {
	b := bytes.Buffer{}
	b.Write([]byte(prefix)) // a frame
	if values != nil && len(values) > 0 {
		bytes, _ := json.Marshal(&values)
		b.Write(bytes)
	}
	b.Write([]byte(suffix)) // a frame
	return writer.Write(b.Bytes())
}

func newSockJSCon() *SockJsConn {
	return &SockJsConn{
		in:  make(chan string), // input
		out: make(chan string), // output
		hb:  make(chan bool),   // heartbeats
		cch: make(chan bool),   // close
	}
}

func newSockjsSession(sessId string) *SockJsConn {
	sessions[sessId] = newSockJSCon()
	return sessions[sessId]
}

func startHeartbeat(sessId string, s *SockJSHandler) {
	defer func() {
		err := recover()
		if err != nil {
			closeXhrSession(sessId)
		}
	}()
	sockjs := sessions[sessId]
	for {
		time.Sleep(time.Duration(s.Config.HeartbeatDelay) * time.Millisecond)
		select {
		case sockjs.hb <- true: // ok
		case <-time.After(5 * time.Second):
			// if we timeout when sending heartbeat consider sockjs session closed
			closeXhrSession(sessId)
			return
		}
	}
}

/*******************  CORS/HTTP utility methods  ****************************/
func setCors(header http.Header, req *http.Request) {
	header.Add("Access-Control-Allow-Credentials", "true")
	header.Add("Access-Control-Allow-Origin", getOriginHeader(req))
	if allow_headers := req.Header.Get("Access-Control-Request-Headers"); allow_headers != "" && allow_headers != "null" {
		header.Add("Access-Control-Allow-Headers", allow_headers)
	}
}

func setCorsAllowedMethods(header http.Header, req *http.Request, allow_methods string) {
	setCors(header, req)
	header.Add("Access-Control-Allow-Methods", allow_methods)
}

func setExpires(header http.Header) {
	header.Add("Expires", time.Now().AddDate(1, 0, 0).Format(time.RFC1123))
	header.Add("Cache-Control", fmt.Sprintf("public, max-age=%d", 365*24*60*60))
	header.Add("Access-Control-Max-Age", fmt.Sprintf("%d", 365*24*60*60))
}

func setContentTypeWithoutCache(header http.Header, content_type string) {
	header.Add("content-type", content_type)
	header.Add("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
}

func getOriginHeader(req *http.Request) string {
	origin := req.Header.Get("Origin")
	if origin == "" || origin == "null" {
		origin = "*"
	}
	return origin
}
