package main

import (
	"time"
)

func debouncer(offsets chan int, delay time.Duration) chan int {
	out := make(chan int)
	unsub := func() {} //nop

	go func() {
		for o := range offsets {
			unsub()
			unsub = emitAfter(o, out, delay)
		}
	}()

	return out
}

func emitAfter(offset int, out chan int, delay time.Duration) func() {
	unsub := make(chan bool, 1)

	go func() {
		select {
		case <-time.After(delay):
			out <- offset
		case <-unsub:
		}
	}()

	return func() {
		unsub <- true
	}
}
