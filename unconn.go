package rudp

import (
	"net"
)

func NewUnConn(conn *net.UDPConn, remoteAddr *net.UDPAddr, rudp *Rudp, close func(string)) *RudpUnConn {
	con := &RudpUnConn{RudpConn: RudpConn{conn: conn, rudp: rudp,
		recvChan: make(chan []byte, 1<<10), recvErr: make(chan error, 2),
		sendChan: make(chan []byte, 1<<10), sendErr: make(chan error, 2),
		Tick: make(chan int, 2)},
		closef: close, remoteAddr: remoteAddr, in: make(chan []byte, 1<<10),
	}
	go con.run()
	return con
}

type RudpUnConn struct {
	RudpConn
	remoteAddr *net.UDPAddr
	closef     func(addr string)

	in chan []byte
}

func (this *RudpUnConn) Close() error {
	if this.closef != nil {
		this.closef(this.remoteAddr.String())
	}
	this.in <- []byte{TYPE_EOF}
	_, err := this.conn.WriteToUDP([]byte{TYPE_CORRUPT}, this.remoteAddr)
	checkErr(err)
	return err
}

func (this *RudpUnConn) RemoteAddr() net.Addr {
	return this.remoteAddr
}

func (this *RudpUnConn) run() {
	go func() {
		data := make([]byte, MAX_PACKAGE)
		for {
			select {
			case bts := <-this.in:
				this.rudp.Write(bts)
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
		}
	}()
	for {
		select {
		case tick := <-this.Tick:
			p := this.rudp.Update(tick)
			for p != nil {
				_, err := this.conn.WriteToUDP(p.Bts, this.remoteAddr)
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
