package rudp

import "log"

func checkErr(err error) {
	if err != nil {
		log.Printf("%v", err)
	}
}
