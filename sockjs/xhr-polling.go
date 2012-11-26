package sockjs

import (
	"net/http"
)

func xhrPolling(rw http.ResponseWriter, req *http.Request, sessId string) {

	sockjs := sessions.Get(sessId)

	setContentTypeWithoutCache(rw.Header(), "application/javascript; encoding=UTF-8")
	setCors(rw.Header(), req)

	if sockjs.closed {
		rw.WriteHeader(http.StatusOK)
		sendFrame(`c[3000,"Go away!"]`, "\n", rw, nil)
		return
	}

	select {
	case val, ok := <-sockjs.out:
		if !ok {
			return
		}
		values := []string{val}
		for loop := true; loop; {
			select {
			case value, ok := <-sockjs.out:
				if !ok {
					return
				}
				values = append(values, value)
			default:
				loop = false
				sendFrame("a", "\n", rw, values)
			}
		}
	case _, ok := <-sockjs.hb:
		if !ok {
			return
		}
		sendFrame("h", "\n", rw, nil)

	case _, ok := <-sockjs.cch:
		if !ok {
			return
		}
		sendFrame(`c[3000,"Go away!"]`, "\n", rw, nil)
	}

}
