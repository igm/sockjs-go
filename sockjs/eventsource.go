package sockjs

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (h *handler) eventSource(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("content-type", "text/event-stream; charset=UTF-8")
	fmt.Fprintf(rw, "\r\n")
	rw.(http.Flusher).Flush()

	recv := newHTTPReceiver(rw, h.options.ResponseLimit, new(eventSourceFrameWriter))
	sess, _ := h.sessionByRequest(req)
	if err := sess.attachReceiver(recv); err != nil {
		recv.sendFrame(cFrame)
		recv.close()
		return
	}

	select {
	case <-recv.doneNotify():
	case <-recv.interruptedNotify():
	}
}

type eventSourceFrameWriter struct{}

var escaper *strings.Replacer = strings.NewReplacer(
	"%", url.QueryEscape("%"),
	"\n", url.QueryEscape("\n"),
	"\r", url.QueryEscape("\r"),
	"\x00", url.QueryEscape("\x00"),
)

func (*eventSourceFrameWriter) write(w io.Writer, frame string) (int, error) {
	return fmt.Fprintf(w, "data: %s\r\n\r\n", escaper.Replace(frame))
}
