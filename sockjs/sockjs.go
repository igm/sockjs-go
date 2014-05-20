package sockjs

import "net/http"

type Handler interface {
	http.Handler
	Prefix() string
}

type Conn interface {
	Recv() (string, error)
	Send(string) error
	Close(status uint32, reason string) error
	SessionId() string
}
