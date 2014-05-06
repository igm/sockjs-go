package sockjs

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestHandler_XhrSendNilBody(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/non_existing_session/xhr_send", nil)
	h.xhrSend(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusInternalServerError)
	}
	if rec.Body.String() != "Payload expected." {
		t.Errorf("Unexcpected body received: '%s'", rec.Body.String())
	}
}

func TestHandler_XhrSendEmptyBody(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/non_existing_session/xhr_send", strings.NewReader(""))
	h.xhrSend(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusInternalServerError)
	}
	if rec.Body.String() != "Payload expected." {
		t.Errorf("Unexcpected body received: '%s'", rec.Body.String())
	}
}

func TestHandler_SessionByRequest(t *testing.T) {
	h := newTestHandler()
	var handlerFuncCalled = make(chan Conn)
	h.handlerFunc = func(conn Conn) {
		handlerFuncCalled <- conn
	}
	req, _ := http.NewRequest("POST", "/server/sessionid/whatever/follows", nil)
	if sess, err := h.sessionByRequest(req); sess == nil || err != nil {
		t.Errorf("Session should be returned")
	} else {
		select {
		case conn := <-handlerFuncCalled: // ok
			if conn != sess {
				t.Errorf("Handler was not passed correct session")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("HandlerFunc was not called")
		}
	}

	req2, _ := http.NewRequest("POST", "/server/sessionid", nil)
	if sess, err := h.sessionByRequest(req2); sess != nil || err == nil {
		t.Errorf("Expected error, got session: '%v'", sess)
	}
}

func TestHandler_XhrSendToExistingSession(t *testing.T) {
	h := newTestHandler()
	sess := newSession(time.Second, time.Second)
	h.sessions["session"] = sess

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"some message\"]"))
	go func() { h.xhrSend(rec, req) }()
	msg, _ := sess.Recv()
	if msg != "some message" {
		t.Errorf("Incorrect message in the channel, should be '%s', was '%s'", "some message", msg)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("Wrong response status received %d, should be %d", rec.Code, http.StatusNoContent)
	}
	if rec.Header().Get("content-type") != "text/plain; charset=UTF-8" {
		t.Errorf("Wrong content type received '%s'", rec.Header().Get("content-type"))
	}
}

func TestHandler_XhrSendInvalidInput(t *testing.T) {
	h := newTestHandler()
	req, _ := http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("some invalid message frame"))
	rec := httptest.NewRecorder()
	h.xhrSend(rec, req)
	if rec.Code != http.StatusInternalServerError || rec.Body.String() != "Broken JSON encoding." {
		t.Errorf("Unexpected response, got '%d,%s' expected '%d,Broken JSON encoding.'", rec.Code, rec.Body.String(), http.StatusInternalServerError)
	}

	// unexpected EOF
	req, _ = http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"x"))
	rec = httptest.NewRecorder()
	h.xhrSend(rec, req)
	if rec.Code != http.StatusInternalServerError || rec.Body.String() != "Broken JSON encoding." {
		t.Errorf("Unexpected response, got '%d,%s' expected '%d,Broken JSON encoding.'", rec.Code, rec.Body.String(), http.StatusInternalServerError)
	}
}

func TestHandler_XhrSendSessionNotFound(t *testing.T) {
	h := handler{}
	req, _ := http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"some message\"]"))
	rec := httptest.NewRecorder()
	h.xhrSend(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
}

type testReceiver struct {
	doneCh chan bool
	frames []string
}

func (t *testReceiver) done() <-chan bool           { return t.doneCh }
func (t *testReceiver) sendBulk(messages ...string) {}
func (t *testReceiver) sendFrame(frame string)      { t.frames = append(t.frames, frame) }

func TestHandler_XhrPoll(t *testing.T) {
	doneCh := make(chan bool)
	rec := &testReceiver{doneCh, nil}
	h := &handler{
		sessions:       make(map[string]*session),
		newXhrReceiver: func(http.ResponseWriter, uint32) receiver { return rec },
	}
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	var sess *session
	go func() {
		h.sessionsMux.Lock()
		defer h.sessionsMux.Unlock()

		if sess = h.sessions["session"]; sess == nil {
			t.Errorf("Session not properly created")
		}
		sess.Lock()
		if sess.recv != rec {
			t.Errorf("Receiver not properly attached to session")
		}
		sess.Unlock()
		close(doneCh)
	}()
	h.xhrPoll(rw, req)
	if sess.recv != nil {
		t.Errorf("receiver did not deattach from session")
	}
	if rw.Header().Get("content-type") != "application/javascript; charset=UTF-8" {
		t.Errorf("Wrong content type received, got '%s'", rw.Header().Get("content-type"))
	}
}

func TestHandler_SessionTimeout(t *testing.T) {
	h := newTestHandler()
	h.options.DisconnectDelay = 10 * time.Millisecond
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	h.sessionByRequest(req)

	h.sessionsMux.Lock()
	if _, exists := h.sessions["session"]; !exists {
		t.Errorf("Session should exist in handler after timeout")
	}
	h.sessionsMux.Unlock()
	time.Sleep(15 * time.Millisecond)
	h.sessionsMux.Lock()
	if _, exists := h.sessions["session"]; exists {
		t.Errorf("Session should not exist in handler after timeout")
	}
	h.sessionsMux.Unlock()
}

type ClosableRecorder struct {
	*httptest.ResponseRecorder
	closeNotifCh chan bool
}

func (cr *ClosableRecorder) CloseNotify() <-chan bool { return cr.closeNotifCh }

func TestHandler_XhrPollConnectionInterrupted(t *testing.T) {
	rec := &testReceiver{nil, nil}
	h := &handler{
		sessions:       make(map[string]*session),
		newXhrReceiver: func(http.ResponseWriter, uint32) receiver { return rec },
	}
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	rw := &ClosableRecorder{httptest.NewRecorder(), make(chan bool)}
	go func() {
		close(rw.closeNotifCh)
	}()
	h.xhrPoll(rw, req)
	runtime.Gosched()
	h.sessionsMux.Lock()
	if len(h.sessions) != 0 {
		t.Errorf("session should be removed from handler in case of interrupted connection")
	}
	h.sessionsMux.Unlock()
}

func TestHandler_XhrPollAnotherConnectionExists(t *testing.T) {
	doneCh := make(chan bool)

	rec1 := &testReceiver{doneCh, nil}
	rec2 := &testReceiver{doneCh, nil}

	receivers := []receiver{rec1, rec2}

	var ll sync.Mutex
	h := &handler{
		sessions: make(map[string]*session),
		newXhrReceiver: func(http.ResponseWriter, uint32) receiver {
			ll.Lock()
			defer ll.Unlock()

			ret := receivers[0]
			receivers = receivers[1:]
			return ret
		},
	}
	// turn of timeoutes and heartbeats
	h.options.HeartbeatDelay = time.Hour
	h.options.DisconnectDelay = time.Hour

	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	go func() {
		rw := httptest.NewRecorder()
		h.xhrPoll(rw, req)
		if len(rec2.frames) != 1 || rec2.frames[0] != "c[2010,\"Another connection still open\"]" {
			t.Errorf("Incorrect close frame retrieved, got '%s'", rec2.frames[0])
		}
		close(doneCh)
	}()
	h.xhrPoll(rw, req)
	if len(rec1.frames) != 1 || rec1.frames[0] != "o" {
		t.Errorf("Missing or wrong open frame '%v'", rec1.frames)
	}

}

func newTestHandler() *handler {
	h := &handler{sessions: make(map[string]*session), newXhrReceiver: dummyXhreceiver}
	h.options.HeartbeatDelay = time.Hour
	h.options.DisconnectDelay = time.Hour
	return h
}

var dummyXhreceiver = func(http.ResponseWriter, uint32) receiver {
	rec := httptest.NewRecorder()
	return newXhrReceiver(rec, 10)
}
