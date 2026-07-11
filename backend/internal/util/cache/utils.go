package cache_utils

import (
	"encoding/json"
	"time"
)

const (
	DefaultCacheExpiry = 10 * time.Minute
)

// CacheUtil is a typed view over the process-local store. It JSON-encodes values
// and namespaces keys with prefix so unrelated callers never collide.
type CacheUtil[T any] struct {
	prefix string
	expiry time.Duration
}

func NewCacheUtil[T any](prefix string) *CacheUtil[T] {
	return &CacheUtil[T]{
		prefix: prefix,
		expiry: DefaultCacheExpiry,
	}
}

func (c *CacheUtil[T]) Get(key string) *T {
	data, exists := store.get(c.prefix + key)
	if !exists {
		return nil
	}

	var item T
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil
	}

	return &item
}

func (c *CacheUtil[T]) Set(key string, item *T) {
	c.SetWithExpiration(key, item, c.expiry)
}

func (c *CacheUtil[T]) SetWithExpiration(key string, item *T, expiry time.Duration) {
	data, err := json.Marshal(item)
	if err != nil {
		return
	}

	store.set(c.prefix+key, string(data), expiry)
}

func (c *CacheUtil[T]) GetAndDelete(key string) *T {
	data, exists := store.getAndDelete(c.prefix + key)
	if !exists {
		return nil
	}

	var item T
	if err := json.Unmarshal([]byte(data), &item); err != nil {
		return nil
	}

	return &item
}

func (c *CacheUtil[T]) Invalidate(key string) {
	store.delete(c.prefix + key)
}
