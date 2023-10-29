package sockjs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testOptions = DefaultOptions

func init() {
	testOptions.RawWebsocket = true
}

func TestHandler_Create(t *testing.T) {
	handler := NewHandler("/echo", testOptions, nil)
	if handler.Prefix() != "/echo" {
		t.Errorf("Prefix not properly set, got '%s' expected '%s'", handler.Prefix(), "/echo")
	}
	if handler.sessions == nil {
		t.Errorf("Handler session map not made")
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/echo")
	if err != nil {
		t.Errorf("There should not be any error, got '%s'", err)
		t.FailNow()
	}
	if resp == nil {
		t.Errorf("Response should not be nil")
		t.FailNow()
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code receiver, got '%d' expected '%d'", resp.StatusCode, http.StatusOK)
	}
}

func TestHandler_RootPrefixInfoHandler(t *testing.T) {
	infoOptions := testOptions
	jSessionCalled := false
	infoOptions.JSessionID = func(writer http.ResponseWriter, request *http.Request) {
		jSessionCalled = true
	}
	handler := NewHandler("", infoOptions, nil)
	if handler.Prefix() != "" {
		t.Errorf("Prefix not properly set, got '%s' expected '%s'", handler.Prefix(), "")
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/info")
	if err != nil {
		t.Errorf("There should not be any error, got '%s'", err)
		t.FailNow()
	}
	if resp == nil {
		t.Errorf("Response should not be nil")
		t.FailNow()
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code receiver, got '%d' expected '%d'", resp.StatusCode, http.StatusOK)
		t.FailNow()
	}
	assert.True(t, jSessionCalled)
	infoData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading body: '%v'", err)
	}
	var i info
	err = json.Unmarshal(infoData, &i)
	if err != nil {
		t.Fatalf("Error unmarshaling info: '%v', data was: '%s'", err, string(infoData))
	}
	if i.Websocket != true {
		t.Fatalf("Expected websocket to be true")
	}
}

func TestHandler_ParseSessionId(t *testing.T) {
	h := Handler{prefix: "/prefix"}
	url, _ := url.Parse("http://server:80/server/session/whatever")
	if session, err := h.parseSessionID(url); session != "session" || err != nil {
		t.Errorf("Wrong session parsed, got '%s' expected '%s' with error = '%v'", session, "session", err)
	}
}

func TestHandler_SessionByRequest(t *testing.T) {
	h := NewHandler("", testOptions, nil)
	h.options.DisconnectDelay = 10 * time.Millisecond
	var handlerFuncCalled = make(chan Session)
	h.handlerFunc = func(s Session) { handlerFuncCalled <- s }
	req, _ := http.NewRequest("POST", "/server/sessionid/whatever/follows", nil)
	sess, err := h.sessionByRequest(req)
	if sess == nil || err != nil {
		t.Errorf("session should be returned")
		// test handlerFunc was called
		select {
		case s := <-handlerFuncCalled: // ok
			if s.session != sess {
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
	h.sessionsMux.Lock()
	if _, exists := h.sessions["sessionid"]; exists {
		t.Errorf("session should not exist in handler after timeout")
	}
	h.sessionsMux.Unlock()
	// test proper behaviour in case URL is not correct
	req, _ = http.NewRequest("POST", "", nil)
	if _, err := h.sessionByRequest(req); err == nil {
		t.Errorf("Expected parser sessionID from URL error, got 'nil'")
	}
}

func TestHandler_StrictPathMatching(t *testing.T) {
	handler := NewHandler("/echo", testOptions, nil)
	server := httptest.NewServer(handler)
	defer server.Close()

	cases := []struct {
		url            string
		expectedStatus int
	}{
		{url: "/echo", expectedStatus: http.StatusOK},
		{url: "/test/echo", expectedStatus: http.StatusNotFound},
		{url: "/echo/test/test/echo", expectedStatus: http.StatusNotFound},
	}

	for _, urlCase := range cases {
		t.Run(urlCase.url, func(t *testing.T) {
			resp, err := http.Get(server.URL + urlCase.url)
			require.NoError(t, err)
			assert.Equal(t, urlCase.expectedStatus, resp.StatusCode)
		})
	}
}

func TestHandler_DisabledMethods(t *testing.T) {
	disabledMethodsOptions := testOptions
	disabledMethodsOptions.DisableXHR = true
	disabledMethodsOptions.DisableJSONP = true
	disabledMethodsOptions.DisableHtmlFile = true
	disabledMethodsOptions.DisableEventSource = true
	disabledMethodsOptions.DisableXHRStreaming = true

	handler := NewHandler("", disabledMethodsOptions, nil)
	server := httptest.NewServer(handler)
	defer server.Close()

	fmt.Println(server.URL)

	cases := []struct {
		URL            string
		expectedStatus int
		HTTPMethod     string
	}{

		{
			URL:            "/",
			expectedStatus: http.StatusOK,
			HTTPMethod:     "GET",
		},
		// XHR
		{
			URL:            "/server/session/xhr",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "POST",
		},
		{
			URL:            "/server/session/xhr_streaming",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "POST",
		},
		{
			URL:            "/server/session/xhr_send",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "POST",
		},

		// EventStream
		{
			URL:            "/server/session/eventsource",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "GET",
		},

		//Htmlfile
		{
			URL:            "/server/session/htmlfile",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "GET",
		},

		// JSONP
		{
			URL:            "/server/session/jsonp",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "GET",
		},
		{
			URL:            "/server/session/jsonp_send",
			expectedStatus: http.StatusNotFound,
			HTTPMethod:     "POST",
		},
	}

	for _, urlCase := range cases {
		t.Run(urlCase.URL, func(t *testing.T) {

			req, err := http.NewRequest(urlCase.HTTPMethod, server.URL+urlCase.URL, nil)
			require.NoError(t, err)
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			fmt.Printf("URL: %v, Expected: %v, Result: %v\n", urlCase.URL, urlCase.expectedStatus, resp.StatusCode)
			assert.Equal(t, urlCase.expectedStatus, resp.StatusCode)
		})
	}
}
