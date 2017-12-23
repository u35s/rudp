package rudp

import (
	"net"
	"time"
)

func NewConn(conn *net.UDPConn, rudp *Rudp) *RudpConn {
	con := &RudpConn{conn: conn, rudp: rudp,
		recvChan: make(chan []byte, 1<<10), recvErr: make(chan error, 2),
		sendChan: make(chan []byte, 1<<10), sendErr: make(chan error, 2),
		Tick: make(chan int, 2),
	}
	go con.run()
	return con
}

type RudpConn struct {
	conn *net.UDPConn

	rudp *Rudp

	recvChan chan []byte
	recvErr  chan error

	sendChan chan []byte
	sendErr  chan error

	Tick chan int
}

func (this *RudpConn) Close() error {
	_, err := this.conn.Write([]byte{TYPE_CORRUPT})
	CheckErr(err)
	return this.conn.Close()
}

func (this *RudpConn) Read(bts []byte) (n int, err error) {
	select {
	case data := <-this.recvChan:
		copy(bts, data)
		return len(data), nil
	case err := <-this.recvErr:
		return 0, err
	}
}

func (this *RudpConn) Write(bts []byte) (n int, err error) {
	select {
	case this.sendChan <- bts:
		return len(bts), nil
	case err := <-this.sendErr:
		return 0, err
	}
}

func (this *RudpConn) LocalAddr() net.Addr {
	return this.conn.LocalAddr()
}

func (this *RudpConn) RemoteAddr() net.Addr {
	return this.conn.RemoteAddr()
}

func (this *RudpConn) SetDeadline(t time.Time) error      { return nil }
func (this *RudpConn) SetReadDeadline(t time.Time) error  { return nil }
func (this *RudpConn) SetWriteDeadline(t time.Time) error { return nil }

func (this *RudpConn) run() {
	go func() {
		data := make([]byte, MAX_PACKAGE)
		for {
			n, err := this.conn.Read(data)
			if err != nil {
				this.recvErr <- err
				return
			}
			this.rudp.Write(data[:n])
			for {
				n, err := this.rudp.Recv(data)
				if err != nil {
					this.recvErr <- err
					return
				} else if n == 0 {
					break
				}
				bts1 := make([]byte, n)
				copy(bts1, data[:n])
				this.recvChan <- bts1
			}
		}
	}()
	for {
		select {
		case tick := <-this.Tick:
			p := this.rudp.Update(tick)
			for p != nil {
				_, err := this.conn.Write(p.Bts)
				if err != nil {
					this.sendErr <- err
					return
				}
				p = p.Next
			}
		case bts := <-this.sendChan:
			_, err := this.rudp.Send(bts)
			if err != nil {
				this.sendErr <- err
				return
			}
		}
	}
}
