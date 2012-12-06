package sockjs

import (
	"time"
)

type Conn interface {
	ReadMessage() ([]byte, error)
	WriteMessage([]byte) (int, error)
	Close() error
}

type HandlerFunc func(Conn)

type Config struct {
	SockjsUrl       string
	Websocket       bool
	ResponseLimit   int
	HeartbeatDelay  time.Duration
	DisconnectDelay time.Duration
	CookieNeeded    bool
}

// Default Configuration with 128kB response limit
var DefaultConfig = Config{
	SockjsUrl:       "http://cdn.sockjs.org/sockjs-0.3.2.min.js", // default JS
	Websocket:       true,                                        // enabled websocket
	ResponseLimit:   128 * 1024,                                  // 128kB
	HeartbeatDelay:  time.Duration(25 * time.Second),             // 25s
	DisconnectDelay: time.Duration(5 * time.Second),              // 5s
	CookieNeeded:    false,
}
