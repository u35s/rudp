package rudp

import (
	"net"
	"time"
)

func NewConn(conn *net.UDPConn, rudp *Rudp) *RudpConn {
	con := &RudpConn{conn: conn, rudp: rudp,
		recvChan: make(chan []byte, 1<<16), recvErr: make(chan error, 2),
		sendChan: make(chan []byte, 1<<16), sendErr: make(chan error, 2),
		SendTick: make(chan int, 2),
	}
	go con.run()
	return con
}

func NewUnConn(conn *net.UDPConn, remoteAddr *net.UDPAddr, rudp *Rudp, close func(string)) *RudpConn {
	con := &RudpConn{conn: conn, rudp: rudp, SendTick: make(chan int, 2),
		recvChan: make(chan []byte, 1<<16), recvErr: make(chan error, 2),
		sendChan: make(chan []byte, 1<<16), sendErr: make(chan error, 2),
		closef: close, remoteAddr: remoteAddr, in: make(chan []byte, 1<<16),
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

	SendTick chan int

	//unconected
	remoteAddr *net.UDPAddr
	closef     func(addr string)
	in         chan []byte
}

func (this *RudpConn) SetDeadline(t time.Time) error      { return nil }
func (this *RudpConn) SetReadDeadline(t time.Time) error  { return nil }
func (this *RudpConn) SetWriteDeadline(t time.Time) error { return nil }
func (this *RudpConn) LocalAddr() net.Addr                { return this.conn.LocalAddr() }
func (this *RudpConn) Connected() bool                    { return this.remoteAddr == nil }
func (this *RudpConn) RemoteAddr() net.Addr {
	if this.remoteAddr != nil {
		return this.remoteAddr
	}
	return this.conn.RemoteAddr()
}
func (this *RudpConn) Close() error {
	var err error
	if this.remoteAddr != nil {
		if this.closef != nil {
			this.closef(this.remoteAddr.String())
		}
		_, err = this.conn.WriteToUDP([]byte{TYPE_CORRUPT}, this.remoteAddr)
		this.in <- []byte{TYPE_EOF}
	} else {
		_, err = this.conn.Write([]byte{TYPE_CORRUPT})
	}
	checkErr(err)
	return err
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

func (this *RudpConn) send(bts []byte) (err error) {
	select {
	case this.sendChan <- bts:
		return nil
	case err := <-this.sendErr:
		return err
	}
}
func (this *RudpConn) Write(bts []byte) (n int, err error) {
	sz := len(bts)
	for len(bts)+MAX_MSG_HEAD > GENERAL_PACKAGE {
		if err := this.send(bts[:GENERAL_PACKAGE-MAX_MSG_HEAD]); err != nil {
			return 0, err
		}
		bts = bts[GENERAL_PACKAGE-MAX_MSG_HEAD:]
	}
	return sz, this.send(bts)
}

func (this *RudpConn) rudpRecv(data []byte) error {
	for {
		n, err := this.rudp.Recv(data)
		if err != nil {
			this.recvErr <- err
			return err
		} else if n == 0 {
			break
		}
		bts := make([]byte, n)
		copy(bts, data[:n])
		this.recvChan <- bts
	}
	return nil
}
func (this *RudpConn) conectedRecvLoop() {
	data := make([]byte, MAX_PACKAGE)
	for {
		n, err := this.conn.Read(data)
		if err != nil {
			this.recvErr <- err
			return
		}
		this.rudp.Input(data[:n])
		if this.rudpRecv(data) != nil {
			return
		}
	}
}
func (this *RudpConn) unconectedRecvLoop() {
	data := make([]byte, MAX_PACKAGE)
	for {
		select {
		case bts := <-this.in:
			this.rudp.Input(bts)
			if this.rudpRecv(data) != nil {
				return
			}
		}
	}
}
func (this *RudpConn) sendLoop() {
	var sendNum int
	for {
		select {
		case tick := <-this.SendTick:
		sendOut:
			for {
				select {
				case bts := <-this.sendChan:
					_, err := this.rudp.Send(bts)
					if err != nil {
						this.sendErr <- err
						return
					}
					sendNum++
					if sendNum >= maxSendNumPerTick {
						break sendOut
					}
				default:
					break sendOut
				}
			}
			sendNum = 0
			p := this.rudp.Update(tick)
			var num, sz int
			for p != nil {
				n, err := int(0), error(nil)
				if this.Connected() {
					n, err = this.conn.Write(p.Bts)
				} else {
					n, err = this.conn.WriteToUDP(p.Bts, this.remoteAddr)
				}
				if err != nil {
					this.sendErr <- err
					return
				}
				sz, num = sz+n, num+1
				p = p.Next
			}
			if num > 1 {
				show := bitShow(sz * int(time.Second/sendTick))
				dbg("send package num %v,sz %v, %v/s,local %v,remote %v",
					num, show, show, this.LocalAddr(), this.RemoteAddr())
			}
		}
	}
}
func (this *RudpConn) run() {
	if autoSend && sendTick > 0 {
		go func() {
			tick := time.Tick(sendTick)
			for {
				select {
				case <-tick:
					this.SendTick <- 1
				}
			}
		}()
	}
	go func() {
		if this.Connected() {
			this.conectedRecvLoop()
		} else {
			this.unconectedRecvLoop()
		}
	}()
	this.sendLoop()
}
