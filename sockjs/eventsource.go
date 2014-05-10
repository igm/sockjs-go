package sockjs

import (
	"fmt"
	"net/http"
)

func (h *handler) eventSource(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("content-type", "text/event-stream; charset=UTF-8")
	fmt.Fprintf(rw, "\r\n")
	rw.(http.Flusher).Flush()
}
