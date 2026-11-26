package routes

import (
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// Evidence belongs to one cached record version, not the parent's updated
// timestamp: an ordinary trail refresh changes that timestamp itself.
type canonicalLogAbsenceSnapshot struct {
	recordID, created, updated, author, trail, trailIRI string
}

func canonicalLogAbsenceSnapshotOf(record, trail *core.Record) canonicalLogAbsenceSnapshot {
	return canonicalLogAbsenceSnapshot{
		recordID: record.Id,
		created:  record.GetString("created"),
		updated:  record.GetString("updated"),
		author:   record.GetString("author"),
		trail:    record.GetString("trail"),
		trailIRI: trail.GetString("iri"),
	}
}

type canonicalLogAbsenceEntry struct {
	snapshot     canonicalLogAbsenceSnapshot
	first404     time.Time
	used         time.Time
	observations uint8 // 0, first 404, or two sufficiently separated 404s
}

// Pointer identity invalidates in-flight probes when evidence is reset, replaced
// or evicted, without an unbounded generation counter or tombstone map.
type canonicalLogAbsenceProbe struct {
	key   string
	entry *canonicalLogAbsenceEntry
}

type canonicalLogAbsenceCache struct {
	sync.Mutex
	entries map[string]*canonicalLogAbsenceEntry
}

var canonicalRemoteLogAbsenceCache = &canonicalLogAbsenceCache{}

// Begin immediately before a selected omitted HTTP probe, never while
// enumerating references: unselected entries could evict real observations.
// Use canonicalLogCacheKey's app/trail/log namespace. Unlike the generic failure
// cache, first-404 evidence has no TTL; eviction or restart only delays proof.
func (cache *canonicalLogAbsenceCache) begin(key string, snapshot canonicalLogAbsenceSnapshot, now time.Time) canonicalLogAbsenceProbe {
	cache.Lock()
	defer cache.Unlock()
	entry := cache.entries[key]
	if entry == nil || entry.snapshot != snapshot {
		if cache.entries == nil {
			cache.entries = map[string]*canonicalLogAbsenceEntry{}
		}
		if entry == nil && len(cache.entries) >= canonicalRemoteLogCacheLimit {
			oldestKey := ""
			var oldest *canonicalLogAbsenceEntry
			for candidate, value := range cache.entries {
				if oldest == nil || value.used.Before(oldest.used) {
					oldestKey, oldest = candidate, value
				}
			}
			delete(cache.entries, oldestKey)
		}
		entry = &canonicalLogAbsenceEntry{snapshot: snapshot}
		cache.entries[key] = entry
	}
	entry.used = now
	return canonicalLogAbsenceProbe{key: key, entry: entry}
}

// Call only for an actual omitted-log 404 under a fresh public parent view.
// An early repeat never advances first404. Readiness is evidence, not a record
// deletion: it remains available for retry if the database transaction fails.
func (cache *canonicalLogAbsenceCache) observe404(probe canonicalLogAbsenceProbe, observedAt time.Time) bool {
	cache.Lock()
	defer cache.Unlock()
	entry := cache.entries[probe.key]
	if entry == nil || entry != probe.entry {
		return false
	}
	entry.used = observedAt
	if entry.observations == 0 {
		entry.first404, entry.observations = observedAt, 1
		return false
	}
	if observedAt.Sub(entry.first404) >= remoteSyncThreshold {
		entry.observations = 2
	}
	return entry.observations == 2
}

// Revalidate inside the deletion transaction against the current cached record.
// A different record version discards this generation rather than inheriting it.
func (cache *canonicalLogAbsenceCache) ready(probe canonicalLogAbsenceProbe, snapshot canonicalLogAbsenceSnapshot) bool {
	cache.Lock()
	defer cache.Unlock()
	entry := cache.entries[probe.key]
	if entry == nil || entry != probe.entry {
		return false
	}
	if entry.snapshot != snapshot {
		delete(cache.entries, probe.key)
		return false
	}
	return entry.observations == 2
}

func (cache *canonicalLogAbsenceCache) reset(key string) {
	cache.Lock()
	defer cache.Unlock()
	delete(cache.entries, key)
}

// Check record changes before generic-cache skips, without allocating evidence
// for the many omitted records that are not selected for a network probe.
func (cache *canonicalLogAbsenceCache) resetIfChanged(key string, snapshot canonicalLogAbsenceSnapshot) {
	cache.Lock()
	defer cache.Unlock()
	if entry := cache.entries[key]; entry != nil && entry.snapshot != snapshot {
		delete(cache.entries, key)
	}
}

func (cache *canonicalLogAbsenceCache) resetTrail(app core.App, trail *core.Record) {
	prefix := canonicalLogCacheKey(app, trail, "")
	cache.Lock()
	defer cache.Unlock()
	for key := range cache.entries {
		if strings.HasPrefix(key, prefix) {
			delete(cache.entries, key)
		}
	}
}

func (cache *canonicalLogAbsenceCache) forget(probe canonicalLogAbsenceProbe) {
	cache.Lock()
	defer cache.Unlock()
	if cache.entries[probe.key] == probe.entry {
		delete(cache.entries, probe.key)
	}
}

// Register only after successful deletion. Rollback retains the evidence, and
// an older transaction's commit must not clear a replacement generation.
func (cache *canonicalLogAbsenceCache) forgetAfterCommit(app core.App, probe canonicalLogAbsenceProbe) {
	if tx := app.TxInfo(); tx != nil {
		tx.OnComplete(func(err error) error {
			if err == nil {
				cache.forget(probe)
			}
			return nil
		})
		return
	}
	cache.forget(probe)
}
