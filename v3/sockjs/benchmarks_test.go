package sockjs

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func BenchmarkSimple(b *testing.B) {
	var messages = make(chan string, 10)
	h := NewHandler("/echo", DefaultOptions, func(session *Session) {
		for m := range messages {
			session.Send(m)
		}
		session.Close(1024, "Close")
	})
	server := httptest.NewServer(h)
	defer server.Close()

	req, _ := http.NewRequest("POST", server.URL+fmt.Sprintf("/echo/server/%d/xhr_streaming", 1000), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	for n := 0; n < b.N; n++ {
		messages <- "some message"
	}
	fmt.Println(b.N)
	close(messages)
	resp.Body.Close()
}

func BenchmarkMessages(b *testing.B) {
	msg := strings.Repeat("m", 10)
	h := NewHandler("/echo", DefaultOptions, func(session *Session) {
		for n := 0; n < b.N; n++ {
			session.Send(msg)
		}
		session.Close(1024, "Close")
	})
	server := httptest.NewServer(h)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(session int) {
			reqc := 0
			req, _ := http.NewRequest("POST", server.URL+fmt.Sprintf("/echo/server/%d/xhr_streaming", session), nil)
			for {
				reqc++
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					log.Fatal(err)
				}
				reader := bufio.NewReader(resp.Body)
				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						goto AGAIN
					}
					if strings.HasPrefix(line, "data: c[1024") {
						resp.Body.Close()
						goto DONE
					}
				}
			AGAIN:
				resp.Body.Close()
			}
		DONE:
			wg.Done()
		}(i)
	}
	wg.Wait()
	server.Close()
}

var size = flag.Int("size", 4*1024, "Size of one message.")

func BenchmarkMessageWebsocket(b *testing.B) {
	flag.Parse()

	msg := strings.Repeat("x", *size)
	wsFrame := []byte(fmt.Sprintf("[%q]", msg))

	opts := Options{
		Websocket:       true,
		SockJSURL:       "//cdnjs.cloudflare.com/ajax/libs/sockjs-client/0.3.4/sockjs.min.js",
		HeartbeatDelay:  time.Hour,
		DisconnectDelay: time.Hour,
		ResponseLimit:   uint32(*size),
	}

	h := NewHandler("/echo", opts, func(session *Session) {
		for {
			msg, err := session.Recv()
			if err != nil {
				if session.GetSessionState() != SessionActive {
					break
				}
				b.Fatalf("Recv()=%s", err)
			}

			if err := session.Send(msg); err != nil {
				b.Fatalf("Send()=%s", err)
			}
		}
	})

	server := httptest.NewServer(h)
	defer server.Close()

	url := "ws" + server.URL[4:] + "/echo/server/0/websocket"

	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		b.Fatalf("Dial()=%s", err)
	}

	_, p, err := client.ReadMessage()
	if err != nil || string(p) != "o" {
		b.Fatalf("failed to start new session: frame=%v, err=%v", p, err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := client.WriteMessage(websocket.TextMessage, wsFrame); err != nil {
			b.Fatalf("WriteMessage()=%s", err)
		}

		if _, _, err := client.ReadMessage(); err != nil {
			b.Fatalf("ReadMessage()=%s", err)
		}
	}

	if err := client.Close(); err != nil {
		b.Fatalf("Close()=%s", err)
	}
}

func BenchmarkHandler_ParseSessionID(b *testing.B) {
	h := Handler{prefix: "/prefix"}
	url, _ := url.Parse("http://server:80/prefix/server/session/whatever")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.parseSessionID(url)
	}
}
