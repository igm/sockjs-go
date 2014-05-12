package sockjs

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_EventSource(t *testing.T) {
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/server/session/eventsource", nil)
	h := newTestHandler()
	go func() {
		h.sessions["session"].recv.close()
	}()
	h.eventSource(rw, req)
	contentType := rw.Header().Get("content-type")
	expected := "text/event-stream; charset=UTF-8"
	if contentType != expected {
		t.Errorf("Unexpected content type, got '%s', extected '%s'", contentType, expected)
	}
	if rw.Code != http.StatusOK {
		t.Errorf("Unexpected response code, got '%d', expected '%d'", rw.Code, http.StatusOK)
	}

	if rw.Body.String() != "\r\ndata: o\r\n\r\n" {
		t.Errorf("Event stream prelude, got '%s'", rw.Body)
	}
}
