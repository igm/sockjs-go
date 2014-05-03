package sockjs

import (
	"sync"
	"testing"
	"time"
)

func newTestSession() *session {
	// session with long expiration and heartbeats
	return newSession(1000*time.Second, 1000*time.Second)
}

func TestCreateSesion(t *testing.T) {
	session := newTestSession()
	session.sendMessage("this is a message")
	if len(session.sendBuffer) != 1 {
		t.Errorf("Session send buffer should contain 1 message")
	}
	session.sendMessage("another message")
	if len(session.sendBuffer) != 2 {
		t.Errorf("Session send buffer should contain 2 messages")
	}
	if session.state != sessionOpening {
		t.Errorf("Session in wrong state %v, should be %v", session.state, sessionOpening)
	}
}

func TestConcurrentSend(t *testing.T) {
	session := newTestSession()
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			session.sendMessage("message D")
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
	if len(session.sendBuffer) != 100 {
		t.Errorf("Session send buffer should contain 102 messages")
	}
}

func TestAttachReceiver(t *testing.T) {
	session := newTestSession()
	recv := &mockRecv{
		_sendFrame: func(frame string) {
			if frame != "o" {
				t.Errorf("Incorrect open header received")
			}
		},
		_sendBulk: func(...string) {},
	}
	if err := session.attachReceiver(recv); err != nil {
		t.Errorf("Should not return error")
	}
	if session.state != sessionActive {
		t.Errorf("Session in wrong state after receiver attached %d, should be %d", session.state, sessionActive)
	}
	session.detachReceiver()
	recv = &mockRecv{
		_sendFrame: func(frame string) {
			t.Errorf("No frame shold be send, got '%s'", frame)
		},
		_sendBulk: func(...string) {},
	}
	if err := session.attachReceiver(recv); err != nil {
		t.Errorf("Should not return error")
	}
}

func TestSessionTimeout(t *testing.T) {
	sess := newSession(10*time.Millisecond, 10*time.Second)
	time.Sleep(11 * time.Millisecond)
	sess.Lock()
	if sess.state != sessionClosing {
		t.Errorf("Session did not timeout")
	}
	sess.Unlock()
	select {
	case <-sess.closeCh:
	default:
		t.Errorf("sess close notification channel should close")
	}
}

func TestSessionTimeoutOfClosedSession(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexcpected error '%v'", r)
		}
	}()
	sess := newSession(time.Millisecond, time.Second)
	sess.close()
}

func TestAttachReceiverAndCheckHeartbeats(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Unexcpected error '%v'", r)
		}
	}()
	session := newSession(time.Second, 10*time.Millisecond) // 10ms heartbeats
	var frames = []string{}
	var mux sync.Mutex
	recv := &mockRecv{
		_sendBulk: func(...string) {},
		_sendFrame: func(frame string) {
			mux.Lock()
			frames = append(frames, frame)
			mux.Unlock()
		},
	}
	session.attachReceiver(recv)
	time.Sleep(120 * time.Millisecond)
	mux.Lock()
	if len(frames) < 10 || len(frames) > 13 { // should get around 10 heartbeats (120ms/10ms)
		t.Fatalf("Wrong number of frames received, got '%d'", len(frames))
	}
	for i := 1; i < 10; i++ {
		if frames[i] != "h" {
			t.Errorf("Heartbeat no received")
		}
	}
}

func TestAttachReceiverAndRefuse(t *testing.T) {
	session := newTestSession()
	if err := session.attachReceiver(&testRecv{}); err != nil {
		t.Errorf("Should not return error")
	}
	end := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			if err := session.attachReceiver(&testRecv{}); err != errSessionReceiverAttached {
				t.Errorf("Should return error as another receiver is already attached")
			}
			end <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-end
	}
}

func TestDetachRecevier(t *testing.T) {
	session := newTestSession()
	session.detachReceiver()
	session.attachReceiver(&testRecv{})
	session.detachReceiver()

}

func TestSendWithRecv(t *testing.T) {
	session := newTestSession()
	session.sendMessage("message A")
	session.sendMessage("message B")
	if len(session.sendBuffer) != 2 {
		t.Errorf("There should be 2 messages in buffer, but there are %d", len(session.sendBuffer))
	}
	recv := &testRecv{}
	session.attachReceiver(recv)
	if len(recv.messages) != 2 {
		t.Errorf("Reciver should get 2 messages from session, got %d", len(recv.messages))
	}
	session.sendMessage("message C")
	if len(recv.messages) != 3 {
		t.Errorf("Reciver should get 3 messages from session, got %d", len(recv.messages))
	}
	session.sendMessage("message D")
	if len(recv.messages) != 4 {
		t.Errorf("Reciver should get 4 messages from session, got %d", len(recv.messages))
	}
	if len(session.sendBuffer) != 0 {
		t.Errorf("Send buffer should be empty now, but there are %d messaged", len(session.sendBuffer))
	}
}

func TestReceiveMessage(t *testing.T) {
	session := newTestSession()
	go func() {
		session.accept("message A")
		session.accept("message B")
	}()
	if msg := <-session.receivedBuffer; msg != "message A" {
		t.Errorf("Got %s, should be %s", msg, "message A")
	}
	if msg := <-session.receivedBuffer; msg != "message B" {
		t.Errorf("Got %s, should be %s", msg, "message B")
	}
}

func TestSessionClose(t *testing.T) {
	session := newTestSession()
	session.close()
	if _, ok := <-session.receivedBuffer; ok {
		t.Errorf("Session's receive buffer channel should close")
	}
	if err := session.sendMessage("some message"); err != errSessionNotOpen {
		t.Errorf("Session should not accept new message after close")
	}
}

type testRecv struct {
	messages       []string
	openHeaderSent bool
}

func (t *testRecv) sendBulk(messages ...string) { t.messages = append(t.messages, messages...) }
func (t *testRecv) sendFrame(frame string)      { t.openHeaderSent = true }
func (t *testRecv) done() <-chan bool           { return nil }

// Session as Conn Tests
func TestSessionAsConn(t *testing.T) { var _ Conn = newSession(0, 0) }

func TestSessionConnRecv(t *testing.T) {
	s := newTestSession()
	go func() {
		s.receivedBuffer <- "message 1"
	}()
	msg, err := s.Recv()
	if msg != "message 1" || err != nil {
		t.Errorf("Should receive a message without error, got '%s' err '%v'", msg, err)
	}
	s.close()
	msg, err = s.Recv()
	if err != errSessionNotOpen {
		t.Errorf("Session not in correct state, got '%v', expected '%v'", err, errSessionNotOpen)
	}
}

func TestSessionConnSend(t *testing.T) {
	s := newTestSession()
	err := s.Send("message A")
	if err != nil {
		t.Errorf("Session should take messages by default")
	}
	if len(s.sendBuffer) != 1 || s.sendBuffer[0] != "message A" {
		t.Errorf("Message not properly queued in session, got '%v'", s.sendBuffer)
	}
}

func TestSessionConnClose(t *testing.T) {
	s := newTestSession()
	s.state = sessionActive
	err := s.Close(1, "some reason")
	if err != nil {
		t.Errorf("Should not get any error, got '%s'", err)
	}
	if s.closeFrame != "c[1,\"some reason\"]" {
		t.Errorf("Incorrect closeFrame, got '%s'", s.closeFrame)
	}
	if s.state != sessionClosing {
		t.Errorf("Incorrect session state, expected 'sessionClosing', got '%v'", s.state)
	}
	// all the receiver trying to attach shoult get the same close frame
	for i := 0; i < 100; i++ {
		var frames []string
		receiver := &mockRecv{
			_sendBulk:  func(messages ...string) {},
			_sendFrame: func(frame string) { frames = append(frames, frame) },
		}
		s.attachReceiver(receiver)
		if len(frames) != 1 || frames[0] != "c[1,\"some reason\"]" {
			t.Errorf("Close frame not received by receiver, frames '%v'", frames)
		}
	}
}

type mockRecv struct {
	_sendBulk  func(...string)
	_sendFrame func(string)
	_done      func() chan bool
}

func (r *mockRecv) sendBulk(messages ...string) { r._sendBulk(messages...) }
func (r *mockRecv) sendFrame(frame string)      { r._sendFrame(frame) }
func (r *mockRecv) done() <-chan bool           { return r._done() }
