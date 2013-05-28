package sockjs

import (
	"log"
	"strings"
	)


type Channel struct {
	name string
	clients map[string]Conn
	OnConnect func(conn Conn)
	OnClose func(conn Conn)
	OnData func(conn Conn, msg string)	
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
	channel.name = name
	channel.clients = make(map[string]Conn)
	channel.OnConnect = func(conn Conn) { channel.SendToClient(conn, "welcome!")}
	channel.OnClose = func(conn Conn) { conn.WriteMessage([]byte("\"bye!\"")) }
	channel.OnData = func(conn Conn, msg string) { channel.Broadcast("something else entirely") }
	return (*channel)
}

func (this ConnectionMultiplexer) Handle(conn Conn) {
	for {
		if msg, err := conn.ReadMessage(); err == nil {
			// add client to a channel
			message := strings.Trim(bytesToString(msg), "\"")
			parts := strings.Split(message, ",")
			if (len(parts) >= 2) {
				var msg_type string
				var msg_channel string
				var msg_payload string
				msg_type, parts = parts[0], parts[1:]
				msg_channel, parts = parts[0], parts[1:]
				msg_payload = strings.Join(parts,"")
				if msg_type == "sub" {
					go this.subscribeClient(conn, msg_channel)
				} else if channel, exists := this.channels[msg_channel]; exists {
					if msg_type == "uns" {
						go channel.UnsubscribeClient(conn)
					} else if msg_type == "msg" {
						go channel.OnData(conn, msg_payload)
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
		go this.SendToClient(client, message)
	}	
}

func (this *Channel) SendToClient(client Conn, message string) {
	message = "\"msg,"+this.name+","+message+"\""
	go client.WriteMessage([]byte(message))
}

func (this *Channel) SubscribeClient(conn Conn) {
	this.clients["foo"] = conn
	this.OnConnect(conn)
}

func (this *Channel) UnsubscribeClient(conn Conn) {
	delete(this.clients, conn.GetSessionID())
	this.OnClose(conn)
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
