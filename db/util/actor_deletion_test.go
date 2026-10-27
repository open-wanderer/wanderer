package util

import (
	"bytes"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type actorCleanupApp struct {
	core.App
	transaction func(func(core.App) error) error
	query       func(any, string, string, int, int, ...dbx.Params) ([]*core.Record, error)
	remove      func(core.Model) error
	logger      *slog.Logger
}

func (a *actorCleanupApp) RunInTransaction(fn func(core.App) error) error {
	return a.transaction(fn)
}

func (a *actorCleanupApp) FindRecordsByFilter(collection any, filter, sort string, limit, offset int, params ...dbx.Params) ([]*core.Record, error) {
	return a.query(collection, filter, sort, limit, offset, params...)
}

func (a *actorCleanupApp) Delete(model core.Model) error { return a.remove(model) }
func (a *actorCleanupApp) Logger() *slog.Logger          { return a.logger }

func TestActorCleanupUsesOneTransactionForChecksAndDeletion(t *testing.T) {
	for _, state := range []string{"remote-copies", "query-error", "own-record"} {
		t.Run(state, func(t *testing.T) {
			actor := core.NewRecord(core.NewBaseCollection("activitypub_actors"))
			actor.Id = "remoteactor0001"
			actor.Set("iri", "https://remote.example/api/v1/activitypub/user/alice")
			if actor.GetString("iri") == "" {
				t.Fatal("actor IRI must be present")
			}
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			inTransaction, removed := false, false
			queries := []string{}
			queryError := errors.New("record query unavailable")
			tx := &actorCleanupApp{logger: logger}
			tx.query = func(collection any, filter, sort string, limit, offset int, params ...dbx.Params) ([]*core.Record, error) {
				if !inTransaction {
					t.Fatal("record query escaped its transaction")
				}
				name := collection.(string)
				queries = append(queries, name)
				if filter != "author={:actor}" || len(params) != 1 || params[0]["actor"] != actor.Id || sort != "" || limit > 0 || offset != 0 {
					t.Fatalf("unexpected record query: %s %v", filter, params)
				}
				if name == "comments" && state == "query-error" {
					return nil, queryError
				}
				if name == "comments" && state == "own-record" {
					record := core.NewRecord(core.NewBaseCollection(name))
					record.Id = "retained0000001"
					record.Set("iri", "https://earlier.example/api/v1/comment/"+record.Id)
					next := core.NewRecord(record.Collection())
					next.Id = "retained0000002"
					next.Set("iri", "https://earlier.example/api/v1/comment/"+next.Id)
					return []*core.Record{record, next}, nil
				}
				return nil, nil
			}
			tx.remove = func(model core.Model) error {
				if !inTransaction || model != actor {
					t.Fatal("actor deletion escaped its transaction")
				}
				removed = true
				return nil
			}
			outer := &actorCleanupApp{logger: logger}
			outer.query = func(any, string, string, int, int, ...dbx.Params) ([]*core.Record, error) {
				t.Fatal("record query used the outer app")
				return nil, nil
			}
			outer.remove = func(core.Model) error { t.Fatal("actor deletion used the outer app"); return nil }
			transactions := 0
			outer.transaction = func(fn func(core.App) error) error {
				transactions++
				inTransaction = true
				defer func() { inTransaction = false }()
				return fn(tx)
			}
			err := DeleteRemoteActor(outer, actor)
			if transactions != 1 {
				t.Fatalf("transactions=%d, want one", transactions)
			}
			if state == "remote-copies" {
				if err != nil || !removed || !slices.Equal(queries, []string{"trails", "lists", "comments", "summit_logs", "waypoints"}) {
					t.Fatalf("cleanup error=%v removed=%t queries=%v", err, removed, queries)
				}
			} else if removed || err == nil {
				t.Fatalf("cleanup continued after %s: removed=%t err=%v", state, removed, err)
			}
			if state == "query-error" && !errors.Is(err, queryError) {
				t.Fatalf("record query error lost: %v", err)
			}
			if state == "own-record" {
				if strings.Contains(output.String(), "retained0000002") {
					t.Fatalf("cleanup log must identify its first retained record: %s", output.String())
				}
				for _, field := range []string{actor.GetString("iri"), `"collection":"comments"`, `"record":"retained0000001"`} {
					if !strings.Contains(output.String(), field) {
						t.Fatalf("retained record log missing %s: %s", field, output.String())
					}
				}
			}
		})
	}
}
