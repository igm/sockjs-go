package sockjs_test

import (
	"testing"
)

func TestEventSource(t *testing.T) {
	given, when, then := newEventSourceStage(t)

	given.
		a_new_sockjs_handler_is_created().and().
		a_server_is_started()

	when.
		a_sockjs_eventsource_connection_is_received().and().
		handler_is_invoked_with_session().and().
		session_is_closed()

	then.
		valid_eventsource_frames_should_be_received()
}

func TestEventSourceMessageInteraction(t *testing.T) {
	given, when, then := newEventSourceStage(t)

	given.
		a_server_is_started_with_handler().
		a_sockjs_eventsource_connection_is_received().
		handler_is_invoked_with_session()

	when.
		a_message_is_sent_from_client("Hello World!").and().
		session_is_closed()

	then.
		same_message_should_be_received_from_session("Hello World!")
}
