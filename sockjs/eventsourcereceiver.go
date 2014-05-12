package sockjs

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type eventSourceReceiver struct {
	sync.Mutex
	state               xhrReceiverState
	rw                  http.ResponseWriter
	currentResponseSize uint32
	maxResponseSize     uint32
	doneCh              chan struct{}
	interruptCh         chan struct{}
}

func newEventSourceReceiver(rw http.ResponseWriter, maxResponse uint32) *eventSourceReceiver {
	recv := &eventSourceReceiver{
		rw:              rw,
		maxResponseSize: maxResponse,
		doneCh:          make(chan struct{}),
		interruptCh:     make(chan struct{}),
	}
	if closeNotifier, ok := rw.(http.CloseNotifier); ok {
		// if supported check for close notifications from http.RW
		go func() {
			select {
			case <-closeNotifier.CloseNotify():
				close(recv.interruptCh)
			case <-recv.doneCh:
				// ok, no action needed here, receiver closed in correct way
				// just finish the routine
			}
		}()
	}
	return recv
}

func (recv *eventSourceReceiver) sendBulk(messages ...string) {
	if len(messages) > 0 {
		recv.sendFrame(fmt.Sprintf("a[%s]",
			strings.Join(
				transform(messages, quote),
				",",
			),
		))
	}
}

func (recv *eventSourceReceiver) sendFrame(value string) {
	recv.Lock()
	defer recv.Unlock()

	if recv.state == stateXhrReceiverActive {
		n, _ := fmt.Fprintf(recv.rw, "data: %s\r\n\r\n", value)
		recv.currentResponseSize += uint32(n)
		if recv.currentResponseSize >= recv.maxResponseSize {
			recv.state = stateXhrReceiverClosed
			close(recv.doneCh)
		} else {
			recv.rw.(http.Flusher).Flush()
		}
	}
}

func (recv *eventSourceReceiver) doneNotify() <-chan struct{}        { return recv.doneCh }
func (recv *eventSourceReceiver) interruptedNotify() <-chan struct{} { return recv.interruptCh }
func (recv *eventSourceReceiver) close() {
	recv.Lock()
	defer recv.Unlock()
	if recv.state < stateXhrReceiverClosed {
		recv.state = stateXhrReceiverClosed
		close(recv.doneCh)
	}
}
