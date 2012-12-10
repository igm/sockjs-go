package sockjs

import (
	"code.google.com/p/gorilla/mux"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func (this *context) XhrSendHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessid := vars["sessionid"]
	if conn, exists := this.get(sessid); exists {
		data, err := ioutil.ReadAll(req.Body)
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
		conn.handleCookie(rw, req)
		rw.WriteHeader(http.StatusNoContent)
		go func() { conn.input_channel <- data }() // does not need to be extra routine?
	} else {
		rw.WriteHeader(http.StatusNotFound)
	}
}
func xhrOptions(rw http.ResponseWriter, req *http.Request) {
	setCors(rw.Header(), req)
	setAllowedMethods(rw.Header(), req, "OPTIONS, POST")
	setExpires(rw.Header())
	rw.WriteHeader(http.StatusNoContent)
}
