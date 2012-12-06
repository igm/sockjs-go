package sockjs

/*
Cotains package internal types (not public)
*/

import (
	"errors"
	"io"
)

type context struct {
	Config
	HandlerFunc
	connections
}

type conn interface {
	Conn
	input() chan []byte
	output() chan []byte
}

type baseConn struct {
	input_channel  chan []byte
	output_channel chan []byte
	context
}

func newBaseConn(ctx *context) baseConn {
	return baseConn{
		input_channel:  make(chan []byte),
		output_channel: make(chan []byte),
		context:        *ctx,
	}
}

func (this *baseConn) ReadMessage() ([]byte, error) {
	if val, ok := <-this.input_channel; ok {
		return val[1 : len(val)-1], nil
	}
	return []byte{}, io.EOF
}

func (this *baseConn) WriteMessage(val []byte) (count int, err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("already closed")
		}
	}()
	val2 := make([]byte, len(val))
	copy(val2, val)
	go func() {
		this.output_channel <- val2
	}()
	return len(val), nil
}

func (this *baseConn) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("already closed")
		}
	}()
	close(this.input_channel)
	close(this.output_channel)
	return
}

func (this *baseConn) input() chan []byte {
	return this.input_channel
}

func (this *baseConn) output() chan []byte {
	return this.output_channel
}
