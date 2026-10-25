package routes

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const canonicalRemoteLogCacheLimit = 1024
const canonicalRemoteLogFailureTTL = 15 * time.Minute

func canonicalLogEntryTTL(entry canonicalLogCacheEntry) time.Duration {
	if entry.failed {
		return canonicalRemoteLogFailureTTL
	}
	return remoteSyncThreshold
}

type canonicalLogCacheEntry struct {
	hint, recordUpdated string
	failed              bool
	checked             time.Time
}

type canonicalLogCache struct {
	sync.Mutex
	entries map[string]canonicalLogCacheEntry
}

var canonicalRemoteLogCache = &canonicalLogCache{}

func canonicalLogCacheKey(app core.App, trail *core.Record, iri string) string {
	parent := trail.Id
	if parent == "" {
		parent = trail.GetString("iri")
	}
	return app.DataDir() + "\x00" + parent + "\x00" + iri
}

func (cache *canonicalLogCache) skip(key, hint, recordUpdated string, exists bool) bool {
	cache.Lock()
	defer cache.Unlock()
	entry, found := cache.entries[key]
	if !found || time.Since(entry.checked) >= canonicalLogEntryTTL(entry) {
		delete(cache.entries, key)
		return false
	}
	return entry.failed || (entry.hint == hint && exists && entry.recordUpdated == recordUpdated)
}

func (cache *canonicalLogCache) remember(key string, entry canonicalLogCacheEntry) {
	cache.Lock()
	defer cache.Unlock()
	if cache.entries == nil {
		cache.entries = map[string]canonicalLogCacheEntry{}
	}
	if len(cache.entries) >= canonicalRemoteLogCacheLimit {
		oldestKey := ""
		var oldest time.Time
		for candidate, value := range cache.entries {
			if time.Since(value.checked) >= canonicalLogEntryTTL(value) {
				delete(cache.entries, candidate)
				continue
			}
			if oldestKey == "" || value.checked.Before(oldest) {
				oldestKey, oldest = candidate, value.checked
			}
		}
		if len(cache.entries) >= canonicalRemoteLogCacheLimit {
			delete(cache.entries, oldestKey)
		}
	}
	entry.checked = time.Now()
	cache.entries[key] = entry
}

func (cache *canonicalLogCache) failure(key, hint string) {
	cache.remember(key, canonicalLogCacheEntry{hint: hint, failed: true})
}

func (cache *canonicalLogCache) success(key, hint, updated string) {
	cache.remember(key, canonicalLogCacheEntry{hint: hint, recordUpdated: updated})
}

func canonicalLogUpdatedHint(data map[string]any) string {
	hint, _ := data["updated"].(string)
	if len(hint) > 128 {
		digest := sha256.Sum256([]byte(hint))
		return fmt.Sprintf("%x", digest)
	}
	return hint
}

func (cache *canonicalLogCache) invalidate(key string) {
	cache.Lock()
	defer cache.Unlock()
	delete(cache.entries, key)
}

func (cache *canonicalLogCache) successAfterCommit(app core.App, key, hint, updated string) {
	if tx := app.TxInfo(); tx != nil {
		tx.OnComplete(func(err error) error {
			if err == nil {
				cache.success(key, hint, updated)
			} else {
				cache.invalidate(key)
			}
			return nil
		})
		return
	}
	cache.success(key, hint, updated)
}

func (cache *canonicalLogCache) invalidateAfterRollback(app core.App, key string) {
	if tx := app.TxInfo(); tx != nil {
		tx.OnComplete(func(err error) error {
			if err != nil {
				cache.invalidate(key)
			}
			return nil
		})
	}
}

func (cache *canonicalLogCache) previousHint(key string) (string, bool) {
	cache.Lock()
	defer cache.Unlock()
	entry, found := cache.entries[key]
	return entry.hint, found && !entry.failed
}
