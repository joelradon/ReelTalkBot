// internal/cache.go

package internal

import (
	"sync"
	"time"
)

type Cache struct {
	data  map[string]string
	mutex sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[string]string),
	}
}

func (c *Cache) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	val, exists := c.data[key]
	return val, exists
}

func (c *Cache) Set(key, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[key] = value
}

// StartEviction periodically removes expired entries from the cache.
// Currently, it deletes all entries at each interval.
// Implement TTL checks or other eviction policies as needed.
func (c *Cache) StartEviction(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			c.mutex.Lock()
			for key := range c.data { // Removed 'value' as it's unused
				// TODO: Implement TTL checks or other eviction policies
				// Example: delete all entries (replace with actual logic)
				delete(c.data, key)
			}
			c.mutex.Unlock()
		}
	}()
}
