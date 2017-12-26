package main

import (
	"fmt"

	"github.com/u35s/rudp"
)

var dumpIdx int

func dumpRecv(U *rudp.Rudp) {
	bts := make([]byte, rudp.MAX_PACKAGE)
	for {
		n, err := U.Recv(bts)
		if err != nil {
			fmt.Println(err)
			break
		} else if n == 0 {
			break
		}
		fmt.Printf("RECV ")
		for i := 0; i < n; i++ {
			fmt.Printf("%02x ", bts[i])
		}
		fmt.Println()
	}
}

func dump(p *rudp.Package) {
	fmt.Printf("%d : ", dumpIdx)
	dumpIdx++
	for p != nil {
		fmt.Printf("(")
		for i := range p.Bts {
			fmt.Printf("%02x ", p.Bts[i])
		}
		fmt.Printf(") ")
		p = p.Next
	}
	fmt.Println()
}

func main() {
	//fmt.Println("vim-go")
	udp := rudp.New()

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

	udp.Send(t1)
	udp.Send(t2)
	dump(udp.Update(1)) //dump0
	dump(udp.Update(1)) //dump1
	udp.Send(t3)
	udp.Send(t4)
	dump(udp.Update(1)) //dump2
	r1 := []byte{rudp.TYPE_REQUEST, 00, 00, 00, 00, rudp.TYPE_REQUEST, 00, 03, 00, 03}
	udp.Input(r1)
	dump(udp.Update(1)) //dump3
	dumpRecv(udp)
	r2 := []byte{rudp.TYPE_NORMAL + 1, 0, 1, 1,
		rudp.TYPE_NORMAL + 1, 0, 3, 3}
	udp.Input(r2)
	dump(udp.Update(1)) //dump4
	dumpRecv(udp)
	r3 := []byte{rudp.TYPE_NORMAL + 1, 0, 0, 0,
		rudp.TYPE_NORMAL + 1, 0, 5, 5}
	udp.Input(r3)
	r4 := []byte{rudp.TYPE_NORMAL + 1, 0, 2, 2}
	udp.Input(r4)
	dump(udp.Update(1)) //dump5
	dumpRecv(udp)
}
