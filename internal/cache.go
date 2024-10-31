package internal

import (
	"sync"
	"time"
)

type CacheEntry struct {
	Response   string
	Expiration time.Time
}

type Cache struct {
	mu      sync.Mutex
	entries map[string]CacheEntry
}

// NewCache initializes a new Cache
func NewCache() Cache {
	return Cache{
		entries: make(map[string]CacheEntry),
	}
}

// Get retrieves a cached response if valid
func (c *Cache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.entries[key]
	if !found || time.Now().After(entry.Expiration) {
		return "", false
	}
	return entry.Response, true
}

// Set saves a response in the cache with a 30-minute expiration
func (c *Cache) Set(key, response string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = CacheEntry{
		Response:   response,
		Expiration: time.Now().Add(30 * time.Minute),
	}
}
