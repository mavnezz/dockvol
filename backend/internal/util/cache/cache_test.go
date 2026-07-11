package cache_utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ClearAllCache_AfterClear_CacheIsEmpty(t *testing.T) {
	storedKeys := []struct {
		prefix string
		key    string
		value  string
	}{
		{"test:user:", "user1", "John Doe"},
		{"test:user:", "user2", "Jane Smith"},
		{"test:session:", "session1", "abc123"},
		{"test:session:", "session2", "def456"},
		{"test:data:", "item1", "value1"},
	}

	for _, stored := range storedKeys {
		cacheUtil := NewCacheUtil[string](stored.prefix)
		cacheUtil.Set(stored.key, &stored.value)
	}

	for _, stored := range storedKeys {
		cacheUtil := NewCacheUtil[string](stored.prefix)
		retrieved := cacheUtil.Get(stored.key)
		assert.NotNil(t, retrieved, "key %s should exist before clearing", stored.prefix+stored.key)
		assert.Equal(t, stored.value, *retrieved, "retrieved value should match set value")
	}

	err := ClearAllCache()
	assert.NoError(t, err, "ClearAllCache should not return an error")

	for _, stored := range storedKeys {
		cacheUtil := NewCacheUtil[string](stored.prefix)
		retrieved := cacheUtil.Get(stored.key)
		assert.Nil(t, retrieved, "key %s should be deleted after clearing", stored.prefix+stored.key)
	}
}

func Test_SetWithExpiration_WhenTTLElapsed_ValueExpires(t *testing.T) {
	t.Cleanup(func() { _ = ClearAllCache() })

	cacheUtil := NewCacheUtil[string]("test:ttl:")

	key := "key1"
	value := "test value"

	cacheUtil.SetWithExpiration(key, &value, 40*time.Millisecond)

	retrieved := cacheUtil.Get(key)
	assert.NotNil(t, retrieved, "value should be stored before TTL elapses")
	assert.Equal(t, value, *retrieved, "retrieved value should match")

	assert.Eventually(t, func() bool {
		return cacheUtil.Get(key) == nil
	}, time.Second, 10*time.Millisecond, "value should expire after its TTL")
}

func Test_GetAndDelete_WhenCalledTwice_SecondCallReturnsNil(t *testing.T) {
	t.Cleanup(func() { _ = ClearAllCache() })

	cacheUtil := NewCacheUtil[string]("test:consume:")

	key := "single_use"
	value := "opaque token"
	cacheUtil.Set(key, &value)

	firstConsume := cacheUtil.GetAndDelete(key)
	assert.NotNil(t, firstConsume, "first consume should return the value")
	assert.Equal(t, value, *firstConsume)

	secondConsume := cacheUtil.GetAndDelete(key)
	assert.Nil(t, secondConsume, "second consume should return nil, proving single-use")
}
