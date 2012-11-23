package sockjs

import (
	"encoding/json"
	"math/rand"
	"net/http"
)

type infoData struct {
	Websocket    bool     `json:"websocket"`
	CookieNeeded bool     `json:"cookie_needed"`
	Origins      []string `json:"origins"`
	Entropy      int32    `json:"entropy"`
}

func createInfoData(ws bool) infoData {
	return infoData{
		Websocket:    ws,
		CookieNeeded: true,
		Origins:      []string{"*:*"},
		Entropy:      rand.Int31(),
	}
}

func infoHandler(rw http.ResponseWriter, req *http.Request, s *SockJSHandler) {
	header := rw.Header()
	setCors(header, req)
	setContentTypeWithoutCache(header, "application/json; charset=UTF-8")
	rw.WriteHeader(http.StatusOK)
	json, _ := json.Marshal(createInfoData(s.Config.Websocket))
	rw.Write(json)
}

func infoOptionsHandler(rw http.ResponseWriter, req *http.Request) {
	header := rw.Header()
	setCorsAllowedMethods(header, req, "OPTIONS, GET")
	setExpires(header)
	rw.WriteHeader(http.StatusNoContent)
}
