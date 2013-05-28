package sockjs

import (
	"time"
)

// Conn is a sockjs data-frame oriented network connection.
type Conn interface {
	// Reads message from the open connection. Or returns error if connection is closed.
	ReadMessage() ([]byte, error)
	// Writes message to the open connection. Or returns error if connection is closed.
	WriteMessage([]byte) (int, error)
	// Closes open conenction.  Or returns error if connection is already closed.
	Close() error
	//
	GetSessionID() string
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
