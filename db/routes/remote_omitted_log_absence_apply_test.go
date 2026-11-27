package routes

import (
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteOmitted404KeepsRecordChangedDuringQualifyingProbe(t *testing.T) {
	for _, change := range []string{"updated", "author", "local-author", "local-identity", "author-origin", "trail", "replacement", "parent-iri"} {
		t.Run(change, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			if len(f.fetch()) != 0 {
				t.Fatal("first 404 produced a deletion candidate")
			}
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			f.beforeProbe = func(index int32) error {
				if index != 2 {
					return nil
				}
				switch change {
				case "updated":
					f.log.Set("text", "Newer cached data")
					return saveOmitted404Newer(f.app, f.log)
				case "author":
					author := core.NewRecord(f.actors)
					author.Set("iri", "https://a.example:8443/users/erin")
					if err := f.app.Save(author); err != nil {
						return err
					}
					f.log.Set("author", author.Id)
				case "local-author":
					f.author.Set("is_local", true)
					return f.app.Save(f.author)
				case "local-identity":
					f.log.Set("iri", "https://b.example/api/v1/summit-log/"+f.log.Id)
				case "author-origin":
					f.author.Set("iri", "https://fourth.example/users/carol")
					return f.app.Save(f.author)
				case "trail":
					parent := core.NewRecord(f.trails)
					parent.Load(map[string]any{"author": f.remoteActor.Id, "iri": "https://a.example/api/v1/trail/other"})
					if err := f.app.Save(parent); err != nil {
						return err
					}
					f.log.Set("trail", parent.Id)
				case "replacement":
					if err := f.app.Delete(f.log); err != nil {
						return err
					}
					replacement := core.NewRecord(f.children)
					replacement.Load(map[string]any{"iri": f.iri, "author": f.author.Id, "trail": f.trail.Id, "text": "Replacement"})
					if err := f.app.Save(replacement); err != nil {
						return err
					}
					f.log = replacement
					return nil
				case "parent-iri":
					current, err := f.app.FindRecordById(f.trails, f.trail.Id)
					if err != nil {
						return err
					}
					current.Set("iri", "https://a.example/api/v1/trail/replacement")
					return f.app.Save(current)
				}
				return f.app.Save(f.log)
			}
			candidate := f.fetch()
			if !candidate[f.iri].removed {
				t.Fatal("qualifying 404 did not reach the transactional guard")
			}
			if err := f.apply(candidate); err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			if f.proof().entry != nil {
				t.Fatal("changed record eligibility retained its old evidence")
			}
		})
	}
}

func TestRemoteOmitted404RollbackDeleteFailureAndUnappliedCandidatesRetry(t *testing.T) {
	for _, scenario := range []string{"delete-failure", "rollback", "unapplied"} {
		t.Run(scenario, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			f.fetch()
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			candidate := f.fetch()
			confirmed := candidate[f.iri]
			if !confirmed.removed || !canonicalRemoteLogAbsenceCache.ready(confirmed.absenceProbe, confirmed.storedSnapshot) {
				t.Fatal("qualified evidence was not prepared")
			}
			sentinel := errors.New("database or later transaction failure")
			switch scenario {
			case "delete-failure":
				handler := f.app.OnRecordDelete("summit_logs").BindFunc(func(*core.RecordEvent) error { return sentinel })
				err := f.apply(candidate)
				f.app.OnRecordDelete("summit_logs").Unbind(handler)
				if !errors.Is(err, sentinel) {
					t.Fatal(err)
				}
			case "rollback":
				err := f.app.RunInTransaction(func(tx core.App) error {
					if err := applyOmittedRemoteSummitLogs(tx, f.trail, candidate, nil); err != nil {
						return err
					}
					// A concurrent failure entry must be invalidated on rollback too.
					canonicalRemoteLogCache.failure(f.key, "canonical-absence")
					return sentinel
				})
				if !errors.Is(err, sentinel) {
					t.Fatal(err)
				}
			}
			f.assertExists(true)
			if !canonicalRemoteLogAbsenceCache.ready(confirmed.absenceProbe, confirmed.storedSnapshot) || canonicalRemoteLogCache.skip(f.key, "canonical-absence", f.log.GetString("updated"), true) {
				t.Fatal("failed/unapplied deletion consumed evidence or suppressed retry")
			}
			// Retry immediately: do not age either cache again.
			candidate = f.fetch()
			if !candidate[f.iri].removed {
				t.Fatal("retry required an extra interval after already qualified evidence")
			}
			if err := f.apply(candidate); err != nil {
				t.Fatal(err)
			}
			f.assertExists(false)
			if f.proof().entry != nil {
				t.Fatal("successful retry did not forget its matching generation")
			}
		})
	}
}

func TestRemoteOmitted404FullSyncEarlyReturnsRetainRetryableEvidence(t *testing.T) {
	for _, scenario := range []string{"newer-trail", "waypoint-failure"} {
		t.Run(scenario, func(t *testing.T) {
			f := newOmitted404Fixture(t)
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.ageFirst(remoteSyncThreshold + time.Second)
			f.expireGeneric()
			sentinel := errors.New("waypoint batch failure")
			if scenario == "newer-trail" {
				f.beforeProbe = func(index int32) error {
					if index != 2 {
						return nil
					}
					current, err := f.app.FindRecordById(f.trails, f.trail.Id)
					if err != nil {
						return err
					}
					current.Set("name", "Newer parent data")
					return saveOmitted404Newer(f.app, current)
				}
			} else {
				f.points = []any{map[string]any{"id": "waypoint", "name": "Waypoint"}}
			}
			fail := true
			handler := f.app.OnRecordCreateExecute("waypoints").BindFunc(func(e *core.RecordEvent) error {
				if fail {
					return sentinel
				}
				return e.Next()
			})
			t.Cleanup(func() { f.app.OnRecordCreateExecute("waypoints").Unbind(handler) })
			err := f.refresh()
			if scenario == "waypoint-failure" && !errors.Is(err, sentinel) || scenario == "newer-trail" && err != nil {
				t.Fatal(err)
			}
			f.assertExists(true)
			probe := f.proof()
			snapshot := canonicalLogAbsenceSnapshotOf(f.log, f.trail)
			if !canonicalRemoteLogAbsenceCache.ready(probe, snapshot) || canonicalRemoteLogCache.skip(f.key, "canonical-absence", f.log.GetString("updated"), true) {
				t.Fatal("an earlier full-sync return consumed or cached an unapplied deletion")
			}
			fail = false
			f.beforeProbe = nil
			if err := f.refresh(); err != nil {
				t.Fatal(err)
			}
			f.assertExists(false)
		})
	}
}
