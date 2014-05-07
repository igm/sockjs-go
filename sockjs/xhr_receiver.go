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
	doneCh              chan bool
}

func newXhrReceiver(rw http.ResponseWriter, maxResponse uint32) *xhrReceiver {
	return &xhrReceiver{
		rw:              rw,
		maxResponseSize: maxResponse,
		doneCh:          make(chan bool),
	}
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
	// n, _ := io.WriteString(recv.rw, value+"\n")
	fmt.Fprintf(recv.rw, "%s\n", value)
	recv.currentResponseSize += uint32(len(value) + 1)
	if recv.currentResponseSize >= recv.maxResponseSize {
		close(recv.doneCh)
	} else {
		recv.rw.(http.Flusher).Flush()
	}
}

func (recv *xhrReceiver) done() <-chan bool {
	return recv.doneCh
}
