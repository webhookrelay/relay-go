package cond

import "sync"

// Cond implements a condition variable, a rendezvous point for goroutines
// waiting for or announcing the occurrence of an event.
//
// Unlike sync.Cond, Cond communicates with waiters via channels registered by
// the waiters. This permits goroutines to wait on Cond events using select.
type Cond struct {
	mu      sync.Mutex
	waiters []chan int
	last    int
}

// Register registers ch to receive a value when Notify is called.
// The value of last is the count of the times Notify has been called on this Cond.
// It functions of a sequence counter, if the value of last supplied to Register
// is less than the Conds internal counter, then the caller has missed at least
// one notification and will fire immediately.
//
// Sends by the broadcaster to ch must not block, therefor ch must have a capacity
// of at least 1.
func (c *Cond) Register(ch chan int, last int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if last < c.last {
		// notify this channel immediately
		ch <- c.last
		return
	}
	c.waiters = append(c.waiters, ch)
}

// Notify notifies all registered waiters that an event has ocurred.
func (c *Cond) Notify() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.last++

	for _, ch := range c.waiters {
		ch <- c.last
	}
	c.waiters = c.waiters[:0]
}
