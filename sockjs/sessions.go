package sockjs

type connections struct {
	connections map[string]*conn
	req         chan func()
}

type connFactory func() *conn

func newConnections() connections {
	connections := connections{
		connections: make(map[string]*conn),
		req:         make(chan func()),
	}
	// go routine to perform concurrent-safe operations of data
	go func() {
		for r := range connections.req {
			r()
		}
	}()
	return connections
}

func (this *connections) get(sessid string) (conn *conn, exists bool) {
	resp := make(chan bool)
	this.req <- func() {
		conn, exists = this.connections[sessid]
		resp <- true
	}
	<-resp
	return
}

func (this *connections) getOrCreate(sessid string, f connFactory) (conn *conn, exists bool) {
	resp := make(chan bool)
	this.req <- func() {
		conn, exists = this.connections[sessid]
		if !exists {
			this.connections[sessid] = f()
			conn = this.connections[sessid]
		}
		resp <- true
	}
	<-resp
	return
}

func (this *connections) delete(sessid string) {
	this.req <- func() {
		delete(this.connections, sessid)
	}
}
