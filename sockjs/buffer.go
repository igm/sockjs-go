package sockjs

// messageBuffer is an unbounded buffer that blocks on
// pop if it's empty until the new element is enqueued.
type messageBuffer struct {
	popCh   chan string
	closeCh <-chan struct{}
}

func newMessageBuffer(closeCh <-chan struct{}) *messageBuffer {
	return &messageBuffer{
		popCh:   make(chan string),
		closeCh: closeCh,
	}
}

func (b *messageBuffer) push(messages ...string) error {
	for _, message := range messages {
		select {
		case b.popCh <- message:
		case <-b.closeCh:
			return ErrSessionNotOpen
		}
	}

	return nil
}

func (b *messageBuffer) pop() (string, error) {
	select {
	case msg := <-b.popCh:
		return msg, nil
	case <-b.closeCh:
		return "", ErrSessionNotOpen
	}
}
