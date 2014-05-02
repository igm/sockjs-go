package sockjs

import "net/http"

type HandlerFunc func(Conn)

type Handler interface {
	http.Handler
	Prefix() string
}

type Conn interface {
	Recv() (string, error)
	Send(string) error
	// SessionId() string
	Close(status uint32, reason string) error
}
