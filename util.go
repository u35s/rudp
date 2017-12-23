package rudp

import "log"

func CheckErr(err error) {
	if err != nil {
		log.Printf("%v", err)
	}
}
