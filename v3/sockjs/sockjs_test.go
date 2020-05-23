package sockjs

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSockJS_ServeHTTP(t *testing.T) {
	h := NewHandler("", DefaultOptions, func(s Session) {
		_ = s.Close(3000, "")
	})
	mux := http.NewServeMux()
	mux.Handle("/", h)

	server := httptest.NewServer(mux)
	req, _ := http.NewRequest("GET", server.URL+"/foo/bar", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
	req, _ = http.NewRequest("GET", server.URL+"/bar", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
	req, _ = http.NewRequest("GET", server.URL+"/", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusOK)
	}
}

func TestSockJS_ServeHTTP_Prefix(t *testing.T) {
	h := NewHandler("/connection/sockjs", DefaultOptions, func(s Session) {
		_ = s.Close(3000, "")
	})

	mux := http.NewServeMux()
	mux.Handle("/connection/sockjs/", h)

	server := httptest.NewServer(mux)
	req, _ := http.NewRequest("GET", server.URL+"/connection/sockjs/foo/bar", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
	req, _ = http.NewRequest("GET", server.URL+"/connection/sockjs/bar", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusNotFound)
	}
	req, _ = http.NewRequest("GET", server.URL+"/connection/sockjs/", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("Unexpected response status, got '%d' expected '%d'", rec.Code, http.StatusOK)
	}
}
