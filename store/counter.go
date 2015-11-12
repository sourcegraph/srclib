package store

import "sync"

// counter is a simple thread-safe integer.
type counter struct {
	sync.RWMutex
	count int
}

// increment increments the counter by one.
func (c *counter) increment() {
	c.Lock()
	c.count++
	c.Unlock()
}

// get returns the counter's current value.
func (c *counter) get() int {
	c.RLock()
	v := c.count
	c.RUnlock()
	return v
}

// set sets the counter value.
func (c *counter) set(i int) {
	c.Lock()
	c.count = i
	c.Unlock()
}
