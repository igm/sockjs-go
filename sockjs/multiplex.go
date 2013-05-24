package sockjs

import (
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