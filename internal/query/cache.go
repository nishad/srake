package query

import (
	"sync"
	"time"
)

// CacheEntry represents a cached item
type CacheEntry struct {
	Value      interface{}
	Expiration time.Time
}

// Cache provides a simple in-memory cache
type Cache struct {
	mu         sync.RWMutex
	items      map[string]*CacheEntry
	maxSize    int
	defaultTTL time.Duration
}

// NewCache creates a new cache
func NewCache(maxSize int, defaultTTL time.Duration) *Cache {
	cache := &Cache{
		items:      make(map[string]*CacheEntry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves an item from the cache
func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.items[key]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(entry.Expiration) {
		return nil
	}

	return entry.Value
}

// Set adds an item to the cache
func (c *Cache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL adds an item to the cache with a specific TTL
func (c *Cache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest items if at max size
	if len(c.items) >= c.maxSize {
		c.evictOldest()
	}

	c.items[key] = &CacheEntry{
		Value:      value,
		Expiration: time.Now().Add(ttl),
	}
}

// Delete removes an item from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear removes all items from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheEntry)
}

// GetStats returns cache statistics
func (c *Cache) GetStats() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	validCount := 0
	expiredCount := 0
	now := time.Now()

	for _, entry := range c.items {
		if now.After(entry.Expiration) {
			expiredCount++
		} else {
			validCount++
		}
	}

	return map[string]int{
		"total":   len(c.items),
		"valid":   validCount,
		"expired": expiredCount,
		"maxSize": c.maxSize,
	}
}

// evictOldest removes the oldest entry from the cache
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.items {
		if oldestKey == "" || entry.Expiration.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.Expiration
		}
	}

	if oldestKey != "" {
		delete(c.items, oldestKey)
	}
}

// cleanup periodically removes expired items
func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
