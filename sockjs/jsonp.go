package sockjs

import (
	"bytes"
	"code.google.com/p/gorilla/mux"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

//jsonp specific connection
type jsonpConn struct {
	baseConn
	requests chan jsonpRequest
}

type jsonpRequest struct {
	rw       http.ResponseWriter
	req      *http.Request
	done     chan bool
	callback string
}

// state function type definition (for xhr connection states)
type jsonpConnectionState func(*jsonpConn) jsonpConnectionState

// run the state machine
func (this *jsonpConn) run(ctx *context, sessId string, initState jsonpConnectionState) {
	for state := initState; state != nil; {
		state = state(this)
	}
	ctx.delete(sessId)
}

func (this *context) JsonpHandler(rw http.ResponseWriter, req *http.Request) {
	sessId := mux.Vars(req)["sessionid"]
	err := req.ParseForm()
	if err != nil {
		http.Error(rw, "Bad query", http.StatusInternalServerError)
		return
	}
	callback := req.Form.Get("c")
	if callback == "" {
		http.Error(rw, `"callback" parameter required`, http.StatusInternalServerError)
		return
	}

	conn, exists := this.getOrCreate(sessId, func() conn {
		conn := &jsonpConn{
			baseConn: newBaseConn(this),
			requests: make(chan jsonpRequest),
		}
		return conn
	})

	jsonp_conn := conn.(*jsonpConn)
	if !exists {
		go jsonp_conn.run(this, sessId, JsonpNewConnection)
		go this.HandlerFunc(jsonp_conn)
	}

	done := make(chan bool)
	// go func() {
	jsonp_conn.requests <- jsonpRequest{
		rw:       rw,
		req:      req,
		done:     done,
		callback: callback,
	}
	// }()
	<-done
}

/**************************************************************************************************/
/********** Jsonp state functions *****************************************************************/
/**************************************************************************************************/
func JsonpNewConnection(conn *jsonpConn) jsonpConnectionState {
	req := <-conn.requests
	defer func() { req.done <- true }()
	setContentTypeWithoutCache(req.rw.Header(), "application/javascript; charset=UTF-8")
	setCors(req.rw.Header(), req.req)
	conn.setCookie(req.rw.Header(), req.req)
	conn.sendOpenFrame(req.rw, req.callback)
	return JsonpOpenConnection
}

func JsonpOpenConnection(conn *jsonpConn) jsonpConnectionState {
	select {
	case req := <-conn.requests:
		defer func() { req.done <- true }()
		setContentTypeWithoutCache(req.rw.Header(), "application/javascript; charset=UTF-8")
		setCors(req.rw.Header(), req.req)
		conn.setCookie(req.rw.Header(), req.req)

		select {
		case frame, ok := <-conn.output():
			if !ok {
				conn.sendCloseFrame(req.rw, req.callback, 3000, "Go away!")
				return JsonpClosedConnection
			}
			frames := [][]byte{frame}
			for drain := true; drain; {
				select {
				case frame, ok = <-conn.output():
					frames = append(frames, frame)
				default:
					drain = false
				}
			}
			conn.sendDataFrame(req.rw, req.callback, frames...)
		case <-time.After(conn.HeartbeatDelay): // heartbeat
			conn.sendHeartbeatFrame(req.rw, req.callback)
		}
		return JsonpOpenConnection
	case <-time.After(conn.DisconnectDelay):
		return nil
	}
	panic("unreachable")
}
func JsonpClosedConnection(conn *jsonpConn) jsonpConnectionState {
	select {
	case req := <-conn.requests:
		defer func() { req.done <- true }()
		conn.sendCloseFrame(req.rw, req.callback, 3000, "Go away!")
		return JsonpClosedConnection
	case <-time.After(conn.DisconnectDelay):
		return nil
	}
	panic("unreachable")
}

func (this *context) JsonpSendHandler(rw http.ResponseWriter, req *http.Request) {

	sessid := mux.Vars(req)["sessionid"]

	if conn, exists := this.get(sessid); exists {
		jsonp_conn := conn.(*jsonpConn)

		payload, err := extractSendContent(req)

		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(payload) < 2 {
			// see https://github.com/sockjs/sockjs-protocol/pull/62
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, "Payload expected.")
			return
		}
		var a []interface{}
		if json.Unmarshal(payload, &a) != nil {
			// see https://github.com/sockjs/sockjs-protocol/pull/62
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, "Broken JSON encoding.")
			return
		}
		go func() { conn.input() <- []byte(payload) }()
		setContentTypeWithoutCache(rw.Header(), "text/plain; charset=UTF-8")
		setCors(rw.Header(), req)
		jsonp_conn.setCookie(rw.Header(), req)
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("ok"))
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}

func extractSendContent(req *http.Request) ([]byte, error) {
	// What are the options? Is this it?
	ctype := req.Header.Get("Content-Type")
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, req.Body)
	req.Body.Close()
	switch ctype {
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(buf.Bytes()))
		if err != nil {
			return []byte{}, errors.New("Could not parse query")
		}
		return []byte(values.Get("d")), nil
	case "text/plain":
		return buf.Bytes(), nil
	}
	return []byte{}, errors.New("Unrecognized content type")
}

/**************************************************************************************************/
/********** Jsonp writers *************************************************************************/
/**************************************************************************************************/
func (conn *jsonpConn) setCookie(header http.Header, req *http.Request) {
	if conn.CookieNeeded { // cookie is needed
		cookie, err := req.Cookie(session_cookie)
		if err == http.ErrNoCookie {
			cookie = test_cookie
		}
		cookie.Path = "/"
		header.Add("set-cookie", cookie.String())
	}
}

func (*jsonpConn) sendOpenFrame(w io.Writer, callback string) (int64, error) {
	n, err := fmt.Fprintf(w, "%s(\"o\");\r\n", callback)
	return int64(n), err
}

func (*jsonpConn) sendHeartbeatFrame(w io.Writer, callback string) (int64, error) {
	n, err := fmt.Fprintf(w, "%s(\"h\");\r\n", callback)
	return int64(n), err
}

func (*jsonpConn) sendDataFrame(w io.Writer, callback string, frames ...[]byte) (int64, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, "%s(\"a[", callback)
	for n, frame := range frames {
		if n > 0 {
			b.Write([]byte(","))
		}

		sesc := re.ReplaceAllFunc(frame, func(s []byte) []byte {
			return []byte(fmt.Sprintf(`\u%04x`, []rune(string(s))[0]))
		})

		bb, _ := json.Marshal(string(sesc))
		b.Write(bb[1 : len(bb)-1])
	}
	fmt.Fprintf(b, "]\");\r\n")
	return b.WriteTo(w)
}

func (*jsonpConn) sendCloseFrame(w io.Writer, callback string, code int, msg string) (int64, error) {
	n, err := fmt.Fprintf(w, "%s(\"c[%d,\\\"%s\\\"]\");\r\n", callback, code, msg)
	return int64(n), err
}
