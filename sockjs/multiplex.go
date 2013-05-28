package sockjs

import (
	"log"
	"strings"
	)


type Channel struct {
	name string
	clients map[string]Conn
	onConnect func(conn Conn)
	onClose func(conn Conn)
	onData func(conn Conn, msg string)	
}

type ConnectionMultiplexer struct {
	channels map[string]Channel
	fallback func(conn Conn, msg string)
}


func NewMultiplexer(fallback func(conn Conn, msg string)) ConnectionMultiplexer {
	muxer := new(ConnectionMultiplexer)
	muxer.fallback = fallback
	muxer.channels = make(map[string]Channel)
	return (*muxer)
}

func NewChannel(name string) Channel{
	channel := new(Channel)
	channel.clients = make(map[string]Conn)
	channel.onConnect = func(conn Conn) { conn.WriteMessage([]byte("welcome!")) }
	channel.onClose = func(conn Conn) { conn.WriteMessage([]byte("bye!")) }
	channel.onData = func(conn Conn, msg string) { channel.Broadcast(msg) }
	return (*channel)
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
					go this.subscribeClient(conn, msg_channel)
				} else if channel, exists := this.channels[msg_channel]; exists {
					if msg_type == "uns" {
						go channel.UnsubscribeClient(conn)
					} else if msg_type == "msg" {
						go channel.onData(conn, msg_payload)
					}
				} else {
					go this.callFallback(conn, message)
				}
			}
		} else {
			break
		}
	}
}



func (this *ConnectionMultiplexer) callFallback(conn Conn, msg string) {
	this.fallback(conn, msg)
}

func (this *ConnectionMultiplexer) RegisterChannel(channel Channel) {
	this.channels[channel.name] = channel
}

func (this *ConnectionMultiplexer) subscribeClient(conn Conn, channel_name string) {
	if channel, exists := this.channels[channel_name]; exists {
		channel.SubscribeClient(conn)
	} else {
		channel := NewChannel(channel_name)
		this.channels[channel_name] = channel
		channel.SubscribeClient(conn)
	}	
}

func (this *Channel) Broadcast(message string) {
	for _, client := range this.clients {
		go client.WriteMessage([]byte(message))
	}	
}

func (this *Channel) SendToClient( client_id string, message string) {
	if client, exists := this.clients[client_id]; exists {
		go this.onData(client, message)
	}
}

func (this *Channel) SubscribeClient(conn Conn) {
	log.Println("sess: "+conn.GetSessionID())
	sessid := conn.GetSessionID()
	this.clients[sessid] = conn
	this.onConnect(conn)
}

func (this *Channel) UnsubscribeClient(conn Conn) {
	delete(this.clients, conn.GetSessionID())
	this.onClose(conn)
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
