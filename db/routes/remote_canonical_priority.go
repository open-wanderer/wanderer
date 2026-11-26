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

// New and changed listed logs lead, reserving progress for both refresh
// classes. Omitted existence probes get at most a third of the window when
// enough listed logs are due; either class may use the other's spare capacity.
func selectCanonicalLogRequests(key string, priorities [3][]canonicalLogRequest) []canonicalLogRequest {
	reserveOmitted, reserveKnown := min(1, len(priorities[1])), min(1, len(priorities[2]))
	newTake := min(len(priorities[0]), canonicalRemoteRecordLimit-reserveOmitted-reserveKnown)
	remaining := canonicalRemoteRecordLimit - newTake
	omittedTake := min(len(priorities[1]), canonicalRemoteRecordLimit/3, remaining-reserveKnown)
	knownTake := min(len(priorities[2]), remaining-omittedTake)
	omittedTake = min(len(priorities[1]), remaining-knownTake)

	windows := [3][]canonicalLogRequest{}
	for priority, take := range [3]int{newTake, omittedTake, knownTake} {
		if take == 0 {
			continue
		}
		jobs := priorities[priority]
		start := canonicalRotationStart(key+string(rune('0'+priority)), len(jobs), take)
		for offset := 0; offset < take; offset++ {
			windows[priority] = append(windows[priority], jobs[(start+offset)%len(jobs)])
		}
	}
	selected := make([]canonicalLogRequest, 0, canonicalRemoteRecordLimit)
	selected = append(selected, windows[0]...)
	// Queue known listed logs first and interleave the refresh classes so slow
	// omitted hosts cannot occupy every worker before any listed work starts.
	for offset := 0; offset < max(knownTake, omittedTake); offset++ {
		if offset < knownTake {
			selected = append(selected, windows[2][offset])
		}
		if offset < omittedTake {
			selected = append(selected, windows[1][offset])
		}
	}
	return selected
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
