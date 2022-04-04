package server

import (
	"time"
)

func PreStartWatcher() chan bool {
	start := make(chan bool)

	go func() {
		for {
			if checkKeystore() {
				start <- true
				return
			}

			time.Sleep(2 * time.Second)
		}
	}()

	return start
}
