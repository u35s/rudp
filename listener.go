package rudp

import (
	"net"
	"sync"
)

func NewListener(conn *net.UDPConn) *RudpListener {
	listen := &RudpListener{conn: conn,
		newRudpUnConn: make(chan *RudpUnConn, 1024),
		newRudpErr:    make(chan error, 12),
		rudpUnConnMap: make(map[string]*RudpUnConn)}
	go listen.run()
	return listen
}

type RudpListener struct {
	conn *net.UDPConn
	lock sync.RWMutex

	newRudpUnConn chan *RudpUnConn
	newRudpErr    chan error
	rudpUnConnMap map[string]*RudpUnConn
}

//net listener interface
func (this *RudpListener) Accept() (net.Conn, error) { return this.AcceptRudp() }
func (this *RudpListener) Close() error {
	this.CloseAllRudp()
	return this.conn.Close()
}
func (this *RudpListener) Addr() net.Addr { return this.conn.LocalAddr() }

func (this *RudpListener) CloseRudp(addr string) {
	this.lock.Lock()
	delete(this.rudpUnConnMap, addr)
	this.lock.Unlock()
}

func (this *RudpListener) CloseAllRudp() {
	this.lock.Lock()
	for _, rconn := range this.rudpUnConnMap {
		rconn.closef = nil
		rconn.Close()
	}
	this.lock.Unlock()
}
func (this *RudpListener) AcceptRudp() (*RudpUnConn, error) {
	select {
	case c := <-this.newRudpUnConn:
		return c, nil
	case e := <-this.newRudpErr:
		return nil, e
	}
}
func (this *RudpListener) run() {
	data := make([]byte, MAX_PACKAGE)
	for {
		n, remoteAddr, err := this.conn.ReadFromUDP(data)
		if err != nil {
			this.CloseAllRudp()
			this.newRudpErr <- err
			return
		}
		this.lock.RLock()
		rudpUnConn, ok := this.rudpUnConnMap[remoteAddr.String()]
		this.lock.RUnlock()
		if !ok {
			rudpUnConn = NewUnConn(this.conn, remoteAddr, New(), this.CloseRudp)
			this.lock.Lock()
			this.rudpUnConnMap[remoteAddr.String()] = rudpUnConn
			this.lock.Unlock()
			this.newRudpUnConn <- rudpUnConn
			go rudpUnConn.run()
		}
		bts := make([]byte, n)
		copy(bts, data[:n])
		rudpUnConn.in <- bts
	}
}
