package rudp

import (
	"net"
	"testing"
)

func Test_bitShow(t *testing.T) {
	if show := bitShow(1023); show != "1023 b" {
		t.Errorf("byte show error %v,show %v", 1023, show)
	}

	if show := bitShow(1025); show != "1 Kb" {
		t.Errorf("byte show error %v,show %v", 1025, show)
	}

	if show := bitShow(1024*1024 + 1); show != "1 Mb" {
		t.Errorf("byte show error %v,show %v", 1024*1024+1, show)
	}
}

func Test_Rudp(t *testing.T) {
	udp := New()

	t1 := []byte{1, 2, 3, 4}
	t2 := []byte{5, 6, 7, 8}
	t3 := []byte{
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 1, 1, 1, 3,
		2, 1, 1, 1, 1, 1, 1, 3, 2, 1, 1, 1, 10, 11, 12, 13,
	}
	t4 := []byte{4, 3, 2, 1}
	send := func(bts []byte) {
		n, err := udp.Send(bts)
		if err != nil {
			t.Error(err)
		}
		if n != len(bts) {
			t.Errorf("send length error,send %v,realy %v", n, len(bts))
		}
	}
	send(t1)
	send(t2)
	if udp.Update(0) != nil {
		t.Errorf("update 0 return error")
	}
	pkg := udp.Update(sendDelayTick)
	sendLen := func(p *Package) (sz int) {
		for p != nil {
			sz += len(p.Bts)
			p = p.Next
		}
		return
	}
	if sendLen(pkg) != len(t1)+3+len(t2)+3 {
		t.Errorf("out pkg t1,t2 length error,out %v,realy %v",
			sendLen(pkg), len(t1)+3+len(t2)+3)
	}
	pkg = udp.Update(sendDelayTick)
	if pkg == nil || len(pkg.Bts) != 1 {
		t.Errorf("ping error,pkg %v", pkg)
	}
	send(t3)
	send(t4)
	pkg = udp.Update(sendDelayTick)
	if sendLen(pkg) != len(t3)+4+len(t4)+3 {
		t.Errorf("out pkg t3,t4 length error,out %v,realy %v",
			sendLen(pkg), len(t3)+4+len(t4)+3)
	}

	r1 := []byte{TYPE_REQUEST, 00, 00, 00, 00, TYPE_REQUEST, 00, 03, 00, 03}
	udp.Input(r1)
	pkg = udp.Update(sendDelayTick)
	if sendLen(pkg) != len(t1)+3+len(t4)+3 {
		t.Errorf("miss out pkg t1,t4 length error,out %v,realy %v",
			sendLen(pkg), len(t1)+3+len(t4)+3)
	}
}

func Test_RudpConn(t *testing.T) {
	addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: 9981}
	sconn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return
	}
	listener := NewListener(sconn)
	var send = []byte{'h', 'e', 'l', 'l', 'o'}
	go func() {
		rconn, err := listener.AcceptRudp()
		if err != nil {
			t.Error(err)
			return
		}
		data := make([]byte, MAX_PACKAGE)
		n, err := rconn.Read(data)
		if err != nil {
			t.Error(err)
			return
		}
		if string(data[:n]) != string(send) {
			t.Error(err)
			return
		}
		rconn.Write(send)
		if err := rconn.Close(); err != nil {
			t.Error(err)
		}
	}()
	raddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9981}
	laddr := net.UDPAddr{IP: net.IPv4zero, Port: 0}
	cconn, err := net.DialUDP("udp", &laddr, &raddr)
	if err != nil {
		t.Error(err)
		return
	}
	rconn := NewConn(cconn, New())
	rconn.Write(send)
	data := make([]byte, MAX_PACKAGE)
	n, err := rconn.Read(data)
	if err != nil {
		return
	}
	if string(data[:n]) != string(send) {
		t.Error(err)
		return
	}
	if err := rconn.Close(); err != nil {
		t.Error(err)
	}
	if err := listener.Close(); err != nil {
		t.Error(err)
	}
}
