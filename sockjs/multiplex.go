package sockjs

import (
	"log"
	"strings"
)

type Channel struct {
	name      string
	clients   map[Conn]bool
	OnConnect func(channel Channel, conn Conn)
	OnClose   func(channel Channel, conn Conn)
	OnData    func(channel Channel, conn Conn, msg string)
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

func NewChannel(name string) Channel {
	channel := new(Channel)
	channel.name = name
	channel.clients = make(map[Conn]bool)
	channel.OnConnect = func(this Channel, conn Conn) { this.SendToClient(conn, "welcome!") }
	channel.OnClose = func(this Channel, conn Conn) { this.SendToClient(conn, "bye!") }
	channel.OnData = func(this Channel, conn Conn, msg string) { this.Broadcast(msg) }
	return (*channel)
}

func (this ConnectionMultiplexer) Handle(conn Conn) {
	for {
		if msg, err := conn.ReadMessage(); err == nil {
			// add client to a channel
			message := strings.Trim(BytesToString(msg), "\"")
			parts := strings.Split(message, ",")
			if len(parts) >= 2 {
				var msg_type string
				var msg_channel string
				var msg_payload string
				msg_type, parts = parts[0], parts[1:]
				msg_channel, parts = parts[0], parts[1:]
				msg_payload = strings.Replace(strings.Join(parts, ","), "\\", "", -1)
				if msg_type == "sub" {
					go this.subscribeClient(conn, msg_channel)
				} else if channel, exists := this.channels[msg_channel]; exists {
					if msg_type == "uns" {
						go channel.UnsubscribeClient(conn)
					} else if msg_type == "msg" {
						go channel.OnData(channel, conn, msg_payload)
					}
				}
			} else {
				log.Println(message)
			}
		} else {
			break
		}
	}
}

func (this *ConnectionMultiplexer) GetHandler() func(conn Conn) {
	return func(conn Conn) { this.Handle(conn) }
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
	for client, _ := range this.clients {
		go this.SendToClient(client, message)
	}
}

func (this *Channel) SendToClient(client Conn, message string) {
	message = strings.Join([]string{"'msg", this.name, message + "'"}, ",")
	go client.WriteMessage([]byte(message))
}

func (this *Channel) SubscribeClient(conn Conn) {
	this.clients[conn] = true
	this.OnConnect((*this), conn)
}

func (this *Channel) UnsubscribeClient(conn Conn) {
	delete(this.clients, conn)
	this.OnClose((*this), conn)
}

func BytesToString(bytes []byte) string {
	n := -1
	for i, b := range bytes {
		if b == 0 {
			break
		}
		n = i
	}
	return string(bytes[:n+1])
}
