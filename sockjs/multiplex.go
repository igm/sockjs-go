package sockjs

import (
	"log"
	"strings"
	)

type ConnectionMultiplexer struct {
	channels map[string]map[Conn]bool
	fallback func(conn Conn, msg string)
}

func (this ConnectionMultiplexer) Handle(conn Conn) {
	for {
		if msg, err := conn.ReadMessage(); err == nil {
			// add client to a channel
			message := bytesToString(msg)
			parts := strings.Split(message, ",")
			if (len(parts) >= 3) {
				var msg_type string
				var msg_channel string
				var msg_payload string
				msg_type, parts = parts[len(parts)-1], parts[:len(parts) - 1]
				msg_channel, parts = parts[len(parts)-1], parts[:len(parts) - 1]
				msg_payload = strings.Join(parts,"")
				if msg_type == "sub" {
					go this.SubscribeClient(conn, msg_channel)
				} else if _, exists := this.channels[msg_channel]; exists {
					if msg_type == "uns" {
						go this.UnsubscribeClient(conn, msg_channel)
					} else if msg_type == "msg" {
						go this.Broadcast(msg_channel, msg_payload)
					}
				} else {
					go this.callFallback(conn, message)
				}
			}
		} else {
			log.Fatal(err)
			break
		}
	}
}

func Multiplexer(fallback func(conn Conn, msg string)) *ConnectionMultiplexer {
	muxer := new(ConnectionMultiplexer)
	muxer.fallback = fallback
	muxer.channels = make(map[string]map[Conn]bool)
	return muxer
}

func (this *ConnectionMultiplexer) SubscribeClient(conn Conn, channelName string) {
	if _, exists := this.channels[channelName]; exists {
		this.RegisterChannel(channelName)
	} 
	this.channels[channelName][conn] = true
}

func (this *ConnectionMultiplexer) UnsubscribeClient(conn Conn, channelName string) {
	if _, exists := this.channels[channelName][conn]; exists {
		delete(this.channels[channelName], conn)
	}
}

func (this *ConnectionMultiplexer) callFallback(conn Conn, msg string) {
	this.fallback(conn, msg)
}

func (this *ConnectionMultiplexer) RegisterChannel(channelName string) {
	this.channels[channelName] = make(map[Conn]bool)
}

func (this *ConnectionMultiplexer) Broadcast(channelName string, message string) {
	for connection, _ := range this.channels[channelName] {
		go connection.WriteMessage([]byte(message))
	}	
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
