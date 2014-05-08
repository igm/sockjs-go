package sockjs

import (
	"fmt"
	"net/http"
	"strings"
)

type xhrReceiver struct {
	rw                  http.ResponseWriter
	maxResponseSize     uint32
	currentResponseSize uint32
	doneCh              chan struct{}
	interruptCh         chan struct{}
}

func newXhrReceiver(rw http.ResponseWriter, maxResponse uint32) *xhrReceiver {
	recv := &xhrReceiver{
		rw:              rw,
		maxResponseSize: maxResponse,
		doneCh:          make(chan struct{}),
		interruptCh:     make(chan struct{}),
	}
	if closeNotifier, ok := rw.(http.CloseNotifier); ok {
		go func() {
			select {
			case <-closeNotifier.CloseNotify():
				close(recv.interruptCh)
			case <-recv.doneCh: // ok, no action needed here
			}
		}()
	}
	return recv
}

func (recv *xhrReceiver) sendBulk(messages ...string) {
	for i, msg := range messages {
		messages[i] = quote(msg)
	}
	if len(messages) > 0 {
		recv.sendFrame(fmt.Sprintf("a[%s]", strings.Join(messages, ",")))
	}
}

func (recv *xhrReceiver) sendFrame(value string) {
	select {
	case <-recv.doneCh:
		return
	default:
	}
	fmt.Fprintf(recv.rw, "%s\n", value)
	recv.currentResponseSize += uint32(len(value) + 1)
	if recv.currentResponseSize >= recv.maxResponseSize {
		recv.close()
	} else {
		recv.rw.(http.Flusher).Flush()
	}
}

func (recv *xhrReceiver) doneNotify() <-chan struct{}        { return recv.doneCh }
func (recv *xhrReceiver) interruptedNotify() <-chan struct{} { return recv.interruptCh }
func (recv *xhrReceiver) close() {
	select {
	case <-recv.doneCh:
	default:
		close(recv.doneCh)
	}
}
