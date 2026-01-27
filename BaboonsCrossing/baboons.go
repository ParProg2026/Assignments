package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// --- Definitions (Same as before) ---

type Direction int
const (
	North Direction = iota
	South
)
func (d Direction) String() string {
	if d == North { return "North" }
	return "South"
}

type LightSwitch struct {
	counter int
	mutex   sync.Mutex
}
func NewLightSwitch() *LightSwitch { return &LightSwitch{counter: 0} }

func (ls *LightSwitch) Lock(sem *sync.Mutex) {
	ls.mutex.Lock()
	ls.counter++
	if ls.counter == 1 {
		sem.Lock()
	}
	ls.mutex.Unlock()
}
func (ls *LightSwitch) Unlock(sem *sync.Mutex) {
	ls.mutex.Lock()
	ls.counter--
	if ls.counter == 0 {
		sem.Unlock()
	}
	ls.mutex.Unlock()
}

type Canyon struct {
	rope         sync.Mutex
	capacity     *semaphore.Weighted
	turnstile    sync.Mutex
	ropeGrabbing sync.Mutex
	northSwitch  *LightSwitch
	southSwitch  *LightSwitch
}

func NewCanyon() *Canyon {
	return &Canyon{
		capacity:    semaphore.NewWeighted(2),
		northSwitch: NewLightSwitch(),
		southSwitch: NewLightSwitch(),
	}
}

// --- Modified Baboons with Logging ---

func (c *Canyon) Male(id int, dir Direction, action func()) {
	mySwitch := c.northSwitch
	if dir == South { mySwitch = c.southSwitch }

	// Entry
	// fmt.Printf("[%s Male %d] Arrived. Waiting for Turnstile...\n", dir, id)
	c.turnstile.Lock()
	// fmt.Printf("[%s Male %d] Passed Turnstile. Locking Direction...\n", dir, id)
	mySwitch.Lock(&c.rope)
	c.turnstile.Unlock()

	// Capacity
	c.ropeGrabbing.Lock()
	_ = c.capacity.Acquire(context.Background(), 2)
	c.ropeGrabbing.Unlock()

	if action != nil { action() }

	// Exit (Atomic Release)
	c.capacity.Release(2)
	mySwitch.Unlock(&c.rope)
}

func (c *Canyon) Female(id int, dir Direction, action func()) {
	mySwitch := c.northSwitch
	if dir == South { mySwitch = c.southSwitch }

	// Entry
	// fmt.Printf("[%s Female %d] Arrived. Waiting for Turnstile...\n", dir, id)
	c.turnstile.Lock()
	// fmt.Printf("[%s Female %d] Passed Turnstile. Locking Direction...\n", dir, id)
	mySwitch.Lock(&c.rope) 
	c.turnstile.Unlock()

	// Capacity
	c.ropeGrabbing.Lock()
	_ = c.capacity.Acquire(context.Background(), 1)
	c.ropeGrabbing.Unlock()

	if action != nil { action() }

	// Exit
	c.capacity.Release(1)
	mySwitch.Unlock(&c.rope)
}

// --- The Scenario ---

func main() {
	c := NewCanyon()
	var wg sync.WaitGroup

	start := time.Now()
	log := func(msg string) {
		fmt.Printf("[%04dms] %s\n", time.Since(start).Milliseconds(), msg)
	}

	// 1. Start Initial North Stream (They get the rope)
	log("--- PHASE 1: North Stream Starts ---")
	wg.Add(2)
	go func() {
		defer wg.Done()
		c.Male(1, North, func() {
			log("North Male 1 (Group 1) CROSSING...")
			time.Sleep(200 * time.Millisecond) // Long crossing
			log("North Male 1 (Group 1) FINISHED")
		})
	}()
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond) // Enters slightly later, but joins stream
		c.Female(2, North, func() {
			log("North Female 2 (Group 1) CROSSING...")
			time.Sleep(200 * time.Millisecond)
			log("North Female 2 (Group 1) FINISHED")
		})
	}()

	time.Sleep(50 * time.Millisecond)

	// 2. South Arrives (Should Block at Rope, Hold Turnstile)
	log("--- PHASE 2: South Arrives (Should block) ---")
	wg.Add(1)
	go func() {
		defer wg.Done()
		log("South Male 3 ARRIVES. Trying to enter...")
		c.Male(3, South, func() {
			log(">>> South Male 3 CROSSING (Finally) <<<")
			time.Sleep(100 * time.Millisecond)
			log("South Male 3 FINISHED")
		})
	}()

	time.Sleep(50 * time.Millisecond)

	// 3. Late North Arrives (Should Block at Turnstile)
	log("--- PHASE 3: Late North Arrives (Should wait for South) ---")
	wg.Add(1)
	go func() {
		defer wg.Done()
		log("North Female 4 (Late) ARRIVES. Trying to enter...")
		c.Female(4, North, func() {
			log("North Female 4 (Late) CROSSING...")
			time.Sleep(100 * time.Millisecond)
			log("North Female 4 (Late) FINISHED")
		})
	}()

	wg.Wait()
	log("--- Simulation Complete ---")
}