package main

import (
	"fmt"

	"github.com/u35s/rudp"
)

var dumpIdx int

func dumpRecv(U *rudp.Rudp) {
	bts := make([]byte, rudp.MAX_PACKAGE)
	for {
		n := U.Recv(bts)
		if n < 0 {
			fmt.Println("corrput")
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
	udp := rudp.New(1, 5)

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
	dump(udp.Update(nil, 1)) //dump0
	dump(udp.Update(nil, 1)) //dump1
	udp.Send(t3)
	udp.Send(t4)
	dump(udp.Update(nil, 1)) //dump2
	r1 := []byte{02, 00, 00, 02, 00, 03}
	dump(udp.Update(r1, 1)) //dump3
	dumpRecv(udp)
	r2 := []byte{5, 0, 1, 1,
		5, 0, 3, 3}
	dump(udp.Update(r2, 1)) //dump4
	dumpRecv(udp)
	r3 := []byte{5, 0, 0, 0,
		5, 0, 5, 5}
	dump(udp.Update(r3, 0)) //dump5
	r4 := []byte{5, 0, 2, 2}
	dump(udp.Update(r4, 1))
	dumpRecv(udp)
}
