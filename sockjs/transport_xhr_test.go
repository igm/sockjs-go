package sockjs

import (
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestHandler_XHrSendWrongUrlPath(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "incorrect", strings.NewReader("[\"a\"]"))
	h.xhrSend(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Unexcpected response status, got '%d', expected '%d'", rec.Code, http.StatusInternalServerError)
	}
}

func TestHandler_XhrSendToExistingSession(t *testing.T) {
	h := newTestHandler()
	sess := newSession(time.Second, time.Second)
	h.sessions["session"] = sess

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"some message\"]"))
	var done = make(chan bool)
	go func() {
		h.xhrSend(rec, req)
		done <- true
	}()
	msg, _ := sess.Recv()
	if msg != "some message" {
		t.Errorf("Incorrect message in the channel, should be '%s', was '%s'", "some message", msg)
	}
	<-done
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

func TestHandler_SessionByRequest(t *testing.T) {
	h := newTestHandler()
	h.options.DisconnectDelay = 10 * time.Millisecond
	var handlerFuncCalled = make(chan Conn)
	h.handlerFunc = func(conn Conn) { handlerFuncCalled <- conn }
	req, _ := http.NewRequest("POST", "/server/sessionid/whatever/follows", nil)
	sess, err := h.sessionByRequest(req)
	if sess == nil || err != nil {
		t.Errorf("Session should be returned")
		// test handlerFunc was called
		select {
		case conn := <-handlerFuncCalled: // ok
			if conn != sess {
				t.Errorf("Handler was not passed correct session")
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("HandlerFunc was not called")
		}
	}
	// test session is reused for multiple requests with same sessionID
	req2, _ := http.NewRequest("POST", "/server/sessionid/whatever", nil)
	if sess2, err := h.sessionByRequest(req2); sess2 != sess || err != nil {
		t.Errorf("Expected error, got session: '%v'", sess)
	}
	// test session expires after timeout
	time.Sleep(15 * time.Millisecond)
	if _, exists := h.sessions["sessionid"]; exists {
		t.Errorf("Session should not exist in handler after timeout")
	}
	// test proper behaviour in case URL is not correct
	req, _ = http.NewRequest("POST", "", nil)
	if _, err := h.sessionByRequest(req); err == nil {
		t.Errorf("Expected parser sessionID from URL error, got 'nil'")
	}
}

func TestHandler_XhrPoll(t *testing.T) {
	h := newTestHandler()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	h.xhrPoll(rw, req)
	if rw.Header().Get("content-type") != "application/javascript; charset=UTF-8" {
		t.Errorf("Wrong content type received, got '%s'", rw.Header().Get("content-type"))
	}
}

func TestHandler_XhrPollConnectionInterrupted(t *testing.T) {
	h := newTestHandler()
	sess := newTestSession()
	sess.state = sessionActive
	h.sessions["session"] = sess
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	rw := newClosableRecorder()
	close(rw.closeNotifCh)
	h.xhrPoll(rw, req)
	if sess.state != sessionClosed {
		t.Errorf("Session should be closed")
	}
}

func TestHandler_XhrPollAnotherConnectionExists(t *testing.T) {
	h := newTestHandler()
	// turn of timeoutes and heartbeats
	sess := newSession(time.Hour, time.Hour)
	h.sessions["session"] = sess
	sess.attachReceiver(&testReceiver{})
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	rw2 := httptest.NewRecorder()
	h.xhrPoll(rw2, req)
	if rw2.Body.String() != "c[2010,\"Another connection still open\"]\n" {
		t.Errorf("Unexpected body, got '%s'", rw2.Body)
	}
}

func TestHandler_XhrStreaming(t *testing.T) {
	h := newTestHandler()
	rw := newClosableRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_streaming", nil)
	h.xhrStreaming(rw, req)
	expected_body := strings.Repeat("h", 2048) + "\no\n"
	if rw.Body.String() != expected_body {
		t.Errorf("Unexpected body, got '%s' expected '%s'", rw.Body, expected_body)
	}
}

func TestHandler_XhrStreamingAnotherReceiver(t *testing.T) {
	h := newTestHandler()
	h.options.ResponseLimit = 4096
	rw1 := newClosableRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_streaming", nil)
	go func() {
		rec := httptest.NewRecorder()
		h.xhrStreaming(rec, req)
		expected_body := strings.Repeat("h", 2048) + "\n" + "c[2010,\"Another connection still open\"]\n"
		if rec.Body.String() != expected_body {
			t.Errorf("Unexpected body got '%s', expected '%s', ", rec.Body, expected_body)
		}
		close(rw1.closeNotifCh)
	}()
	h.xhrStreaming(rw1, req)
}

// various test only structs
func newTestHandler() *handler {
	h := &handler{sessions: make(map[string]*session)}
	h.options.HeartbeatDelay = time.Hour
	h.options.DisconnectDelay = time.Hour
	return h
}

type testReceiver struct {
	doneCh chan bool
	frames []string
}

func (t *testReceiver) done() <-chan bool           { return t.doneCh }
func (t *testReceiver) sendBulk(messages ...string) {}
func (t *testReceiver) sendFrame(frame string)      { t.frames = append(t.frames, frame) }

type ClosableRecorder struct {
	*httptest.ResponseRecorder
	closeNotifCh chan bool
}

func newClosableRecorder() *ClosableRecorder {
	return &ClosableRecorder{httptest.NewRecorder(), make(chan bool)}
}

func (cr *ClosableRecorder) CloseNotify() <-chan bool { return cr.closeNotifCh }
