package sockjs

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

// SessionState defines the current state of the session
type SessionState uint32

const (
	// brand new session, need to send "h" to receiver
	SessionOpening SessionState = iota
	// active session
	SessionActive
	// session being closed, sending "closeFrame" to receivers
	SessionClosing
	// closed session, no activity at all, should be removed from handler completely and not reused
	SessionClosed
)

var (
	// ErrSessionNotOpen error is used to denote session not in open state.
	// Recv() and Send() operations are not supported if session is closed.
	ErrSessionNotOpen          = errors.New("sockjs: session not in open state")
	errSessionReceiverAttached = errors.New("sockjs: another receiver already attached")
	errSessionParse            = errors.New("sockjs: unable to parse URL for session")
)

type Session = *session

type session struct {
	mux   sync.RWMutex
	id    string
	req   *http.Request
	state SessionState

	recv       receiver       // protocol dependent receiver (xhr, eventsource, ...)
	sendBuffer []string       // messages to be sent to client
	recvBuffer *messageBuffer // messages received from client to be consumed by application
	closeFrame string         // closeFrame to send after session is closed

	// do not use SockJS framing for raw websocket connections
	raw bool

	// internal timer used to handle session expiration if no receiver is attached, or heartbeats if recevier is attached
	sessionTimeoutInterval time.Duration
	heartbeatInterval      time.Duration
	timer                  *time.Timer
	// once the session timeouts this channel also closes
	closeCh chan struct{}
}

type receiver interface {
	// sendBulk send multiple data messages in frame frame in format: a["msg 1", "msg 2", ....]
	sendBulk(...string) error
	// sendFrame sends given frame over the wire (with possible chunking depending on receiver)
	sendFrame(string) error
	// close closes the receiver in a "done" way (idempotent)
	close()
	canSend() bool
	// done notification channel gets closed whenever receiver ends
	doneNotify() <-chan struct{}
	// interrupted channel gets closed whenever receiver is interrupted (i.e. http connection drops,...)
	interruptedNotify() <-chan struct{}
}

// session is a central component that handles receiving and sending frames. It maintains internal state
func newSession(req *http.Request, sessionID string, sessionTimeoutInterval, heartbeatInterval time.Duration) *session {
	s := &session{
		id:                     sessionID,
		req:                    req,
		heartbeatInterval:      heartbeatInterval,
		recvBuffer:             newMessageBuffer(),
		closeCh:                make(chan struct{}),
		sessionTimeoutInterval: sessionTimeoutInterval,
	}

	s.mux.Lock() // "go test -race" complains if ommited, not sure why as no race can happen here
	s.timer = time.AfterFunc(sessionTimeoutInterval, s.close)
	s.mux.Unlock()
	return s
}

func (s *session) sendMessage(msg string) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.state > SessionActive {
		return ErrSessionNotOpen
	}
	s.sendBuffer = append(s.sendBuffer, msg)
	if s.recv != nil && s.recv.canSend() {
		if err := s.recv.sendBulk(s.sendBuffer...); err != nil {
			return err
		}
		s.sendBuffer = nil
	}
	return nil
}

func (s *session) attachReceiver(recv receiver) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.recv != nil {
		return errSessionReceiverAttached
	}
	s.recv = recv
	go func(r receiver) {
		select {
		case <-r.doneNotify():
			s.detachReceiver()
		case <-r.interruptedNotify():
			s.detachReceiver()
			s.close()
		}
	}(recv)

	if s.state == SessionClosing {
		if !s.raw {
			if err := s.recv.sendFrame(s.closeFrame); err != nil {
				return err
			}
		}
		s.recv.close()
		return nil
	}
	if s.state == SessionOpening {
		if !s.raw {
			if err := s.recv.sendFrame("o"); err != nil {
				return err
			}
		}
		s.state = SessionActive
	}
	if err := s.recv.sendBulk(s.sendBuffer...); err != nil {
		return err
	}
	s.sendBuffer = nil
	s.timer.Stop()
	if s.heartbeatInterval > 0 {
		s.timer = time.AfterFunc(s.heartbeatInterval, s.heartbeat)
	}
	return nil
}

func (s *session) detachReceiver() {
	s.mux.Lock()
	s.timer.Stop()
	s.timer = time.AfterFunc(s.sessionTimeoutInterval, s.close)
	s.recv = nil
	s.mux.Unlock()
}

func (s *session) heartbeat() {
	s.mux.Lock()
	if s.recv != nil { // timer could have fired between Lock and timer.Stop in detachReceiver
		_ = s.recv.sendFrame("h")
		s.timer = time.AfterFunc(s.heartbeatInterval, s.heartbeat)
	}
	s.mux.Unlock()
}

func (s *session) accept(messages ...string) error {
	return s.recvBuffer.push(messages...)
}

// idempotent operation
func (s *session) closing() {
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.state < SessionClosing {
		s.state = SessionClosing
		s.recvBuffer.close()
		if s.recv != nil {
			_ = s.recv.sendFrame(s.closeFrame)
			s.recv.close()
		}
	}
}

// idempotent operation
func (s *session) close() {
	s.closing()
	s.mux.Lock()
	defer s.mux.Unlock()
	if s.state < SessionClosed {
		s.state = SessionClosed
		s.timer.Stop()
		close(s.closeCh)
	}
}

func (s *session) Close(status uint32, reason string) error {
	s.mux.Lock()
	if s.state < SessionClosing {
		s.closeFrame = closeFrame(status, reason)
		s.mux.Unlock()
		s.closing()
		return nil
	}
	s.mux.Unlock()
	return ErrSessionNotOpen
}

func (s *session) Recv() (string, error)                       { return s.recvBuffer.pop(context.Background()) }
func (s *session) RecvCtx(ctx context.Context) (string, error) { return s.recvBuffer.pop(ctx) }
func (s *session) Send(msg string) error                       { return s.sendMessage(msg) }
func (s *session) ID() string                                  { return s.id }
func (s *session) Request() *http.Request                      { return s.req }
func (s *session) GetSessionState() SessionState {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.state
}
