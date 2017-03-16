package sockjs

import "sync"

// messageBuffer is an unbounded buffer that blocks on
// pop if it's empty until the new element is enqueued.
type messageBuffer struct {
	popCh   chan string
	sleepCh chan struct{}
	closeCh <-chan struct{}

	mu    sync.Mutex
	queue []string
}

func newMessageBuffer(closeCh <-chan struct{}) *messageBuffer {
	b := &messageBuffer{
		popCh:   make(chan string),
		sleepCh: make(chan struct{}, 1),
		closeCh: closeCh,
	}

	go b.process()

	return b
}

func (b *messageBuffer) push(messages ...string) {
	if len(messages) == 0 {
		return
	}

	b.mu.Lock()
	sleeping := len(b.queue) == 0
	b.queue = append(b.queue, messages...)
	b.mu.Unlock()

	if sleeping {
		b.sleepCh <- struct{}{}
	}
}

func (b *messageBuffer) next() (msg string, ok bool) {
	b.mu.Lock()
	if len(b.queue) != 0 {
		msg, b.queue = b.queue[0], b.queue[1:]
		ok = true
	}
	b.mu.Unlock()

	return msg, ok
}

func (b *messageBuffer) pop() (string, error) {
	select {
	case msg := <-b.popCh:
		return msg, nil
	case <-b.closeCh:
		b.clear()
		return "", ErrSessionNotOpen
	}
}

func (b *messageBuffer) process() {
	for {
		select {
		case <-b.sleepCh:
			for msg, ok := b.next(); ok; msg, ok = b.next() {
				select {
				case b.popCh <- msg:
				case <-b.closeCh:
					b.clear()
					return
				}
			}
		case <-b.closeCh:
			return
		}
	}
}

func (b *messageBuffer) clear() {
	b.mu.Lock()
	b.queue = nil
	b.mu.Unlock()
}
