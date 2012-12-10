package sockjs

import (
	"fmt"
	"net/http"
	"time"
)

/*******************  CORS/HTTP utility methods  ****************************/
func setCors(header http.Header, req *http.Request) {
	header.Add("Access-Control-Allow-Credentials", "true")
	header.Add("Access-Control-Allow-Origin", getOriginHeader(req))
	if allow_headers := req.Header.Get("Access-Control-Request-Headers"); allow_headers != "" && allow_headers != "null" {
		header.Add("Access-Control-Allow-Headers", allow_headers)
	}
}

func setAllowedMethods(header http.Header, req *http.Request, allow_methods string) {
	header.Add("Access-Control-Allow-Methods", allow_methods)
}

func setExpires(header http.Header) {
	header.Add("Expires", time.Now().AddDate(1, 0, 0).Format(time.RFC1123))
	header.Add("Cache-Control", fmt.Sprintf("public, max-age=%d", 365*24*60*60))
	header.Add("Access-Control-Max-Age", fmt.Sprintf("%d", 365*24*60*60))
}

func setContentType(header http.Header, content_type string) {
	header.Add("content-type", content_type)
}

func disableCache(header http.Header) {
	header.Add("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
}

func getOriginHeader(req *http.Request) string {
	origin := req.Header.Get("Origin")
	if origin == "" || origin == "null" {
		origin = "*"
	}
	return origin
}
