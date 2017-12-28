package rudp

import (
	"fmt"
	"log"
)

func dbg(format string, v ...interface{}) {
	if debug {
		log.Printf(format, v...)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Printf("%v", err)
	}
}

func bitShow(n int) string {
	var ext string = "b"
	if n >= 1024 {
		n /= 1024
		ext = "Kb"
	}
	if n >= 1024 {
		n /= 1024
		ext = "Mb"
	}
	return fmt.Sprintf("%v %v", n, ext)
}
