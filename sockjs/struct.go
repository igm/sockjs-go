package sockjs

import (
	"encoding/json"
	"errors"
)

/*
Params: https://github.com/sockjs/sockjs-node
  - sockjs_url (string, required)
  - prefix (string)
  - response_limit (integer)
  - websocket (boolean)
  - jsessionid (boolean or function)
  - log (function(severity, message))
  - heartbeat_delay (milliseconds)
  - disconnect_delay (milliseconds)
*/

/* For detailed explanation see sockjs-node documentation: https://github.com/sockjs/sockjs-node#server-class */
type Config struct {
	SockjsUrl     string // (required) i.e. http://cdn.sockjs.org/sockjs-0.3.2.min.js
	Prefix        string // URL path prefix
	ResponseLimit int    // 128kB
	Websocket     bool   // false
	// JsessionId      bool   // false
	// Log             bool   // false
	HeartbeatDelay  int // 25000ms (25s)
	DisconnectDelay int // 5000ms (5s)
}

// SockJS connection function type.  
type SockJsHandler func(*SockJsConn)

type SockJSHandler struct {
	Handler SockJsHandler
	Config  Config
}

// sockjs error type denotes a connection closed error.
var ErrSocketClosed = errors.New("sockjs connection closed.")

type SockJsConn struct {
	in     chan string // input channel
	out    chan string // output channel
	hb     chan bool   // heartbeat channel
	cch    chan bool   // close channel
	closed bool        // closed flag
}

func (s *SockJsConn) ReadObject(obj interface{}) (err error) {
	str, err := s.Read()
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(str), &obj)
	return
}

/*
Read from sockjs connection. Operation blocks if no message is available.
*/
func (s *SockJsConn) Read() (string, error) {
	msg, ok := <-s.in
	if !ok {
		return msg, ErrSocketClosed
	}
	return msg, nil
}

/*
Writes a string message to sockjs connection. Operation blocks until message is send to client.
*/
func (s *SockJsConn) Write(msg string) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = ErrSocketClosed
		}
	}()
	s.out <- msg
	return
}

//	Closes sockjs session. All Read/Write operations return ErrSocketClosed error after closing the session.
func (s *SockJsConn) Close() {
	defer func() {
		_ = recover()
		// ignore closed channel write
	}()
	s.cch <- true
	s.closed = true
}

func (s *SockJsConn) close() {
	defer func() {
		_ = recover()
		// ignore closed channel write
	}()
	s.closed = true
	close(s.in)
	close(s.out)
	close(s.hb)
	close(s.cch)
}
