package main

// taken from https://github.com/igm/pubsub

import (
	"sync"
)

// Subscription Reader is used to read messages published by Publisher
type SubReader interface {
	// Read operation blocks and waits for message from Publisher
	Read() interface{}
}

// Publisher is used to publish messages. Can be directly created.
type Publisher struct {
	m       sync.Mutex
	lastMsg *msg
}

type subscriber struct{ in chan *msg }

type msg struct {
	val  interface{}
	next chan *msg
}

func newMsg(val interface{}) *msg { return &msg{val: val, next: make(chan *msg, 1)} }

// Publish publishes a message to all existing subscribers
func (p *Publisher) Publish(val interface{}) {
	p.m.Lock()
	defer p.m.Unlock()

	msg := newMsg(val)
	if p.lastMsg != nil {
		p.lastMsg.next <- msg
	}
	p.lastMsg = msg
}

// SubReader returns a new reader for reading published messages and a last published message.
func (p *Publisher) SubReader() (reader SubReader, lastMsg interface{}) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.lastMsg == nil {
		p.lastMsg = newMsg(nil)
	}
	return &subscriber{p.lastMsg.next}, p.lastMsg.val
}

// SubChannel returns a new channel for reading published messages and a last published message.
// If published messages equals (==) finalMsg then channel is closed afer putting message into channel.
func (p *Publisher) SubChannel(finalMsg interface{}) (msgChan <-chan interface{}, lastMsg interface{}) {
	listener, cur := p.SubReader()
	outch := make(chan interface{})
	go listen(listener, outch, finalMsg)
	return outch, cur
}

func listen(subscriber SubReader, ch chan interface{}, finalMsg interface{}) {
	defer close(ch)
	for {
		state := subscriber.Read()
		ch <- state
		if state == finalMsg {
			return
		}
	}
}

func (s *subscriber) Read() interface{} {
	msg := <-s.in
	s.in <- msg
	s.in = msg.next
	return msg.val
}
