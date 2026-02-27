package slugify

import (
	"container/list"
	"sync"
)

// lruCache is a minimal least recently used cache implementation.
//
// This type exists because this library needs bounded caching to avoid unbounded
// memory growth, and it should work without requiring consumers to pull in a large
// dependency.
//
// The implementation is safe for concurrent use.
type lruCache[K comparable, V any] struct {
	mutex    sync.Mutex
	capacity int
	entries  map[K]*list.Element
	order    *list.List
}

type lruCacheEntry[K comparable, V any] struct {
	key   K
	value V
}

// newLRUCache creates a new least recently used cache.
//
// A capacity of zero or less disables caching.
func newLRUCache[K comparable, V any](capacity int) *lruCache[K, V] {
	if capacity < 0 {
		capacity = 0
	}
	return &lruCache[K, V]{
		capacity: capacity,
		entries:  make(map[K]*list.Element),
		order:    list.New(),
	}
}

// Get returns the cached value for key and marks it as recently used.
func (c *lruCache[K, V]) Get(key K) (V, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var zero V
	if c.capacity <= 0 {
		return zero, false
	}

	element, ok := c.entries[key]
	if !ok {
		return zero, false
	}
	c.order.MoveToFront(element)
	return element.Value.(lruCacheEntry[K, V]).value, true
}

// Add inserts or updates a cached value and marks it as recently used.
func (c *lruCache[K, V]) Add(key K, value V) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.capacity <= 0 {
		return
	}

	if element, ok := c.entries[key]; ok {
		element.Value = lruCacheEntry[K, V]{key: key, value: value}
		c.order.MoveToFront(element)
		return
	}

	element := c.order.PushFront(lruCacheEntry[K, V]{key: key, value: value})
	c.entries[key] = element

	for len(c.entries) > c.capacity {
		back := c.order.Back()
		if back == nil {
			break
		}
		entry := back.Value.(lruCacheEntry[K, V])
		delete(c.entries, entry.key)
		c.order.Remove(back)
	}
}

// Clear removes all entries from the cache.
func (c *lruCache[K, V]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[K]*list.Element)
	c.order.Init()
}

// SetCapacity changes the maximum number of entries the cache may hold.
//
// If the capacity is reduced, least recently used entries are evicted until the
// cache size is within the new bound.
func (c *lruCache[K, V]) SetCapacity(capacity int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if capacity < 0 {
		capacity = 0
	}
	c.capacity = capacity

	if c.capacity <= 0 {
		c.entries = make(map[K]*list.Element)
		c.order.Init()
		return
	}

	for len(c.entries) > c.capacity {
		back := c.order.Back()
		if back == nil {
			break
		}
		entry := back.Value.(lruCacheEntry[K, V])
		delete(c.entries, entry.key)
		c.order.Remove(back)
	}
}
