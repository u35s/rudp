package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/u35s/rudp"
)

func main() {
	raddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9981}
	//raddr := net.UDPAddr{IP: net.ParseIP("47.89.180.105"), Port: 9981}
	laddr := net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.DialUDP("udp", &laddr, &raddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	rconn := rudp.NewConn(conn, rudp.New())
	defer func() { fmt.Println("defer close", rconn.Close()) }()
	go func() {
		bts := make([]byte, 1)
		for i := uint8(0); ; i++ {
			bts[0] = byte(i)
			_, err := rconn.Write(bts)
			if err != nil {
				fmt.Printf("write err %v\n", err)
				os.Exit(1)
				break
			}
			time.Sleep(1e9)
		}
	}()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGHUP, syscall.SIGINT)
	select {
	case <-signalChan:
	}
}
