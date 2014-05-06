package sockjs

import (
	"net/http/httptest"
	"testing"
)

func TestXhrReceiver_Create(t *testing.T) {
	rec := httptest.NewRecorder()
	recv := newXhrReceiver(rec, 1024)
	if recv.doneCh != recv.done() {
		t.Errorf("Calling done() must return close channel, but it does not")
	}
	if recv.rw != rec {
		t.Errorf("Http.ResponseWriter not properly initialized")
	}
	if recv.maxResponseSize != 1024 {
		t.Errorf("MaxResponseSize not properly initialized")
	}
}

func TestXhrReceiver_SendEmptyFrames(t *testing.T) {
	rec := httptest.NewRecorder()
	recv := newXhrReceiver(rec, 1024)
	recv.sendBulk()
	if rec.Body.String() != "" {
		t.Errorf("Incorrect body content received from receiver '%s'", rec.Body.String())
	}
}

func TestXhrReceiver_SendFrame(t *testing.T) {
	rec := httptest.NewRecorder()
	recv := newXhrReceiver(rec, 1024)
	recv.sendFrame("some frame content")
	if rec.Body.String() != "some frame content\n" {
		t.Errorf("Incorrent body content received, go '%s'", rec.Body.String())
	}

}

func TestXhrReceiver_SendBulk(t *testing.T) {
	rec := httptest.NewRecorder()
	recv := newXhrReceiver(rec, 1024)
	recv.sendBulk("message 1", "message 2", "message 3")
	if rec.Body.String() != "a[\"message 1\",\"message 2\",\"message 3\"]\n" {
		t.Errorf("Incorrect body content received from receiver '%s'", rec.Body.String())
	}
}

func TestXhrReceiver_MaximumResponseSize(t *testing.T) {
	rec := httptest.NewRecorder()
	recv := newXhrReceiver(rec, 54)
	recv.sendBulk("message 1", "message 2") // produces 27 bytes of response in 1 frame
	if recv.currentResponseSize != 27 {
		t.Errorf("Incorrect response size calcualated, got '%d' expected '%d'", recv.currentResponseSize, 27)
	}
	select {
	case <-recv.doneCh:
	default: // ok
	}
	recv.sendBulk("message 1", "message 2") // produces another 27 bytes of response in 1 frame to go over max resposne size
	select {
	case <-recv.doneCh: // ok
	default:
		t.Errorf("Receiver closed channel did not close")
	}
}
