package sockjs

import (
	"errors"
	"net"
	"net/http"
)

// hijack connection
func hijack(rw http.ResponseWriter) (conn net.Conn, err error) {
	hj, ok := rw.(http.Hijacker)
	if !ok {
		err = errors.New("webserver doesn't support hijacking")
		return
	}
	conn, _, err = hj.Hijack()
	if err != nil {
		return
	}
	return
}

// watch net.Connection for EOF and notify via provided channel
func connectionClosedGuard(conn net.Conn, conn_interrupted chan<- bool) {
	_, err := conn.Read([]byte{1})
	if err != nil {
		select {
		case conn_interrupted <- true:
		default:
		}
	}
}
