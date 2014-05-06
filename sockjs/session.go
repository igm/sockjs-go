package sockjs

import (
	"encoding/gob"
	"errors"
	"io"
	"sync"
	"time"
)

type sessionState uint32

const (
	// brand new session, need to send "h" to receiver
	sessionOpening sessionState = iota
	// active session
	sessionActive
	// session being closed, sending "closeFrame" to receivers
	sessionClosing
	// closed session, no activity at all, should be removed from handler completely and not reused
	sessionClosed
)

var (
	errSessionNotOpen          = errors.New("sockjs: session not in open state")
	errSessionReceiverAttached = errors.New("sockjs: another receiver already attached")
)

type session struct {
	sync.Mutex
	state sessionState
	// protocol dependent receiver (xhr, eventsource, ...)
	recv receiver
	// messages to be sent to client
	sendBuffer []string
	// messages received from client to be consumed by application
	// receivedBuffer chan string
	msgReader  *io.PipeReader
	msgWriter  *io.PipeWriter
	msgEncoder *gob.Encoder
	msgDecoder *gob.Decoder

	// closeFrame to send after session is closed
	closeFrame string

	// internal timer used to handle session expiration if no receiver is attached, or heartbeats if recevier is attached
	sessionTimeoutInterval time.Duration
	heartbeatInterval      time.Duration
	timer                  *time.Timer
	// once the session timeouts this channel also closes
	closeCh chan bool
}

type receiver interface {
	// sendBulk send multiple data messages in frame frame in format: a["msg 1", "msg 2", ....]
	sendBulk(...string)
	// sendFrame sends given frame over the wire (with possible chunking depending on receiver)
	sendFrame(string)
	// done notification channel gets closed whenever receiver ends
	done() <-chan bool
}

// Session is a central component that handles receiving and sending frames. It maintains internal state
func newSession(sessionTimeoutInterval, heartbeatInterval time.Duration) *session {
	r, w := io.Pipe()
	e := gob.NewEncoder(w)
	d := gob.NewDecoder(r)
	s := &session{
		// receivedBuffer:         make(chan string),
		msgReader:              r,
		msgWriter:              w,
		msgEncoder:             e,
		msgDecoder:             d,
		sessionTimeoutInterval: sessionTimeoutInterval,
		heartbeatInterval:      heartbeatInterval,
		closeCh:                make(chan bool)}
	s.Lock()
	s.timer = time.AfterFunc(sessionTimeoutInterval, s.close)
	s.Unlock()
	return s
}

func (s *session) close() {
	s.Lock()
	defer s.Unlock()
	if s.state < sessionClosing {
		s.msgWriter.Close()
		s.msgReader.Close()
	}
	if s.state < sessionClosed {
		close(s.closeCh)
	}
	s.state = sessionClosed
	s.timer.Stop()
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
		s.timer = time.AfterFunc(s.heartbeatInterval, s.heartbeat)
	}
}

func (s *session) detachReceiver() {
	s.Lock()
	defer s.Unlock()
	s.timer.Stop()
	s.timer = time.AfterFunc(s.sessionTimeoutInterval, s.close)
	s.recv = nil

}

func (s *session) accept(messages ...string) error {
	for _, msg := range messages {
		if err := s.msgEncoder.Encode(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) closing() {
	s.Lock()
	defer s.Unlock()
	if s.state < sessionClosing {
		s.msgReader.Close()
		s.msgWriter.Close()
		s.state = sessionClosing
	}
}

// Conn interface implementation
func (s *session) Close(status uint32, reason string) error {
	s.closeFrame = closeFrame(status, reason)
	s.closing()
	return nil
}

func (s *session) Recv() (string, error) {
	var msg string
	err := s.msgDecoder.Decode(&msg)
	if err == io.ErrClosedPipe {
		err = errSessionNotOpen
	}
	return msg, err
}

func (s *session) Send(msg string) error {
	return s.sendMessage(msg)
}
