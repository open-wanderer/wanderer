package routes

import (
	"sync"
	"time"
)

type canonicalRotation struct {
	next int
	used time.Time
}

var canonicalRotations = struct {
	sync.Mutex
	entries map[string]canonicalRotation
}{entries: map[string]canonicalRotation{}}

func canonicalRotationStart(key string, size, take int) int {
	canonicalRotations.Lock()
	defer canonicalRotations.Unlock()
	now := time.Now()
	for key, value := range canonicalRotations.entries {
		if now.Sub(value.used) >= 24*time.Hour {
			delete(canonicalRotations.entries, key)
		}
	}
	if len(canonicalRotations.entries) >= canonicalRemoteLogCacheLimit {
		oldest := ""
		var used time.Time
		for key, value := range canonicalRotations.entries {
			if oldest == "" || value.used.Before(used) {
				oldest, used = key, value.used
			}
		}
		delete(canonicalRotations.entries, oldest)
	}
	start := canonicalRotations.entries[key].next % size
	canonicalRotations.entries[key] = canonicalRotation{next: (start + take) % size, used: now}
	return start
}

// New and changed references lead while a rotating refresh window keeps
// the remaining known entries current.
func selectCanonicalLogRequests(key string, priorities [2][]canonicalLogRequest) []canonicalLogRequest {
	return selectCanonicalRemoteRequests(key, priorities[:])
}

func selectCanonicalRemoteRequests(key string, priorities [][]canonicalLogRequest) []canonicalLogRequest {
	selected := make([]canonicalLogRequest, 0, canonicalRemoteRecordLimit)
	for priority, jobs := range priorities {
		if len(jobs) == 0 {
			continue
		}
		remaining := canonicalRemoteRecordLimit - len(selected)
		reserve := 0
		for _, later := range priorities[priority+1:] {
			if len(later) > 0 {
				reserve++
			}
		}
		take := min(len(jobs), remaining-reserve)
		if take <= 0 {
			continue
		}
		start := canonicalRotationStart(key+string(rune('0'+priority)), len(jobs), take)
		for offset := 0; offset < take; offset++ {
			selected = append(selected, jobs[(start+offset)%len(jobs)])
		}
	}
	return selected
}
