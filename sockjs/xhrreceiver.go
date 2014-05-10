package sockjs

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type xhrReceiverState int

const (
	stateXhrReceiverActive xhrReceiverState = iota
	stateXhrReceiverClosed
)

type xhrReceiver struct {
	sync.Mutex
	state xhrReceiverState

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

func (recv *xhrReceiver) sendBulk(messages ...string) {
	if len(messages) > 0 {
		recv.sendFrame(fmt.Sprintf("a[%s]",
			strings.Join(
				transform(messages, quote),
				",",
			),
		))
	}
}

func (recv *xhrReceiver) sendFrame(value string) {
	recv.Lock()
	defer recv.Unlock()

	if recv.state == stateXhrReceiverActive {
		fmt.Fprintf(recv.rw, "%s\n", value)
		recv.currentResponseSize += uint32(len(value) + 1)
		if recv.currentResponseSize >= recv.maxResponseSize {
			recv.state = stateXhrReceiverClosed
			close(recv.doneCh)
		} else {
			recv.rw.(http.Flusher).Flush()
		}
	}
}

func (recv *xhrReceiver) doneNotify() <-chan struct{}        { return recv.doneCh }
func (recv *xhrReceiver) interruptedNotify() <-chan struct{} { return recv.interruptCh }
func (recv *xhrReceiver) close() {
	recv.Lock()
	defer recv.Unlock()
	if recv.state < stateXhrReceiverClosed {
		recv.state = stateXhrReceiverClosed
		close(recv.doneCh)
	}
}
