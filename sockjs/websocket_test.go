package sockjs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

func TestHandler_WebSocket(t *testing.T) {
	h := newTestHandler()
	server := httptest.NewServer(http.HandlerFunc(h.sockjs_websocket))
	defer server.Close()
	url := "ws" + server.URL[4:]
	header := http.Header{}
	conn, resp, err := websocket.DefaultDialer.Dial(url, header)
	if err != websocket.ErrBadHandshake {
		t.Errorf("Expected error '%v', got '%v'", websocket.ErrBadHandshake, err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Unexpected response code, got '%d', expected '%d'", resp.StatusCode, http.StatusForbidden)
	}
	if conn != nil {
		t.Errorf("Connection should be nil, got '%v'", conn)
	}
	header.Set("origin", server.URL)
	_, resp, err = websocket.DefaultDialer.Dial(url, header)
	fmt.Println(conn, resp, err)
}
