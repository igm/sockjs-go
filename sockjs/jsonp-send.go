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
)

// TODO try to refactor and reuse code with xhr_send
func (this *context) JsonpSendHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessid := vars["sessionid"]
	if conn, exists := this.get(sessid); exists {
		// data, err := ioutil.ReadAll(req.Body)
		data, err := extractSendContent(req)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, err.Error())
			return
		}

		if len(data) < 2 {
			// see https://github.com/sockjs/sockjs-protocol/pull/62
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, "Payload expected.")
			return
		}
		var a []interface{}
		if json.Unmarshal(data, &a) != nil {
			// see https://github.com/sockjs/sockjs-protocol/pull/62
			rw.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(rw, "Broken JSON encoding.")
			return
		}
		setCors(rw.Header(), req)
		setContentType(rw.Header(), "text/plain; charset=UTF-8")
		disableCache(rw.Header())
		// TODO refactor
		if conn.CookieNeeded { // cookie is needed
			cookie, err := req.Cookie(session_cookie)
			if err == http.ErrNoCookie {
				cookie = test_cookie
			}
			cookie.Path = "/"
			rw.Header().Add("set-cookie", cookie.String())
		}
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("ok"))
		go func() { conn.input_channel <- data }() // does not need to be extra routine?
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
