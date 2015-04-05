package main

import "time"

type cancel struct{}

func debouncer(offsets chan int, delay time.Duration) chan int {
	out := make(chan int)
	c := make(chan cancel)

	go func() {
		for o := range offsets {
			close(c)
			c = make(chan cancel)

			go func(o int, c chan cancel) {
				select {
				case <-time.After(delay):
					out <- o
				case <-c:
				}
			}(o, c)
		}
	}()

	return out
}
