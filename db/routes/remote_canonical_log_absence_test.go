package routes

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func testCanonicalLogAbsenceSnapshot() canonicalLogAbsenceSnapshot {
	return canonicalLogAbsenceSnapshot{
		recordID: "cached-log", created: "created", updated: "version-one",
		author: "author", trail: "trail", trailIRI: "https://parent.example/api/v1/trail/trail",
	}
}

func confirmCanonicalLogAbsence(t *testing.T, cache *canonicalLogAbsenceCache, key string, snapshot canonicalLogAbsenceSnapshot, at time.Time) canonicalLogAbsenceProbe {
	t.Helper()
	probe := cache.begin(key, snapshot, at)
	if cache.observe404(probe, at) || !cache.observe404(probe, at.Add(remoteSyncThreshold)) {
		t.Fatal("two sufficiently separated observations did not confirm absence")
	}
	return probe
}

func TestCanonicalLogAbsenceUsesConfiguredIntervalWithoutExpiringFirst404(t *testing.T) {
	for _, interval := range []time.Duration{time.Minute, 37 * time.Minute} {
		t.Run(interval.String(), func(t *testing.T) {
			previousInterval := remoteSyncThreshold
			remoteSyncThreshold = interval
			t.Cleanup(func() { remoteSyncThreshold = previousInterval })
			cache := &canonicalLogAbsenceCache{}
			snapshot := testCanonicalLogAbsenceSnapshot()
			// The zero time is a valid explicit observation time too.
			first := time.Time{}
			probe := cache.begin("log", snapshot, first)
			if cache.ready(probe, snapshot) || cache.observe404(probe, first) || cache.ready(probe, snapshot) {
				t.Fatal("a first observation authorized deletion")
			}
			elapsedTimes := []time.Duration{-time.Second, 0, interval / 2, interval - time.Nanosecond}
			if interval > canonicalRemoteLogFailureTTL {
				elapsedTimes = append(elapsedTimes, canonicalRemoteLogFailureTTL+time.Minute)
			}
			for _, elapsed := range elapsedTimes {
				probe = cache.begin("log", snapshot, first.Add(elapsed))
				if cache.observe404(probe, first.Add(elapsed)) || cache.ready(probe, snapshot) {
					t.Fatalf("absence confirmed too early, after %s", elapsed)
				}
			}
			if !cache.observe404(probe, first.Add(interval)) || !cache.ready(probe, snapshot) {
				t.Fatal("the exact configured interval did not confirm absence")
			}
			if probe.entry.first404 != first || probe.entry.observations != 2 {
				t.Fatal("early observations moved the first timestamp or the evidence did not saturate")
			}
			for i := 0; i < 300; i++ {
				if !cache.observe404(probe, first.Add(time.Hour)) {
					t.Fatal("confirmed evidence was lost on another observation")
				}
			}
			if probe.entry.observations != 2 {
				t.Fatal("observation state did not stay bounded")
			}
			// Preparing a candidate does not imply that application was reached.
			retry := cache.begin("log", snapshot, first.Add(time.Hour))
			if retry.entry != probe.entry || !cache.ready(retry, snapshot) {
				t.Fatal("an unapplied deletion candidate consumed its evidence")
			}
		})
	}
}

func TestCanonicalLogAbsenceResetInvalidatesInFlightGenerations(t *testing.T) {
	cache := &canonicalLogAbsenceCache{}
	snapshot := testCanonicalLogAbsenceSnapshot()
	at := time.Unix(1700000000, 0)
	if cache.observe404(canonicalLogAbsenceProbe{}, at) || cache.ready(canonicalLogAbsenceProbe{}, snapshot) {
		t.Fatal("an empty probe authorized deletion")
	}
	probe := confirmCanonicalLogAbsence(t, cache, "log", snapshot, at)
	cache.reset("log")
	if cache.ready(probe, snapshot) || cache.observe404(probe, at.Add(2*remoteSyncThreshold)) {
		t.Fatal("reset evidence was resurrected by a delayed response")
	}
	replacement := cache.begin("log", snapshot, at.Add(2*remoteSyncThreshold))
	if replacement.entry == probe.entry || cache.observe404(replacement, at.Add(2*remoteSyncThreshold)) {
		t.Fatal("a new generation inherited old observations")
	}
	cache.forget(probe)
	if cache.begin("log", snapshot, at).entry != replacement.entry {
		t.Fatal("forgetting a stale generation cleared its replacement")
	}
}

func TestCanonicalLogAbsenceRejectsEveryChangedSnapshotField(t *testing.T) {
	for _, field := range []string{"recordID", "created", "updated", "author", "trail", "trailIRI"} {
		t.Run(field, func(t *testing.T) {
			cache := &canonicalLogAbsenceCache{}
			snapshot := testCanonicalLogAbsenceSnapshot()
			at := time.Unix(1700000000, 0)
			cache.resetIfChanged("missing", snapshot)
			if len(cache.entries) != 0 {
				t.Fatal("checking a version allocated evidence for an unselected probe")
			}
			probe := confirmCanonicalLogAbsence(t, cache, "log", snapshot, at)
			cache.resetIfChanged("log", snapshot)
			if !cache.ready(probe, snapshot) {
				t.Fatal("an unchanged snapshot lost its evidence")
			}
			changed := snapshot
			switch field {
			case "recordID":
				changed.recordID += "-replacement"
			case "created":
				changed.created += "-replacement"
			case "updated":
				changed.updated += "-newer"
			case "author":
				changed.author += "-other"
			case "trail":
				changed.trail += "-other"
			case "trailIRI":
				changed.trailIRI += "-other"
			}
			cache.resetIfChanged("log", changed)
			if cache.observe404(probe, at.Add(2*remoteSyncThreshold)) {
				t.Fatal("a changed snapshot retained in-flight evidence")
			}
			replacement := confirmCanonicalLogAbsence(t, cache, "log", changed, at)
			if cache.ready(replacement, snapshot) || cache.ready(replacement, changed) {
				t.Fatal("a snapshot mismatch at apply time did not discard evidence")
			}
			probe = confirmCanonicalLogAbsence(t, cache, "log", snapshot, at)
			replacement = cache.begin("log", changed, at)
			if cache.ready(probe, snapshot) || cache.observe404(probe, at.Add(2*remoteSyncThreshold)) || cache.observe404(replacement, at) {
				t.Fatal("beginning a changed record version inherited or retained proof")
			}
		})
	}
}

func TestCanonicalLogAbsenceSnapshotExcludesParentUpdated(t *testing.T) {
	logs := core.NewBaseCollection("summit_logs")
	for _, field := range []string{"created", "updated", "author", "trail"} {
		logs.Fields.Add(&core.TextField{Name: field})
	}
	log := core.NewRecord(logs)
	log.Id = "cached-log"
	log.Load(map[string]any{"created": "created", "updated": "version-one", "author": "author", "trail": "trail"})
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "updated"})
	trail := core.NewRecord(trails)
	trail.Id = "trail"
	trail.Load(map[string]any{"iri": "https://parent.example/api/v1/trail/trail", "updated": "before-sync"})
	want := testCanonicalLogAbsenceSnapshot()
	if got := canonicalLogAbsenceSnapshotOf(log, trail); got != want {
		t.Fatalf("snapshot = %+v, want %+v", got, want)
	}
	trail.Set("updated", "after-sync")
	if canonicalLogAbsenceSnapshotOf(log, trail) != want {
		t.Fatal("ordinary parent refresh invalidated the cached-log snapshot")
	}
}

func TestCanonicalLogAbsenceNamespacesAndTrailResetAreIsolated(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	otherApp := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.TextField{Name: "iri"})
	trail := func(id, iri string) *core.Record {
		record := core.NewRecord(trails)
		record.Id = id
		record.Set("iri", iri)
		return record
	}
	parent := trail("parent", "https://parent.example/api/v1/trail/parent")
	otherParent := trail("parent-suffix", "https://parent.example/api/v1/trail/other")
	unsaved := trail("", "https://parent.example/api/v1/trail/unsaved")
	otherUnsaved := trail("", "https://parent.example/api/v1/trail/other-unsaved")
	iri := "https://third.example/api/v1/summit-log/log"
	keys := []string{
		canonicalLogCacheKey(app, parent, iri),
		canonicalLogCacheKey(otherApp, parent, iri),
		canonicalLogCacheKey(app, otherParent, iri),
		canonicalLogCacheKey(app, unsaved, iri),
		canonicalLogCacheKey(app, otherUnsaved, iri),
		canonicalLogCacheKey(app, parent, iri+"-other"),
	}
	cache := &canonicalLogAbsenceCache{}
	snapshot := testCanonicalLogAbsenceSnapshot()
	at := time.Unix(1700000000, 0)
	probes := make([]canonicalLogAbsenceProbe, len(keys))
	for i, key := range keys {
		probes[i] = confirmCanonicalLogAbsence(t, cache, key, snapshot, at)
	}
	if len(cache.entries) != len(keys) {
		t.Fatal("app, trail or log namespaces shared evidence")
	}
	cache.resetTrail(app, parent)
	for i, probe := range probes {
		if got, want := cache.ready(probe, snapshot), i != 0 && i != 5; got != want {
			t.Fatalf("trail reset changed namespace %d: ready=%t want=%t", i, got, want)
		}
	}
	cache.resetTrail(app, unsaved)
	if cache.ready(probes[3], snapshot) || !cache.ready(probes[4], snapshot) {
		t.Fatal("unsaved trail identity fallback crossed namespaces")
	}
}

func TestCanonicalLogAbsenceUnselectedResetChecksDoNotAllocateOrEvict(t *testing.T) {
	cache := &canonicalLogAbsenceCache{}
	snapshot := testCanonicalLogAbsenceSnapshot()
	at := time.Unix(1700000000, 0)
	probe := cache.begin("attempted", snapshot, at)
	if cache.observe404(probe, at) {
		t.Fatal("a first observation authorized deletion")
	}
	// Stored omissions may exceed both the fresh expand limit and cache capacity.
	// Enumeration checks versions and counterevidence without reserving probes.
	for i := 0; i < 4*canonicalRemoteLogCacheLimit; i++ {
		cache.resetIfChanged(fmt.Sprintf("unselected-omission/%d", i), snapshot)
		cache.reset(fmt.Sprintf("listed-again/%d", i))
		cache.resetIfChanged("attempted", snapshot)
	}
	if len(cache.entries) != 1 || cache.entries["attempted"] != probe.entry {
		t.Fatal("unselected-reference checks allocated entries or evicted real evidence")
	}
	if !cache.observe404(probe, at.Add(remoteSyncThreshold)) || !cache.ready(probe, snapshot) {
		t.Fatal("unselected-reference checks lost the first actual observation")
	}
}

func TestCanonicalLogAbsenceCapacityEvictionAndRestartOnlyDelay(t *testing.T) {
	cache := &canonicalLogAbsenceCache{}
	snapshot := testCanonicalLogAbsenceSnapshot()
	at := time.Unix(1700000000, 0)
	oldest := confirmCanonicalLogAbsence(t, cache, "oldest", snapshot, at)
	for i := 1; i < canonicalRemoteLogCacheLimit; i++ {
		cache.begin(fmt.Sprint(i), snapshot, at.Add(remoteSyncThreshold+time.Duration(i)*time.Second))
	}
	lastKey := fmt.Sprint(canonicalRemoteLogCacheLimit - 1)
	changed := snapshot
	changed.updated += "-newer"
	cache.begin(lastKey, changed, at.Add(2*remoteSyncThreshold))
	if len(cache.entries) != canonicalRemoteLogCacheLimit || !cache.ready(oldest, snapshot) {
		t.Fatal("replacing an existing entry unnecessarily evicted other evidence")
	}
	cache.begin("overflow", snapshot, at.Add(3*remoteSyncThreshold))
	if len(cache.entries) != canonicalRemoteLogCacheLimit || cache.ready(oldest, snapshot) || cache.observe404(oldest, at.Add(3*remoteSyncThreshold)) {
		t.Fatal("capacity was exceeded or an evicted generation still authorized deletion")
	}
	restarted := &canonicalLogAbsenceCache{}
	for _, current := range []*canonicalLogAbsenceCache{cache, restarted} {
		probe := current.begin("oldest", snapshot, at.Add(3*remoteSyncThreshold))
		if current.ready(probe, snapshot) || current.observe404(probe, at.Add(3*remoteSyncThreshold)) || current.ready(oldest, snapshot) {
			t.Fatal("eviction or restart inherited old proof")
		}
		if !current.observe404(probe, at.Add(4*remoteSyncThreshold)) {
			t.Fatal("cache loss prevented fresh, sufficiently separated observations")
		}
	}
}

func TestCanonicalLogAbsenceForgetWaitsForCommitAndMatchesGeneration(t *testing.T) {
	f := newRemoteSyncFixture(t)
	snapshot := testCanonicalLogAbsenceSnapshot()
	at := time.Unix(1700000000, 0)
	sentinel := errors.New("later transaction failure")
	for _, scenario := range []string{"commit", "rollback", "replacement"} {
		t.Run(scenario, func(t *testing.T) {
			cache := &canonicalLogAbsenceCache{}
			probe := confirmCanonicalLogAbsence(t, cache, "log", snapshot, at)
			var replacement canonicalLogAbsenceProbe
			err := f.app.RunInTransaction(func(tx core.App) error {
				cache.forgetAfterCommit(tx, probe)
				if !cache.ready(probe, snapshot) {
					t.Error("evidence was consumed before commit")
				}
				if scenario == "rollback" {
					return sentinel
				}
				if scenario == "replacement" {
					cache.reset("log")
					replacement = cache.begin("log", snapshot, at)
					if cache.observe404(replacement, at) {
						t.Error("replacement inherited proof before commit")
					}
				}
				return nil
			})
			if scenario == "rollback" {
				if !errors.Is(err, sentinel) || !cache.ready(probe, snapshot) || !cache.observe404(probe, at.Add(remoteSyncThreshold)) {
					t.Fatalf("rollback consumed evidence or prevented immediate retry: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cache.ready(probe, snapshot) {
				t.Fatal("a completed transaction retained its old proof")
			}
			if scenario == "replacement" && !cache.observe404(replacement, at.Add(remoteSyncThreshold)) {
				t.Fatal("an older commit forgot its replacement generation")
			}
		})
	}
	cache := &canonicalLogAbsenceCache{}
	probe := confirmCanonicalLogAbsence(t, cache, "log", snapshot, at)
	cache.forgetAfterCommit(f.app, probe)
	if cache.ready(probe, snapshot) {
		t.Fatal("nontransactional successful deletion retained proof")
	}
}

func TestCanonicalLogAbsenceConcurrentObservationsResetsAndCapacity(t *testing.T) {
	cache := &canonicalLogAbsenceCache{}
	snapshot := testCanonicalLogAbsenceSnapshot()
	at := time.Unix(1700000000, 0)
	shared := cache.begin("shared", snapshot, at)
	cache.observe404(shared, at)
	var wait sync.WaitGroup
	start := make(chan struct{})
	for worker := 0; worker < 32; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			for i := 0; i < 50; i++ {
				probe := cache.begin("shared", snapshot, at)
				if cache.observe404(probe, at.Add(remoteSyncThreshold-time.Nanosecond)) || cache.ready(probe, snapshot) {
					t.Error("concurrent early observations authorized deletion")
				}
			}
		}()
	}
	close(start)
	wait.Wait()
	if !cache.observe404(shared, at.Add(remoteSyncThreshold)) {
		t.Fatal("concurrent early observations advanced the first timestamp")
	}
	for worker := 0; worker < 32; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for i := 0; i < 96; i++ {
				key := fmt.Sprintf("%d/%d", worker, i)
				probe := cache.begin(key, snapshot, at)
				if cache.observe404(probe, at) || cache.ready(probe, snapshot) {
					t.Error("a concurrent first observation authorized deletion")
				}
				switch i % 4 {
				case 0:
					cache.reset(key)
				case 1:
					changed := snapshot
					changed.updated += "-newer"
					cache.resetIfChanged(key, changed)
				}
			}
		}()
	}
	wait.Wait()
	if len(cache.entries) > canonicalRemoteLogCacheLimit {
		t.Fatal("concurrent probes exceeded the memory bound")
	}
	cache.reset("shared")
	if cache.observe404(shared, at.Add(2*remoteSyncThreshold)) {
		t.Fatal("a stale generation survived concurrent cache changes")
	}
}
