package main

import "time"

func debouncer(inOffset chan int, delay time.Duration) chan int {
	unsub := make(chan bool, 1)
	outOffset := make(chan int)

	go func() {
		for curOffset := range inOffset {
			unsub <- true
			unsub = make(chan bool, 1)

			go func() {
				select {
				case <-time.After(delay):
					outOffset <- curOffset
				case <-unsub:
				}
			}()
		}
	}()

	return outOffset
}
