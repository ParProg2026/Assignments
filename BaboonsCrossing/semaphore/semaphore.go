package semaphore

import "sync"

type Semaphore struct {
	mutex   *sync.Mutex
	free    *sync.Cond
	counter int
}

// acquire resource
func (s *Semaphore) P() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for s.counter <= 0 {
		s.free.Wait()
	}
	s.counter--
}

// release resource
func (s *Semaphore) V() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.counter++
	s.free.Signal()
}

// Create a new Semaphore
func MakeSemaphore(N int) (s *Semaphore) {
	s = &Semaphore{}
	s.mutex = &sync.Mutex{}
	s.free = sync.NewCond(s.mutex)
	s.counter = N
	return
}

/* http://www.golangpatterns.info/concurrency/semaphores
   and others get signaling semaphores wrong. */
