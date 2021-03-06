/*
Package implements LRU cache data structure, its statistic and crawler.
*/
package cache

import (
	"container/list"
	"time"
)

// Interface for applying some arbitrary type to LRU cache
type Cacheable interface {
	Key() string
	Size() int
}

// Structure implements LRU cache element.
// Structure consists of data, additional flags, expiration timestamp, unique id and list element for recentness.
type LRUCacheItem struct {
	Cacheable Cacheable
	Flags int
	Exptime int64
	Cas_unique int64
	listElement *list.Element
	touched bool
	ts int64
}

// Structure for storage statistics.
type LRUCacheStat struct {
	Volume int64
	Evictions uint64
	Expired_unfetched uint64
	Evicted_unfetched uint64
	Current_items int
	Total_items int64
	Crawler_reclaimed int64
	Outofmem int64
}

// Implementation of LRUCache itself.
// Structure consists max allowed size of memory, collection of elements and list for defining of recently usages.
type LRUCache struct {
	capacity int64 // bytes
	items map[string] *LRUCacheItem
	list *list.List
	Stats *LRUCacheStat
	Crawler *LRUCrawler
}

// Private method of LRUCache for promoting item to the top of list.
func (c *LRUCache) promote(item *LRUCacheItem) {
	item.touched = true
	c.list.MoveToFront(item.listElement)
}

// Private method of LRUCache for releasing of memory.
// Function receives amount of items to dispose. These items will be discarded from the tail of list.
// Amount == -1 - flushes all.
func (c *LRUCache) prune(amount int) {
	var counter = 0
	for{
		if amount != -1 && counter == amount { return }
		tail := c.list.Back()
		if tail == nil{ return }
		item := c.list.Remove(tail).(*LRUCacheItem)
		if amount != -1 {
			c.Stats.Evictions ++
			if !item.touched {
				c.Stats.Evicted_unfetched ++
			}
		}
		delete(c.items, item.Cacheable.Key())
		c.capacity += int64(item.Cacheable.Size())
		counter ++
		c.Stats.Current_items --
	}
}

// Public method of LRUCache, which retrieving data from it by received param "key"
// and returns pointer to structure LRUCacheItem with flags, data, id and exptime.
// If data is expired function will remove it and will return nil.
// Function also return nil if item with such key doesn't exist.
func (c *LRUCache) Get(key string) *LRUCacheItem {
	item, exists := c.items[key]
	if exists == false {
		return nil
	}
	// Passive expiration
	if c.deleteExpired(item.Cacheable) {
		return nil
	}
	c.promote(item)
	return item
}

// Public method of LRUCache, which sets item to the cache.
// Function receives item (with built-in size and key), flags for item, expiration timestamp and unique id.
// Function will update an item in cache, if such item does exist.
// Also function automatically can discard last 50 items if there is no space for new one.
// Function returns true if item was stored or false if there was no space for it.
func (c *LRUCache) Set(Cacheable Cacheable, flags int, expiration_ts int64, cas_unique int64) bool {
	if c.capacity < int64(Cacheable.Size()) {
		c.Stats.Outofmem ++
		c.prune(50)
	}
	//still not enough room, fail
	if c.capacity < int64(Cacheable.Size()) {
		return false
	}
	item, exists := c.items[Cacheable.Key()]
	if exists {
		old_size := item.Cacheable.Size()
		item.Cacheable = Cacheable
		item.Cas_unique = cas_unique
		item.Flags = flags
		item.Exptime = expiration_ts
		c.capacity -= int64(Cacheable.Size() - old_size)
		c.promote(item)
	} else {
		item = &LRUCacheItem{
			Cacheable: Cacheable,
			Flags: flags,
			Exptime: expiration_ts,
			Cas_unique: cas_unique,
			touched: false,
			ts: time.Now().Unix(),
		}
		item.listElement = c.list.PushFront(item)
		c.items[Cacheable.Key()] = item
		c.capacity -= int64(Cacheable.Size())
		c.Stats.Current_items ++
		c.Stats.Total_items ++
	}
	return true
}

// Public method of LRUCache, which discard item by received key param.
// Function returns true if such item does exist, otherwise false.
func (c *LRUCache) Flush(key string) bool {
	item, exists := c.items[key]
	if exists {
		c.list.Remove(item.listElement)
		delete(c.items, key)
		c.Stats.Current_items --
		return true
	} else { return false }
}

// Public method of LRUCache, which discard all items in cache.
func (c *LRUCache) FlushAll(){
	c.prune(-1)
}

// Public method of LRUCache, which sets Cas_unique field's value to passed param cas
// for existed item with passed param key.
// Returns true if item does exist, otherwise false.
func (c *LRUCache) SetCas(key string, cas int64) bool {
	_, exists := c.items[key]
	if exists {
		c.items[key].Cas_unique = cas
		return true
	}
	return false
}

// Getter for private capacity param
func (c *LRUCache) Capacity() int64 {
	return c.capacity
}

// Public function, which creates LRUCache instance.
// Function receives capacity param, which is uses for set of max allocating memory.
// Function returns pointer to created instance or nil if capacity is invalid.
func New(capacity int64 /* bytes */) *LRUCache {
	if capacity <= 0 { return nil }
	return &LRUCache {
		capacity: capacity,
		items: make(map[string] *LRUCacheItem, 10000),
		list: list.New(),
		Stats: &LRUCacheStat{capacity, 0, 0, 0, 0, 0, 0, 0},
		Crawler: NewCrawler(),
	}
}

// Private method of LRUCache, for flushing expired items.
// Function receives an item to check. If it does exist and it's timestamp is less than Now, item will be discarded and
// function will return true, otherwise false.
func (c *LRUCache) deleteExpired(Cacheable Cacheable) bool {
	item, exists := c.items[Cacheable.Key()]
	if exists {
		if item.Exptime < time.Now().Unix() && item.Exptime != 0 {
			c.list.Remove(item.listElement)
			if !item.touched {
				c.Stats.Expired_unfetched ++
			}
			delete(c.items, item.Cacheable.Key())
			c.Stats.Current_items --
			return true
		}
	}
	return false
}

// Function returns a timestamp of oldest stored item.
func (s *LRUCache) Oldest() int64 {
	if s.list.Back() == nil {
		return time.Now().Unix()
	}
	return s.list.Back().Value.(*LRUCacheItem).ts
}
