package sockjs

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type xhrReceiver struct {
	rw                  http.ResponseWriter
	maxResponseSize     uint32
	currentResponseSize uint32
	closedNotifCh       chan interface{}
}

func newXhrReceiver(rw http.ResponseWriter, maxResponse uint32) *xhrReceiver {
	return &xhrReceiver{
		rw:              rw,
		maxResponseSize: maxResponse,
		closedNotifCh:   make(chan interface{}),
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
	n, _ := io.WriteString(recv.rw, value+"\n")
	recv.currentResponseSize += uint32(n)
	if recv.currentResponseSize >= recv.maxResponseSize {
		close(recv.closedNotifCh)
	} else {
		recv.rw.(http.Flusher).Flush()
	}
}

func (recv *xhrReceiver) done() chan interface{} {
	return recv.closedNotifCh
}
