package main

import (
	"fmt"
	"highway/semaphore"
	"time"
)

const (
	N = true
	S = false
)

var (
	nos  int = 0
	non  int = 0
	d1   int = 0
	d2   int = 0
	last bool
	s1   = semaphore.MakeSemaphore(0)
	s2   = semaphore.MakeSemaphore(0)
	e    = semaphore.MakeSemaphore(1)
)

func main() {
	go north()
	go south()
	time.Sleep(time.Second * 5)
}

func north() {
	fmt.Println("North Start")
	for true {
		e.P()
		if !(nos == 0 && (d2 == 0 || last == S)) {
			d1++
			e.V()
			s1.P()
		}
		non++
		fmt.Println("North Car going")
		time.Sleep(time.Millisecond * 10)
		fmt.Println("North Car leaving")
		signal()
		non--
	}
}

func south() {
	fmt.Println("South Start")
	for true {
		e.P()
		if !(non == 0 && (d1 == 0 || last == N)) {
			d2++
			e.V()
			s2.P()
		}
		nos++
		fmt.Println("South Car going")
		time.Sleep(time.Millisecond * 10)
		fmt.Println("South Car leaving")
		signal()
		nos--
	}
}

func signal() {
	if d1 > 0 && nos == 0 && (d2 == 0 || last == S) {
		d1--
		last = N
		s1.V()
	} else if d2 > 0 && non == 0 && (d1 == 0 || last == N) {
		d2--
		last = S
		s2.V()
	} else {
		e.V()
	}
}
