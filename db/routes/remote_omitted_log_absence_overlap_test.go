package routes

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteOmitted404OverlappingProbesCannotResurrectResetEvidence(t *testing.T) {
	for _, scenario := range []string{"verified-200", "listed-cache-hit", "private-cache-hit", "other-candidate-deletes", "both-qualified"} {
		t.Run(scenario, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			f.fetch()
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			releaseProbe := func() { once.Do(func() { close(release) }) }
			t.Cleanup(releaseProbe)
			f.beforeProbe = func(index int32) error {
				if index == 2 {
					close(entered)
					select {
					case <-release:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			}
			first := make(chan map[string]canonicalRemoteSummitLog, 1)
			go func() { first <- f.fetch() }()
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			switch scenario {
			case "verified-200":
				f.status.Store(http.StatusOK)
			case "listed-cache-hit":
				f.entries = []any{map[string]any{"id": "copy", "iri": f.iri}}
				canonicalRemoteLogCache.failure(f.key, "canonical-absence")
			case "private-cache-hit":
				f.public = false
				canonicalRemoteLogCache.failure(f.key, "canonical-absence")
			}
			second := f.fetch()
			if scenario != "both-qualified" {
				if err := f.apply(second); err != nil {
					t.Fatal(err)
				}
			}
			releaseProbe()
			var candidate map[string]canonicalRemoteSummitLog
			select {
			case candidate = <-first:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			if scenario == "both-qualified" {
				if !candidate[f.iri].removed || !second[f.iri].removed {
					t.Fatal("overlapping valid observations failed to prepare candidates")
				}
			} else if len(candidate) != 0 {
				t.Fatal("a stale in-flight response recreated invalidated evidence")
			}
			if err := f.apply(candidate); err != nil {
				t.Fatal(err)
			}
			if err := f.apply(second); err != nil {
				t.Fatal("an overlapping candidate failed after another transaction committed:", err)
			}
			f.assertExists(scenario != "other-candidate-deletes" && scenario != "both-qualified")
			if f.proof().entry != nil {
				t.Fatal("a stale response resurrected evidence after reset or committed deletion")
			}
		})
	}
}

func TestRemoteOmitted404PreparedCandidatesRequireCurrentGeneration(t *testing.T) {
	for _, counterevidence := range []string{"listed", "private", "200"} {
		t.Run(counterevidence, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			f.fetch()
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			candidate := f.fetch()
			if !candidate[f.iri].removed {
				t.Fatal("qualifying observation did not prepare a candidate")
			}
			switch counterevidence {
			case "listed":
				f.entries = []any{map[string]any{"id": "copy", "iri": f.iri}}
				canonicalRemoteLogCache.failure(f.key, "canonical-absence")
			case "private":
				f.public = false
				canonicalRemoteLogCache.failure(f.key, "canonical-absence")
			case "200":
				f.status.Store(http.StatusOK)
			}
			f.fetch()
			if err := f.apply(candidate); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
		})
	}
}

func TestRemoteOmitted404EvidenceDoesNotCrossApps(t *testing.T) {
	first, second := newOmitted404Fixture(t), newOmitted404Fixture(t)
	first.fetch()
	first.ageFirst(remoteSyncThreshold + time.Second)
	first.expireGeneric()
	if len(second.fetch()) != 0 {
		t.Fatal("another app inherited the first app's observation")
	}
	if err := first.apply(first.fetch()); err != nil {
		t.Fatal(err)
	}
	first.assertExists(false)
	second.assertExists(true)
	if second.proof().entry == nil {
		t.Fatal("one app's commit forgot another app's evidence")
	}
}

func TestRemoteOmitted404RejectsForeignCandidateNamespace(t *testing.T) {
	f := newOmitted404Fixture(t)
	f.fetch()
	f.ageFirst(remoteSyncThreshold + time.Second)
	f.expireGeneric()
	candidate := f.fetch()
	confirmed := candidate[f.iri]
	if !confirmed.removed {
		t.Fatal("qualifying observation did not prepare a candidate")
	}
	otherApp := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	confirmed.cacheKey = canonicalLogCacheKey(otherApp, f.trail, f.iri)
	candidate[f.iri] = confirmed
	if err := f.apply(candidate); err != nil {
		t.Fatal(err)
	}
	f.assertExists(true)
	if !canonicalRemoteLogAbsenceCache.ready(confirmed.absenceProbe, confirmed.storedSnapshot) {
		t.Fatal("rejecting a foreign namespace consumed another generation's evidence")
	}
}

func TestRemoteOmitted404OnlyAttemptedJobsReserveEvidence(t *testing.T) {
	f := newOmitted404Fixture(t)
	f.fetch()
	probe := f.proof()
	err := f.app.RunInTransaction(func(tx core.App) error {
		for i := 0; i < canonicalRemoteLogCacheLimit+8; i++ {
			log := core.NewRecord(f.children)
			log.Load(map[string]any{"iri": fmt.Sprintf("https://a.example:8443/api/v1/summit-log/unselected%d", i), "author": f.author.Id, "trail": f.trail.Id})
			if err := tx.Save(log); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	f.fetch()
	if f.requests.Load() != 1+canonicalRemoteRecordLimit || f.proof().entry != probe.entry {
		t.Fatal("unselected references displaced evidence or changed the request bound")
	}
	canonicalRemoteLogAbsenceCache.Lock()
	count := 0
	for key := range canonicalRemoteLogAbsenceCache.entries {
		if strings.HasPrefix(key, f.app.DataDir()+"\x00") {
			count++
		}
	}
	canonicalRemoteLogAbsenceCache.Unlock()
	if count != 1+canonicalRemoteRecordLimit {
		t.Fatalf("evidence reserved without an HTTP attempt: entries=%d", count)
	}
	fetchCanonicalRemoteSummitLogsWithin(f.app, t.Context(), f.document(), f.trail, "https://a.example", true, 0)
	if f.requests.Load() != 1+canonicalRemoteRecordLimit || f.proof().entry != probe.entry {
		t.Fatal("expired-budget jobs allocated evidence or attempted HTTP")
	}
}
