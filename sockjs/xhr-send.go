package sockjs

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
)

func (ctx *context) XhrSendHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	sessid := vars["sessionid"]
	if conn, exists := ctx.get(sessid); exists {
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
		dataStrings := make([]string, 1)
		if json.Unmarshal(data, &dataStrings); err != nil {
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
		for _, s := range dataStrings {
			// Convert multiple frames into single frames
			tmpArray := [1]string{s}
			b, _ := json.Marshal(tmpArray)
			conn.input_channel <- b
		}
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
