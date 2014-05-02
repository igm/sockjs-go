package sockjs

import (
	"errors"
	"sync"
	"time"
)

type sessionState uint32

const (
	sessionOpening sessionState = iota
	sessionActive
	sessionClosing
)

var (
	errSessionNotOpen          = errors.New("session not in open state")
	errSessionReceiverAttached = errors.New("another receiver already attached")
)

type session struct {
	sync.Mutex
	state sessionState
	// protocol dependent receiver (xhr, eventsource, ...)
	recv receiver
	// messages to be sent to client
	sendBuffer []string
	// messages received from client to be consumed by application
	receivedBuffer chan string

	// closeFrame to send after session is closed
	closeFrame string

	sessionTimeoutInterval time.Duration
	heartbeatInterval      time.Duration
	// internal timer used to handle session expiration if no receiver is attached, or heartbeats is recevier is attached
	timer *time.Timer
}

type receiver interface {
	// sendBulk send multiple data messages in frame frame in format: a["msg 1", "msg 2", ....]
	sendBulk(...string)
	// sendFrame sends given frame over the wire (with possible chunking depending on receiver)
	sendFrame(string)
	// done notification channel gets closed whenever receiver ends
	done() chan interface{}
}

// Session is a central component that handles receiving and sending frames. It maintains internal state
func newSession(sessionTimeoutInterval, heartbeatInterval time.Duration) *session {
	s := &session{receivedBuffer: make(chan string), sessionTimeoutInterval: sessionTimeoutInterval}
	s.Lock()
	s.timer = time.AfterFunc(sessionTimeoutInterval, s.sessionTimeout)
	s.Unlock()
	return s
}

func (s *session) sessionTimeout() {
	s.close()
}

func (s *session) sendMessage(msg string) error {
	s.Lock()
	defer s.Unlock()
	if s.state > sessionActive {
		return errSessionNotOpen
	}
	s.sendBuffer = append(s.sendBuffer, msg)
	if s.recv != nil {
		s.recv.sendBulk(s.sendBuffer...)
		s.sendBuffer = nil
	}
	return nil
}

func (s *session) attachReceiver(recv receiver) error {
	s.Lock()
	defer s.Unlock()
	if s.recv != nil {
		return errSessionReceiverAttached
	}
	s.recv = recv
	if s.state == sessionClosing {
		s.recv.sendFrame(s.closeFrame)
		s.recv = nil
		return nil
	}
	if s.state == sessionOpening {
		s.recv.sendFrame("o")
		s.state = sessionActive
	}
	s.recv.sendBulk(s.sendBuffer...)
	s.sendBuffer = nil
	s.timer.Stop()
	s.timer = time.AfterFunc(s.heartbeatInterval, s.heartbeat)
	return nil
}

func (s *session) heartbeat() {
	s.Lock()
	defer s.Unlock()
	if s.recv != nil { // timer could have fired between Lock and timer.Stop in detachReceiver
		s.recv.sendFrame("h")
	}
	s.timer = time.AfterFunc(s.heartbeatInterval, s.heartbeat)
}

func (s *session) detachReceiver() {
	s.Lock()
	defer s.Unlock()
	s.timer.Stop()
	s.timer = time.AfterFunc(s.sessionTimeoutInterval, s.sessionTimeout)
	s.recv = nil

}

func (s *session) accept(messages ...string) {
	for _, msg := range messages {
		s.receivedBuffer <- msg
	}
}

// Conn interface implementation
func (s *session) close() {
	s.Lock()
	defer s.Unlock()
	close(s.receivedBuffer)
	s.state = sessionClosing
	s.timer.Stop()
}

func (s *session) Close(status uint32, reason string) error {
	s.closeFrame = closeFrame(status, reason)
	s.close()
	return nil
}

func (s *session) Recv() (string, error) {
	if s.state > sessionActive {
		return "", errSessionNotOpen
	}
	return <-s.receivedBuffer, nil
}

func (s *session) Send(msg string) error {
	return s.sendMessage(msg)
}
