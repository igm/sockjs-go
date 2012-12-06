package sockjs

import (
	"bufio"
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
	var buf *bufio.ReadWriter
	conn, buf, err = hj.Hijack()
	if err != nil {
		return
	}
	buf.Flush()
	return
}

// watch net.Connection for EOF and notify via provided channel
func connectionClosedGuard(conn net.Conn, conn_interrupted chan<- bool) {
	_, err := conn.Read([]byte{1})
	if err != nil {
		conn_interrupted <- true
	}
}
