package sockjs_test

import (
	"testing"
)

func TestHtmlFile_StartHandler(t *testing.T) {
	given, when, then := newHtmlFileStage(t)

	given.
		a_new_sockjs_handler_is_created().and().
		a_server_is_started()

	when.
		a_sockjs_htmlfile_connection_is_received()

	then.
		correct_http_response_should_be_received().and().
		handler_should_be_started_with_session()
}

func TestHtmlFile_CloseSession(t *testing.T) {
	given, when, then := newHtmlFileStage(t)

	given.
		a_server_is_started_with_handler()

	when.
		active_session_is_closed()

	then.
		valid_htmlfile_response_should_be_received()
}

func TestHtmlFile_SendMessage(t *testing.T) {
	given, when, then := newHtmlFileStage(t)

	given.
		a_server_is_started_with_handler()

	when.
		session_is_active().and().
		a_message_is_sent_from_client("Hello World!").and().
		active_session_is_closed()

	then.
		same_message_should_be_received_from_session("Hello World!")
}
