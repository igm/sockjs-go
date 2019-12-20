package sockjs_go

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestSockJS_ServeHTTP(t *testing.T) {
	return
	//TODO fix this case
	m := handler{mappings: make([]*mapping, 0)}
	m.mappings = []*mapping{
		&mapping{"POST", regexp.MustCompile("/foo/.*"), []http.HandlerFunc{func(http.ResponseWriter, *http.Request) {}}},
	}
	req, _ := http.NewRequest("GET", "/foo/bar", nil)
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusMethodNotAllowed)
	}
	req, _ = http.NewRequest("GET", "/bar", nil)
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
}
