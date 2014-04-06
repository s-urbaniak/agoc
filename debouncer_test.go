package main

import (
	"log"
	"testing"
	"time"
)

func TestDebouncer(*testing.T) {
	o := make(chan int)
	d := debouncer(o, 100 * time.Millisecond)
	o <- 1
	o <- 2
	time.Sleep(50 * time.Millisecond)
	o <- 3
	time.Sleep(50 * time.Millisecond)
	o <- 4
	time.Sleep(200 * time.Millisecond)
	log.Printf("%v\n", <-d)
	
	o <- 5
	o <- 6
	time.Sleep(50 * time.Millisecond)
	o <- 7
	time.Sleep(200 * time.Millisecond)
	log.Printf("%v\n", <-d)
}
