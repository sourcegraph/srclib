package store

import "testing"

type mockCacheableIndexStore struct{}

func (m *mockCacheableIndexStore) StoreKey() interface{} { return "test" }

type mockIndex struct {
	id int
}

func (m *mockIndex) Ready() bool              { return true }
func (m *mockIndex) Covers(f interface{}) int { return 1 }

func TestCache(t *testing.T) {
	store := &mockCacheableIndexStore{}
	index1 := &mockIndex{123}
	index2 := &mockIndex{456}

	// empty cache, should use fallback
	if index1 != cacheGet(store, "test_index", index1) {
		t.Errorf("cacheGet expected to use fallback value")
	}

	// We put in 2, and get with fallback 1. We should get back 2
	cachePut(store, "test_index", index2)
	if index2 != cacheGet(store, "test_index", index1) {
		t.Errorf("cachePut followed by cacheGet returns different results")
	}

	// Same test, but on different key with indexes swapped
	if index2 != cacheGet(store, "test_index_2", index2) {
		t.Errorf("cacheGet expected to use fallback value")
	}
	cachePut(store, "test_index_2", index1)
	if index1 != cacheGet(store, "test_index_2", index2) {
		t.Errorf("cachePut followed by cacheGet returns different results")
	}

}
