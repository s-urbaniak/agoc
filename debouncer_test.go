package main

import (
	"testing"
	"time"
)

func TestDebouncer(t *testing.T) {
	o := make(chan int)
	d := debouncer(o, 100*time.Millisecond)

	o <- 1
	o <- 2
	time.Sleep(50 * time.Millisecond)
	o <- 3
	time.Sleep(50 * time.Millisecond)
	o <- 4
	time.Sleep(200 * time.Millisecond)

	if v := <-d; v != 4 {
		t.Errorf("expected 4, got %v\n", v)
	}

	o <- 5
	o <- 6
	time.Sleep(50 * time.Millisecond)
	o <- 7
	time.Sleep(200 * time.Millisecond)

	if v := <-d; v != 7 {
		t.Errorf("expected 7, got %v\n", v)
	}
}
