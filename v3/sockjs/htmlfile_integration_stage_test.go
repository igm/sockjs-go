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

type htmlFileStage struct {
	t                *testing.T
	handler          *sockjs.Handler
	server           *httptest.Server
	resp             *http.Response
	err              error
	session          sockjs.Session
	haveSession      chan struct{}
	receivedMessages chan string
}

func newHtmlFileStage(t *testing.T) (*htmlFileStage, *htmlFileStage, *htmlFileStage) {
	stage := &htmlFileStage{
		t:                t,
		haveSession:      make(chan struct{}),
		receivedMessages: make(chan string, 1024),
	}
	return stage, stage, stage
}

func (s *htmlFileStage) a_new_sockjs_handler_is_created() *htmlFileStage {
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

func (s *htmlFileStage) a_server_is_started() *htmlFileStage {
	s.server = httptest.NewServer(s.handler)
	return s
}

func (s *htmlFileStage) a_sockjs_htmlfile_connection_is_received() *htmlFileStage {
	s.resp, s.err = http.Get(s.server.URL + "/prefix/123/123/htmlfile?c=testCallback")
	return s
}

func (s *htmlFileStage) correct_http_response_should_be_received() *htmlFileStage {
	require.NoError(s.t, s.err)
	assert.Equal(s.t, http.StatusOK, s.resp.StatusCode)
	assert.Equal(s.t, "text/html; charset=UTF-8", s.resp.Header.Get("content-type"))
	assert.Equal(s.t, "true", s.resp.Header.Get("access-control-allow-credentials"))
	assert.Equal(s.t, "*", s.resp.Header.Get("access-control-allow-origin"))
	return s
}

func (s *htmlFileStage) handler_should_be_started_with_session() *htmlFileStage {
	select {
	case <-s.haveSession:
	case <-time.After(1 * time.Second):
		s.t.Fatal("no session was created")
	}
	assert.Equal(s.t, sockjs.ReceiverTypeHtmlFile, s.session.ReceiverType())
	return s
}

func (s *htmlFileStage) session_is_closed() *htmlFileStage {
	require.NoError(s.t, s.session.Close(1024, "Close"))
	assert.Error(s.t, s.session.Context().Err())
	select {
	case <-s.session.Context().Done():
	case <-time.After(1 * time.Second):
		s.t.Fatal("context should have been done")
	}
	return s
}

func (s *htmlFileStage) valid_htmlfile_response_should_be_received() *htmlFileStage {
	all, err := ioutil.ReadAll(s.resp.Body)
	require.NoError(s.t, err)
	assert.Contains(s.t, string(all), `p("o");`, string(all))
	assert.Contains(s.t, string(all), `p("c[1024,\"Close\"]")`, string(all))
	assert.Contains(s.t, string(all), `var c = parent.testCallback;`, string(all))
	return s
}

func (s *htmlFileStage) and() *htmlFileStage { return s }

func (s *htmlFileStage) a_server_is_started_with_handler() *htmlFileStage {
	s.a_new_sockjs_handler_is_created()
	s.a_server_is_started()
	return s
}

func (s *htmlFileStage) active_session_is_closed() *htmlFileStage {
	s.session_is_active()
	s.session_is_closed()
	return s
}

func (s *htmlFileStage) session_is_active() *htmlFileStage {
	s.a_sockjs_htmlfile_connection_is_received()
	s.handler_should_be_started_with_session()
	return s
}

func (s *htmlFileStage) a_message_is_sent_from_client(msg string) *htmlFileStage {
	out, err := json.Marshal([]string{msg})
	require.NoError(s.t, err)
	r, err := http.Post(s.server.URL+"/prefix/123/123/xhr_send", "application/json", bytes.NewReader(out))
	require.NoError(s.t, err)
	require.Equal(s.t, http.StatusNoContent, r.StatusCode)
	return s
}

func (s *htmlFileStage) same_message_should_be_received_from_session(expectredMsg string) *htmlFileStage {
	select {
	case msg := <-s.receivedMessages:
		assert.Equal(s.t, expectredMsg, msg)
	case <-time.After(1 * time.Second):
		s.t.Fatal("no message was received")
	}
	return s
}
