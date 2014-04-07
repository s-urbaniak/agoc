package main

import (
	"time"
)

func debouncer(offsets chan int, delay time.Duration) chan int {
	out := make(chan int)
	cancel := make(chan bool, 1)

	go func() {
		for o := range offsets {
			cancel <- true
			cancel = make(chan bool, 1)

			go func(o int, cancel chan bool) {
				select {
				case <-time.After(delay):
					out <- o
				case <-cancel:
				}
			}(o, cancel)
		}
	}()

	return out
}
