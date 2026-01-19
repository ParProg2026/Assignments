package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// We don't need to use compare and swap as everybody has its own flags
var alices_flag = make([]atomic.Bool, 4)
var bobs_flag = make([]atomic.Bool, 4)
var charlies_flag = make([]atomic.Bool, 3)

const TOTAL = 10000

var counter = 0

func Alice(wg *sync.WaitGroup) {
	for i := 0; i < TOTAL; i++ {
		// Entry
		time.Sleep(1 * time.Microsecond)
		// Round 1 Alice vs Bob
		alices_flag[0].Store(true) // A wants to enter round 1

		// Alice acts as "Copier": She sets her flag equal to Bob's.
		alices_flag[1].Store(bobs_flag[1].Load())

		// Wait while Bob is present AND the flags are Equal (since Alice made them equal, she yields).
		for bobs_flag[0].Load() && alices_flag[1].Load() == bobs_flag[1].Load() {
			runtime.Gosched()
		}

		// Alice wins vs Bob.
		// Round 2 Alice vs Charlie
		alices_flag[2].Store(true)

		// Alice acts as "Copier" vs Charlie: She sets her flag equal to Charlie's.
		alices_flag[3].Store(charlies_flag[1].Load())

		// Wait while Charlie is present AND the flags are Equal.
		for charlies_flag[0].Load() && alices_flag[3].Load() == charlies_flag[1].Load() {
			runtime.Gosched()
		}

		// Critical
		// Alice's dog in the yard
		counter++
		println("Alice's dog in the yard")

		// Exit

		// Reset rounds access
		alices_flag[2].Store(false)
		alices_flag[0].Store(false)
	}
	wg.Done()
}

func Bob(wg *sync.WaitGroup) {
	for i := 0; i < TOTAL; i++ {
		// Entry
		time.Sleep(1 * time.Microsecond)
		// Round 1 Bob vs Alice
		bobs_flag[0].Store(true)

		// Bob acts as "Inverter": He sets his flag opposite to Alice's.
		bobs_flag[1].Store(!alices_flag[1].Load())

		// Wait while Alice is present AND the flags are Different (since Bob made them different, he yields).
		for alices_flag[0].Load() && bobs_flag[1].Load() != alices_flag[1].Load() {
			runtime.Gosched()
		}

		// Round 2 Bob vs Charlie
		bobs_flag[2].Store(true)

		// Bob acts as "Copier" vs Charlie: He sets his flag equal to Charlie's.
		bobs_flag[3].Store(charlies_flag[2].Load())

		// Wait while Charlie is present AND the flags are Equal.
		for charlies_flag[0].Load() && bobs_flag[3].Load() == charlies_flag[2].Load() {
			runtime.Gosched()
		}

		// critical
		// Bob's dog is in the garden
		counter++
		println("Bob's dog is in the garden")

		// exit

		// Reset rounds
		bobs_flag[2].Store(false)
		bobs_flag[0].Store(false)
	}
	wg.Done()
}

func Charlie(wg *sync.WaitGroup) {
	for i := 0; i < TOTAL; i++ {
		// entry
		time.Sleep(1 * time.Microsecond)
		charlies_flag[0].Store(true)

		// Charlie acts as "Inverter" against BOTH Alice and Bob.
		// Vs Alice: Make it different
		charlies_flag[1].Store(!alices_flag[3].Load())
		// Vs Bob: Make it different
		charlies_flag[2].Store(!bobs_flag[3].Load())

		// Wait Logic:
		// waits if:
		// (Alice is present AND flags are Different) OR (Bob is present AND flags are Different)
		for (alices_flag[2].Load() && charlies_flag[1].Load() != alices_flag[3].Load()) ||
			(bobs_flag[2].Load() && charlies_flag[2].Load() != bobs_flag[3].Load()) {
		}

		// critical
		// Charlie's dog is in the garden
		counter++
		println("Charlie's dog is in the garden")

		// exit
		charlies_flag[0].Store(false)
	}
	wg.Done()
}

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(3)

	go Alice(wg)
	go Bob(wg)
	go Charlie(wg)
	wg.Wait()

	println("Done")
	println(counter)
}
