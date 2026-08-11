package manifest

import (
	"container/list"
	"sync"
)

// DefaultCacheCapacity is the default capacity for the manifest LRU cache.
const DefaultCacheCapacity = 50

type cacheItem struct {
	key   string
	value *Manifest
}

// Cache is a thread-safe LRU cache for snapshot manifests keyed by snapshot ID.
type Cache struct {
	capacity int
	mu       sync.RWMutex
	items    map[string]*list.Element
	evictList *list.List
}

var globalCache = NewCache(DefaultCacheCapacity)

// NewCache constructs an LRU cache with the specified capacity.
func NewCache(capacity int) *Cache {
	if capacity <= 0 {
		capacity = DefaultCacheCapacity
	}
	return &Cache{
		capacity:  capacity,
		items:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

// Get retrieves a manifest from the cache. Returns (nil, false) if missing.
func (c *Cache) Get(id string) (*Manifest, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.evictList.MoveToFront(elem)
		return elem.Value.(*cacheItem).value, true
	}
	return nil, false
}

// Put adds or updates a manifest in the cache, evicting the least recently used item if full.
func (c *Cache) Put(m *Manifest) {
	if m == nil || m.ID == "" {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[m.ID]; ok {
		c.evictList.MoveToFront(elem)
		elem.Value.(*cacheItem).value = m
		return
	}

	if c.evictList.Len() >= c.capacity {
		c.removeOldest()
	}

	item := &cacheItem{key: m.ID, value: m}
	elem := c.evictList.PushFront(item)
	c.items[m.ID] = elem
}

// Invalidate removes an item by snapshot ID from the cache.
func (c *Cache) Invalidate(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[id]; ok {
		c.removeElement(elem)
	}
}

// Clear purges all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*list.Element)
	c.evictList.Init()
}

// Len returns the current count of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.evictList.Len()
}

func (c *Cache) removeOldest() {
	elem := c.evictList.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

func (c *Cache) removeElement(elem *list.Element) {
	c.evictList.Remove(elem)
	item := elem.Value.(*cacheItem)
	delete(c.items, item.key)
}
