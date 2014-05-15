package sockjs

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_jsonpNoCallback(t *testing.T) {
	h := newTestHandler()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/server/session/jsonp", nil)
	h.jsonp(rw, req)
	if rw.Code != http.StatusInternalServerError {
		t.Errorf("Unexpected response code, got '%d', expected '%d'", rw.Code, http.StatusInternalServerError)
	}
	expectedContentType := "text/plain; charset=utf-8"
	if rw.Header().Get("content-type") != expectedContentType {
		t.Errorf("Unexpected content type, got '%s', expected '%s'", rw.Header().Get("content-type"), expectedContentType)
	}
}

func TestHandler_jsonp(t *testing.T) {
	h := newTestHandler()
	rw := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/server/session/jsonp?c=testCallback", nil)
	h.jsonp(rw, req)
	expectedContentType := "application/javascript; charset=UTF-8"
	if rw.Header().Get("content-type") != expectedContentType {
		t.Errorf("Unexpected content type, got '%s', expected '%s'", rw.Header().Get("content-type"), expectedContentType)
	}
	expectedBody := "testCallback(\"o\");\r\n"
	if rw.Body.String() != expectedBody {
		t.Errorf("Unexpected body, got '%s', expected '%s'", rw.Body, expectedBody)
	}
}
