package sockjs

import "net/http"

// Handler is a interface that is returned by NewHandler() method that.
type Handler interface {
	http.Handler
	Prefix() string
}

// Session represents a connection between server and client. This is 1 to 1 relation.
type Session interface {
	// Id returns a session id
	ID() string
	// Recv reads one text frame from session
	Recv() (string, error)
	// Send sends one text frame to session
	Send(string) error
	// Close closes the session with provided code and reason.
	Close(status uint32, reason string) error
}
