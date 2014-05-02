package sockjs

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCreateHandler(t *testing.T) {
	handler := NewHandler("/echo", DefaultOptions, nil)
	if handler.Prefix() != "/echo" {
		t.Errorf("Prefix not properly set, got '%s' expected '%s'", handler.Prefix(), "/echo")
	}
	if handler.sessions == nil {
		t.Errorf("Handler session map not made")
	}
	// TODO(igm) add more handler *unit* tests
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/echo")
	if err != nil {
		t.Errorf("There should not be any error, got '%s'", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code receiver, got '%d' expected '%d'", resp.StatusCode, http.StatusOK)
	}
}

func TestParseSessionId(t *testing.T) {
	h := handler{prefix: "/prefix"}
	url, _ := url.Parse("http://server:port/prefix/server/session/whatever")
	if session, err := h.parseSessionID(url); session != "session" || err != nil {
		t.Errorf("Wrong session parsed, got '%s' expected '%s' with error = '%v'", session, "session", err)
	}
	url, _ = url.Parse("http://server:port/asdasd/server/session/whatever")
	if _, err := h.parseSessionID(url); err == nil {
		t.Errorf("Should return error")
	}
}

func TestHandlerCreateReceivers(t *testing.T) {
	handler := NewHandler("/echo", DefaultOptions, nil)
	if handler.newXhrReceiver(nil, 10) == nil {
		t.Errorf("Receiver not created properly, got 'nil'")
	}
}
