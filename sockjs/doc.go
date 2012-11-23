/*
Package sockjs provides SockJS server implementation. Following protocols are implemented:

	* xhr-streaming
	* xhr-polling
	* iframe-eventsource
	* websocket (hixie-76,hybi-10 uses "code.google.com/p/go.net/websocket")

For the complete list of supported sockjs protocols see: 
https://github.com/sockjs/sockjs-client#supported-transports-by-browser-html-served-from-http-or-https

Example:
	config := sockjs.Config{
		SockjsUrl: "http://cdn.sockjs.org/sockjs-0.3.2.min.js",
		Websocket: true,
		Prefix:    "/echo",
	}


	http.HandleFunc("/echo/", sockjs.SockJSHandler{
		Handler: SockJSHandler,
		Config:  config,
	})


	func SockJSHandler(conn *sockjs.SockJsConn)  {
		for {
			if msg, err := conn.Read(); err!=nil {
				return
			}
			go conn.Write(msg)
		}
	}

*/
package sockjs
