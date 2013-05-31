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

func (c *connections) get(sessid string) (conn *conn, exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	conn, exists = c.connections[sessid]
	return
}

func (c *connections) getOrCreate(sessid string, f connFactory) (conn *conn, exists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, exists = c.connections[sessid]
	if !exists {
		c.connections[sessid] = f()
		conn = c.connections[sessid]
	}
	return
}

func (c *connections) delete(sessid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.connections, sessid)
}
