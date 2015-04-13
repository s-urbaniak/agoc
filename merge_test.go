package main

import "testing"

func TestEmptyMerge(t *testing.T) {
	var c1, c2 chan struct{}

	merge(c1, c2)
}

func TestOneSideMerge(t *testing.T) {
	c1 := make(chan struct{})
	var c2 chan struct{}

	go func(c chan struct{}) {
		c <- struct{}{}
	}(c1)

	<-merge(c1, c2)
}

func TestMerge(t *testing.T) {
	c1, c2 := make(chan struct{}), make(chan struct{})

	f := func(c chan struct{}) {
		c <- struct{}{}
	}

	go f(c1)
	go f(c2)

	x := merge(c1, c2)
	<-x
	<-x
}
