package sockjs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

func TestHandler_XhrSendNilBody(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/non_existing_session/xhr_send", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "non_existing_session"})
	h.xhrSend(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusBadRequest)
	}
	if rec.Body.String() != "Payload expected." {
		t.Errorf("Unexcpected body received: '%s'", rec.Body.String())
	}
}

func TestHandler_XhrSendEmptyBody(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/non_existing_session/xhr_send", strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"session": "non_existing_session"})
	h.xhrSend(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusBadRequest)
	}
	if rec.Body.String() != "Payload expected." {
		t.Errorf("Unexcpected body received: '%s'", rec.Body.String())
	}
}

func TestHandler_XhrSendWrongUrlPath(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "incorrect", strings.NewReader("[\"a\"]"))
	h.xhrSend(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("Unexcpected response status, got '%d', expected '%d'", rec.Code, http.StatusBadRequest)
	}
}

func TestHandler_XhrSendToExistingSession(t *testing.T) {
	h := newTestHandler()
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"some message\"]"))
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	sess := newSession(req, "session", time.Second, time.Second)
	h.sessions["session"] = sess

	req, _ = http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"some message\"]"))
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
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
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	rec := httptest.NewRecorder()
	h.xhrSend(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != "Broken JSON encoding." {
		t.Errorf("Unexpected response, got '%d,%s' expected '%d,Broken JSON encoding.'", rec.Code, rec.Body.String(), http.StatusBadRequest)
	}

	// unexpected EOF
	req, _ = http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"x"))
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	rec = httptest.NewRecorder()
	h.xhrSend(rec, req)
	if rec.Code != http.StatusBadRequest || rec.Body.String() != "Broken JSON encoding." {
		t.Errorf("Unexpected response, got '%d,%s' expected '%d,Broken JSON encoding.'", rec.Code, rec.Body.String(), http.StatusBadRequest)
	}
}

func TestHandler_XhrSendSessionNotFound(t *testing.T) {
	h := Handler{}
	req, _ := http.NewRequest("POST", "/server/session/xhr_send", strings.NewReader("[\"some message\"]"))
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	rec := httptest.NewRecorder()
	h.xhrSend(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
}

func TestHandler_XhrPoll(t *testing.T) {
	h := newTestHandler()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	h.xhrPoll(rw, req)
	if rw.Header().Get("content-type") != "application/javascript; charset=UTF-8" {
		t.Errorf("Wrong content type received, got '%s'", rw.Header().Get("content-type"))
	}
	sess, _ := h.sessionByRequest(req)
	if rt := sess.ReceiverType(); rt != ReceiverTypeXHR {
		t.Errorf("Unexpected recevier type, got '%v', extected '%v'", rt, ReceiverTypeXHR)
	}
}

func TestHandler_XhrPollConnectionInterrupted(t *testing.T) {
	h := newTestHandler()
	sess := newTestSession()
	sess.state = SessionActive
	h.sessions["session"] = sess
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rw := httptest.NewRecorder()
	cancel()
	h.xhrPoll(rw, req)
	time.Sleep(1 * time.Millisecond)
	sess.mux.Lock()
	if sess.state != SessionClosed {
		t.Errorf("session should be closed")
	}
}

func TestHandler_XhrPollAnotherConnectionExists(t *testing.T) {
	h := newTestHandler()
	req, _ := http.NewRequest("POST", "/server/session/xhr", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	// turn of timeoutes and heartbeats
	sess := newSession(req, "session", time.Hour, time.Hour)
	h.sessions["session"] = sess
	noError(t, sess.attachReceiver(newTestReceiver()))
	req, _ = http.NewRequest("POST", "/server/session/xhr", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	rw2 := httptest.NewRecorder()
	h.xhrPoll(rw2, req)
	if rw2.Body.String() != "c[2010,\"Another connection still open\"]\n" {
		t.Errorf("Unexpected body, got '%s'", rw2.Body)
	}
}

func TestHandler_XhrStreaming(t *testing.T) {
	h := newTestHandler()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_streaming", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	go func() {
		var sess *session
		for exists := false; !exists; {
			runtime.Gosched()
			h.sessionsMux.Lock()
			sess, exists = h.sessions["session"]
			h.sessionsMux.Unlock()
		}
		for exists := false; !exists; {
			runtime.Gosched()
			sess.mux.RLock()
			exists = sess.recv != nil
			sess.mux.RUnlock()
		}
		if rt := sess.ReceiverType(); rt != ReceiverTypeXHRStreaming {
			t.Errorf("Unexpected recevier type, got '%v', extected '%v'", rt, ReceiverTypeXHRStreaming)
		}
		sess.mux.RLock()
		sess.recv.close()
		sess.mux.RUnlock()
	}()
	h.xhrStreaming(rw, req)
	expectedBody := strings.Repeat("h", 2048) + "\no\n"
	if rw.Body.String() != expectedBody {
		t.Errorf("Unexpected body, got '%s' expected '%s'", rw.Body, expectedBody)
	}
	sess, _ := h.sessionByRequest(req)
	if rt := sess.ReceiverType(); rt != ReceiverTypeXHRStreaming {
		t.Errorf("Unexpected recevier type, got '%v', extected '%v'", rt, ReceiverTypeXHRStreaming)
	}
}

func TestHandler_XhrStreamingAnotherReceiver(t *testing.T) {
	h := newTestHandler()
	h.options.ResponseLimit = 4096
	rw1 := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/xhr_streaming", nil)
	req = mux.SetURLVars(req, map[string]string{"session": "session"})
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	go func() {
		rec := httptest.NewRecorder()
		h.xhrStreaming(rec, req)
		expectedBody := strings.Repeat("h", 2048) + "\n" + "c[2010,\"Another connection still open\"]\n"
		if rec.Body.String() != expectedBody {
			t.Errorf("Unexpected body got '%s', expected '%s', ", rec.Body, expectedBody)
		}
		cancel()
	}()
	h.xhrStreaming(rw1, req)
}

// various test only structs
func newTestHandler() *Handler {
	h := NewHandler(DefaultOptions, func(s *session) {})
	h.options.HeartbeatDelay = time.Hour
	h.options.DisconnectDelay = time.Hour
	return h
}
