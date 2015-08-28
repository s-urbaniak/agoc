package main

import (
	"sync"
	"time"
)

type cancel struct{}

func debouncer(offsets chan int, delay time.Duration) chan int {
	out := make(chan int)
	c := make(chan cancel)
	var wg sync.WaitGroup

	go func() {
		defer func() {
			close(c)
			wg.Wait()
			close(out)
		}()

		for o := range offsets {
			close(c)
			c = make(chan cancel)
			wg.Add(1)

			go func(o int, c chan cancel) {
				defer wg.Done()

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
