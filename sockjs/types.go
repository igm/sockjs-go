package sockjs

/*
Cotains package internal types (not public)
*/

import (
	"errors"
	"io"
)

var ErrConnectionClosed = errors.New("Connection closed.")

type context struct {
	Config
	HandlerFunc
	connections
}

type conn struct {
	context
	input_channel    chan []byte
	output_channel   chan []byte
	httpTransactions chan *httpTransaction
}

func newConn(ctx *context) *conn {
	return &conn{
		input_channel:    make(chan []byte),
		output_channel:   make(chan []byte),
		httpTransactions: make(chan *httpTransaction),
		context:          *ctx,
	}
}

func (this *conn) ReadMessage() ([]byte, error) {
	if val, ok := <-this.input_channel; ok {
		return val[1 : len(val)-1], nil
	}
	return []byte{}, io.EOF
}

func (this *conn) WriteMessage(val []byte) (count int, err error) {
	defer func() {
		if recover() != nil {
			err = ErrConnectionClosed
		}
	}()
	val2 := make([]byte, len(val))
	copy(val2, val)
	go func() {
		this.output_channel <- val2
	}()
	return len(val), nil
}

func (this *conn) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = ErrConnectionClosed
		}
	}()
	close(this.input_channel)
	close(this.output_channel)
	return
}

type connectionStateFn func(*conn) connectionStateFn

func (this *conn) run(cleanupFn func()) {
	for state := openConnectionState; state != nil; {
		state = state(this)
	}
	cleanupFn()
}
