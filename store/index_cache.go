package store

import "sync"

// cacheableIndexStore is an index store which can allow the indexes to be
// shared across instances of the store. The store itself needs to be
// instrumented with calls to `cacheGet` and `cachePut`
type cacheableIndexStore interface {
	StoreKey() interface{}
}

// indexCacheKey is used to uniquely identify a (store, index) pair.
type indexCacheKey struct {
	// storeKey is a string or struct to uniquely identify a
	// store. Usually this is just a URI or path to the underlying store
	storeKey  interface{}
	indexName string
}

// indexCache stores indexes for use across stores. This is to prevent the
// cost of deserializing the index from the underlying VFS store
type indexCache struct {
	indexes map[indexCacheKey]Index
	sync.RWMutex
}

var defaultIndexCache *indexCache = &indexCache{
	indexes: map[indexCacheKey]Index{},
}

// cacheGet attempts to fetch an instance of a loaded Index from an in-memory
// cache. If it fails, it will return the fallback index.
//
// Note: Their may be multiple goroutines reading from the Index
// concurrently. Pleasue ensure all uses of the Index are read-only to prevent
// concurrency issues.
func cacheGet(store cacheableIndexStore, name string, fallback Index) Index {
	return defaultIndexCache.cacheGet(store, name, fallback)
}

// cachePut will store an index in the cache
func cachePut(store cacheableIndexStore, name string, index Index) {
	defaultIndexCache.cachePut(store, name, index)
}

func (c *indexCache) cacheGet(store cacheableIndexStore, name string, fallback Index) Index {
	c.RLock()
	defer c.RUnlock()
	key := indexCacheKey{
		storeKey:  store.StoreKey(),
		indexName: name,
	}
	if index, ok := c.indexes[key]; ok {
		vlog.Printf("%s: loaded from cache key=%v", name, key)
		return index
	} else {
		vlog.Printf("%s: not in cache key=%v", name, key)
		return fallback
	}
}

func (c *indexCache) cachePut(store cacheableIndexStore, name string, index Index) {
	key := indexCacheKey{
		storeKey:  store.StoreKey(),
		indexName: name,
	}

	// We don't need to store something in the cache that is already
	// stored
	c.RLock()
	if _, ok := c.indexes[key]; ok {
		// NOP we already have it cached
		c.RUnlock()
		return
	}
	c.RUnlock()

	vlog.Printf("%s: updating cache key=%v", name, key)
	// Update cache
	c.Lock()
	defer c.Unlock()
	c.indexes[key] = index
}
