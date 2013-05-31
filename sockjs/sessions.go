package sockjs

import (
	"sync"
)

type connections struct {
	connections map[string]*conn
	mu          sync.RWMutex
}

type connFactory func() *conn

func newConnections() connections {
	return connections{
		connections: make(map[string]*conn),
	}
}

func (this *connections) get(sessid string) (conn *conn, exists bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	conn, exists = this.connections[sessid]
	return
}

func (this *connections) getOrCreate(sessid string, f connFactory) (conn *conn, exists bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	conn, exists = this.connections[sessid]
	if !exists {
		this.connections[sessid] = f()
		conn = this.connections[sessid]
	}
	return
}

func (this *connections) delete(sessid string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	delete(this.connections, sessid)
}
