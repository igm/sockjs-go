package sockjs

import (
	"strings"
	"log"
	"container/list"
	)

type ConnectionMultiplexer struct {
	channels map[string]*list.List
	fallback func(conn Conn)
}

func (this ConnectionMultiplexer) Handle(conn Conn) {
	for {
		if msg, err := conn.ReadMessage(); err == nil {
			// add client to a channel
			parts := strings.Split(bytesToString(msg), ",")
			if (len(parts) >= 3) {
				var msg_type string
				var msg_channel string
				var msg_payload string
				msg_type, parts = parts[len(parts)-1], parts[:len(parts) - 1]
				msg_channel, parts = parts[len(parts)-1], parts[:len(parts) - 1]
				msg_payload = strings.Join(parts,"")
				if msg_type == 'sub' {
					if 
				} else if msg_type == 'uns' {
					
				} else if msg_type == 'msg' {
					
				} else {
					
				}
			}
			// create new channel (if it doesn't exists)
			// broadcast message to channel
			// remove client from channel
			// fallback to another handler function
			log.Println(msg)
		} else {
			return
		}
	}
}

func New(fallback func(conn Conn)) *ConnectionMultiplexer {
	muxer := new(ConnectionMultiplexer)
	muxer.fallback = fallback
	return muxer
}

func (this *ConnectionMultiplexer) SubscribeClient(conn Conn, channelName string) {
}

func (this *ConnectionMultiplexer) UnsubscribeClient(conn Conn, channelName string) {
}

func (this *ConnectionMultiplexer) callFallback(conn Conn, msg string) {
}

func (this *ConnectionMultiplexer) RegisterChannel(channelName string) {	
}

func (this *ConnectionMultiplexer) Broadcast(channelName string, message string) {
}

func bytesToString(bytes []byte) string {
	n := -1
	for i, b := range bytes {
		if b == 0 {
			break
		}
		n = i
	}
	return string(bytes[:n+1])
}
