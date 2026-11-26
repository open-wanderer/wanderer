package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type boundedCleanupCandidateApp struct {
	core.App
	t     *testing.T
	reads int
}

func (app *boundedCleanupCandidateApp) FindAllRecords(collection any, exprs ...dbx.Expression) ([]*core.Record, error) {
	app.t.Helper()
	query := app.App.RecordQuery(collection)
	for _, expr := range exprs {
		query.AndWhere(expr)
	}
	built := query.Build()
	if app.TxInfo() == nil || len(built.Params()) == 0 || len(built.Params()) > remoteSyncWriteBatch || !strings.Contains(built.SQL(), "WHERE `id`") {
		app.t.Fatalf("candidate reads must be bounded PK reads in the same transaction: %s (%d binds)", built.SQL(), len(built.Params()))
	}
	app.reads++
	return app.App.FindAllRecords(collection, exprs...)
}

func TestRemoteCleanupChunksKeepFrozenCandidatesAndRollbackAtomically(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		t.Run(target, func(t *testing.T) {
			f := newRemoteCleanupFixture(t, target)
			kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
			var candidates []*core.Record
			// Older caches can exceed even the current expand limit. Reads must
			// stay bounded while every delete remains in one transaction.
			for i := 0; i < remoteChildExpandLimit+remoteSyncWriteBatch+1; i++ {
				candidates = append(candidates, f.child(fmt.Sprintf("https://a.example/api/v1/%s/old%d", kind, i), f.remoteActor.Id, f.trail.Id))
			}
			stored, err := storedRemoteTrailChildIDs(f.app, f.trail.Id, "https://a.example", f.children.Name)
			if err != nil || len(stored) != len(candidates) {
				t.Fatalf("candidate snapshot count=%d error=%v", len(stored), err)
			}
			newArrival := f.child("https://a.example/api/v1/"+kind+"/new", f.remoteActor.Id, f.trail.Id)
			otherTrail := f.save(f.trails, map[string]any{"author": f.remoteActor.Id})
			candidates[0].Set("trail", otherTrail.Id)
			candidates[1].Set("author", f.localActor.Id)
			candidates[1].Set("iri", "https://a.example/api/v1/"+kind+"/"+candidates[1].Id)
			candidates[2].Set("iri", "https://elsewhere.example/api/v1/"+kind+"/reassigned")
			for _, record := range candidates[:3] {
				if err := f.app.Save(record); err != nil {
					t.Fatal(err)
				}
			}
			var attempted, afterSuccess int
			abort := errors.New("later cleanup chunk failed")
			f.app.OnRecordDelete(f.children.Name).BindFunc(func(e *core.RecordEvent) error {
				// Full current records, not projected snapshots, reach hooks.
				if e.Record.GetString(map[string]string{"waypoint": "name", "summit_log": "text"}[target]) != "Original" {
					t.Error("delete hook received a projected record")
				}
				attempted++
				if attempted == remoteSyncWriteBatch+1 {
					return abort
				}
				return e.Next()
			})
			f.app.OnRecordAfterDeleteSuccess(f.children.Name).BindFunc(func(e *core.RecordEvent) error {
				afterSuccess++
				return e.Next()
			})
			var failedReads, successfulReads int
			err = f.app.RunInTransaction(func(tx core.App) error {
				checked := &boundedCleanupCandidateApp{App: tx, t: t}
				err := pruneRemoteTrailChildren(checked, f.trail.Id, "https://a.example", f.children.Name, map[string]struct{}{}, stored, nil)
				failedReads = checked.reads
				return err
			})
			if !errors.Is(err, abort) || afterSuccess != 0 || failedReads < 2 {
				t.Fatalf("later chunk must roll back all deletions: error=%v afterSuccess=%d", err, afterSuccess)
			}
			for _, record := range candidates {
				if _, err := f.app.FindRecordById(f.children, record.Id); err != nil {
					t.Fatalf("committed a prefix of failed cleanup: %v", err)
				}
			}
			// The failure hook only aborts at its exact count; retry the same
			// frozen IDs and protect current ownership/identity and new arrivals.
			err = f.app.RunInTransaction(func(tx core.App) error {
				checked := &boundedCleanupCandidateApp{App: tx, t: t}
				err := pruneRemoteTrailChildren(checked, f.trail.Id, "https://a.example", f.children.Name, map[string]struct{}{}, stored, nil)
				successfulReads = checked.reads
				return err
			})
			if err != nil || afterSuccess != len(candidates)-3 || successfulReads != (len(stored)+remoteSyncWriteBatch-1)/remoteSyncWriteBatch {
				t.Fatalf("cleanup count=%d want=%d error=%v", afterSuccess, len(candidates)-3, err)
			}
			for _, record := range append(candidates[:3:3], newArrival) {
				if _, err := f.app.FindRecordById(f.children, record.Id); err != nil {
					t.Fatalf("deleted a reassigned/local/new record: %v", err)
				}
			}
			if _, err := f.app.FindRecordById(f.children, candidates[3].Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("unlisted frozen candidate survived: %v", err)
			}
		})
	}
}

func TestCanonicalLogReferencesKeepHistoricalLocalIdentityGuard(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteCleanupFixture(t, "summit_log")
	local := f.child("", f.localActor.Id, f.trail.Id)
	local.Set("iri", "https://old-local.example/api/v1/summit-log/"+local.Id)
	if err := f.app.Save(local); err != nil {
		t.Fatal(err)
	}
	remote := f.child("https://c.example/api/v1/summit-log/remote", f.remoteActor.Id, f.trail.Id)
	refs := canonicalLogReferences(f.app, f.trail, "https://a.example", nil, true)
	if len(refs) != 1 || refs[0].data["iri"] != remote.GetString("iri") || !refs[0].omitted {
		t.Fatalf("narrow projection lost the local author identity guard: %+v", refs)
	}
}
