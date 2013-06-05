package sockjs

import (
	"sync"
)

type connections struct {
	sync.RWMutex
	connections map[string]*conn
}

type connFactory func() *conn

func newConnections() connections {
	return connections{
		connections: make(map[string]*conn),
	}
}

func (c *connections) get(sessid string) (conn *conn, exists bool) {
	c.RLock()
	defer c.RUnlock()
	conn, exists = c.connections[sessid]
	return
}

func (c *connections) getOrCreate(sessid string, f connFactory) (conn *conn, exists bool) {
	c.Lock()
	defer c.Unlock()
	conn, exists = c.connections[sessid]
	if !exists {
		c.connections[sessid] = f()
		conn = c.connections[sessid]
	}
	return
}

func (c *connections) delete(sessid string) {
	c.Lock()
	defer c.Unlock()
	delete(c.connections, sessid)
}
