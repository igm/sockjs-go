package sockjs

import (
	"net"
	"net/http"
)

// http requests (hijacked)
type clientRequest struct {
	conn net.Conn
	req  *http.Request
}

// xhr-streaming specific connection with request channel
type xhrStreamConn struct {
	baseConn
	requests chan clientRequest
}

// state function type definition (for xhr connection states)
type xhrConnectionState func(*xhrStreamConn) xhrConnectionState

// run the state machine
func (this *xhrStreamConn) run(ctx *context, sessId string, initState xhrConnectionState) {
	for state := initState; state != nil; {
		state = state(this)
	}
	ctx.delete(sessId)
}
