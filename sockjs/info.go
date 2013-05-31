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

func createInfoData(ctx *context) infoData {
	return infoData{
		Websocket:    ctx.Websocket,
		CookieNeeded: ctx.CookieNeeded,
		Origins:      []string{"*:*"},
		Entropy:      rand.Int31(),
	}
}

func (ctx *context) infoHandler(rw http.ResponseWriter, req *http.Request) {
	header := rw.Header()
	setCors(header, req)
	setContentType(header, "application/json; charset=UTF-8")
	disableCache(header)
	rw.WriteHeader(http.StatusOK)
	json, _ := json.Marshal(createInfoData(ctx))
	rw.Write(json)
}

func infoOptionsHandler(rw http.ResponseWriter, req *http.Request) {
	header := rw.Header()
	setCors(header, req)
	setAllowedMethods(header, req, "OPTIONS, GET")
	setExpires(header)
	rw.WriteHeader(http.StatusNoContent)
}
