package sockjs

var sessions = NewSessions()

type Sessions struct {
	data map[string]*SockJsConn
	in   chan func()
}

func NewSessions() *Sessions {
	sessions := &Sessions{
		data: map[string]*SockJsConn{},
		in:   make(chan func()),
	}
	go func(sessions *Sessions) {
		for req := range sessions.in {
			req()
		}
	}(sessions)
	return sessions
}

func (s *Sessions) Get(sessId string) (conn *SockJsConn) {
	resp := make(chan bool)
	s.in <- func() {
		conn = s.data[sessId]
		resp <- true
	}
	<-resp
	return
}

func (s *Sessions) GetOrCreate(sessId string) (conn *SockJsConn, new bool) {
	resp := make(chan bool)
	s.in <- func() {
		conn = s.data[sessId]
		new = false
		if conn == nil {
			new = true
			conn = newSockJSCon()
			s.data[sessId] = conn
		}
		resp <- true
	}
	<-resp
	return
}

func (s *Sessions) GetAndDelete(sessId string) (conn *SockJsConn) {
	resp := make(chan bool)
	s.in <- func() {
		conn = s.data[sessId]
		if conn == nil {
			delete(s.data, sessId)
		}
		resp <- true
	}
	<-resp
	return
}
