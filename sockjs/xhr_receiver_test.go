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
	var frame = "some frame content"
	recv.sendFrame(frame)
	if rec.Body.String() != frame+"\n" {
		t.Errorf("Incorrect body content received, got '%s', expected '%s'", rec.Body.String(), frame)
	}

}

func TestXhrReceiver_SendBulk(t *testing.T) {
	rec := httptest.NewRecorder()
	recv := newXhrReceiver(rec, 1024)
	recv.sendBulk("message 1", "message 2", "message 3")
	expected := "a[\"message 1\",\"message 2\",\"message 3\"]\n"
	if rec.Body.String() != expected {
		t.Errorf("Incorrect body content received from receiver, got '%s' expected '%s'", rec.Body.String(), expected)
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
		t.Errorf("Receiver should not be done yet")
	default: // ok
	}
	recv.sendBulk("message 1", "message 2") // produces another 27 bytes of response in 1 frame to go over max resposne size
	select {
	case <-recv.doneCh: // ok
	default:
		t.Errorf("Receiver closed channel did not close")
	}
}
