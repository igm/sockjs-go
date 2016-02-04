package sockjs

// Session represents a connection between server and client.
type Session interface {
	// Id returns a session id
	ID() string
	// IP returns the remote address of the connected client
	IP() string
	// Recv reads one text frame from session
	Recv() (string, error)
	// Send sends one text frame to session
	Send(string) error
	// Close closes the session with provided code and reason.
	Close(status uint32, reason string) error
}
