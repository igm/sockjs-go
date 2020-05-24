package sockjs_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/igm/sockjs-go/v3/sockjs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type eventSourceStage struct {
	t                *testing.T
	handler          *sockjs.Handler
	server           *httptest.Server
	resp             *http.Response
	err              error
	session          sockjs.Session
	haveSession      chan struct{}
	receivedMessages chan string
}

func newEventSourceStage(t *testing.T) (*eventSourceStage, *eventSourceStage, *eventSourceStage) {
	stage := &eventSourceStage{
		t:                t,
		haveSession:      make(chan struct{}),
		receivedMessages: make(chan string, 1024),
	}
	return stage, stage, stage
}

func (s *eventSourceStage) a_new_sockjs_handler_is_created() *eventSourceStage {
	s.handler = sockjs.NewHandler("/prefix", sockjs.DefaultOptions, func(sess sockjs.Session) {
		s.session = sess
		close(s.haveSession)
		for {
			msg, err := sess.Recv()
			if err == sockjs.ErrSessionNotOpen {
				return
			}
			require.NoError(s.t, err)
			s.receivedMessages <- msg
		}
	})
	return s
}

func (s *eventSourceStage) a_server_is_started() *eventSourceStage {
	s.server = httptest.NewServer(s.handler)
	return s
}

func (s *eventSourceStage) a_sockjs_eventsource_connection_is_received() *eventSourceStage {
	s.resp, s.err = http.Get(s.server.URL + "/prefix/123/456/eventsource")
	return s
}

func (s *eventSourceStage) handler_is_invoked_with_session() *eventSourceStage {
	select {
	case <-s.haveSession:
	case <-time.After(1 * time.Second):
		s.t.Fatal("no session was created")
	}
	assert.Equal(s.t, sockjs.ReceiverTypeEventSource, s.session.ReceiverType())
	return s
}

func (s *eventSourceStage) session_is_closed() *eventSourceStage {
	s.session.Close(1024, "Close")
	assert.Error(s.t, s.session.Context().Err())
	select {
	case <-s.session.Context().Done():
	case <-time.After(1 * time.Second):
		s.t.Fatal("context should have been done")
	}
	return s
}

func (s *eventSourceStage) valid_eventsource_frames_should_be_received() *eventSourceStage {
	require.NoError(s.t, s.err)
	assert.Equal(s.t, "text/event-stream; charset=UTF-8", s.resp.Header.Get("content-type"))
	assert.Equal(s.t, "true", s.resp.Header.Get("access-control-allow-credentials"))
	assert.Equal(s.t, "*", s.resp.Header.Get("access-control-allow-origin"))

	all, err := ioutil.ReadAll(s.resp.Body)
	require.NoError(s.t, err)
	expectedBody := "\r\ndata: o\r\n\r\ndata: c[1024,\"Close\"]\r\n\r\n"
	assert.Equal(s.t, expectedBody, string(all))
	return s
}

func (s *eventSourceStage) a_message_is_sent_from_client(msg string) *eventSourceStage {
	out, err := json.Marshal([]string{msg})
	require.NoError(s.t, err)
	r, err := http.Post(s.server.URL+"/prefix/123/456/xhr_send", "application/json", bytes.NewReader(out))
	require.NoError(s.t, err)
	require.Equal(s.t, http.StatusNoContent, r.StatusCode)
	return s
}

func (s *eventSourceStage) same_message_should_be_received_from_session(expectredMsg string) *eventSourceStage {
	select {
	case msg := <-s.receivedMessages:
		assert.Equal(s.t, expectredMsg, msg)
	case <-time.After(1 * time.Second):
		s.t.Fatal("no message was received")
	}
	return s
}

func (s *eventSourceStage) and() *eventSourceStage { return s }

func (s *eventSourceStage) a_server_is_started_with_handler() *eventSourceStage {
	s.a_new_sockjs_handler_is_created()
	s.a_server_is_started()
	return s
}
