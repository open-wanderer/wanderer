package main

import (
	"testing"

	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func TestOrphanPhotoDeletionUsesOneTransaction(t *testing.T) {
	for _, restricted := range []bool{false, true} {
		t.Run(map[bool]string{false: "any author", true: "matching author"}[restricted], func(t *testing.T) {
			f := newRecordHooksFixture(t)
			author := f.localActor("photographer")
			photo := f.photo(author)
			sawDelete := false
			f.app.OnRecordDelete("assets").BindFunc(func(e *core.RecordEvent) error {
				sawDelete = true
				if e.App.TxInfo() == nil {
					t.Fatal("orphan check and delete do not share a transaction")
				}
				return e.Next()
			})
			owner := ""
			if restricted {
				owner = author.Id
			}
			deleted, err := util.DeleteAssetIfOrphanedByAuthor(f.app, photo.Id, owner)
			if err != nil || !deleted || !sawDelete || f.exists(photo) {
				t.Fatalf("orphan photo not deleted: deleted=%v seen=%v err=%v", deleted, sawDelete, err)
			}
		})
	}
}

func TestDeletedTargetCollectsPhotosAfterDeleteTransactionStarts(t *testing.T) {
	for _, kind := range []string{"trail", "waypoint", "summit_log"} {
		t.Run(kind, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			actor := f.localActor("owner")
			trail := f.save("trails", map[string]any{"name": "Trail", "author": actor.Id})
			target := trail
			switch kind {
			case "waypoint":
				target = f.save("waypoints", map[string]any{"name": "Point", "trail": trail.Id, "author": actor.Id})
			case "summit_log":
				target = f.save("summit_logs", map[string]any{"trail": trail.Id, "author": actor.Id, "date": "2026-01-01"})
			}
			photo := f.photo(actor)
			f.bindHooks()
			// A link committed after the initial delete event must be collected too.
			f.app.OnRecordDelete(target.Collection().Name).Bind(&hook.Handler[*core.RecordEvent]{Priority: 50, Func: func(e *core.RecordEvent) error {
				linkCollection, err := e.App.FindCollectionByNameOrId(kind + "_assets")
				if err != nil {
					return err
				}
				link := core.NewRecord(linkCollection)
				link.Set(kind, target.Id)
				link.Set("asset", photo.Id)
				if err := e.App.Save(link); err != nil {
					return err
				}
				return e.Next()
			}})
			sawTransaction := false
			f.app.OnRecordDeleteExecute(target.Collection().Name).Bind(&hook.Handler[*core.RecordEvent]{Priority: 101, Func: func(e *core.RecordEvent) error {
				sawTransaction = e.App.TxInfo() != nil
				return e.Next()
			}})
			if err := f.app.Delete(target); err != nil {
				t.Fatal(err)
			}
			if !sawTransaction {
				t.Fatal("target deletion did not enter its transaction")
			}
			if f.exists(photo) {
				t.Fatal("photo committed before the delete transaction was missed")
			}
		})
	}
}

func TestOrphanPhotoDeletionPreservesAConcurrentlyLinkedPhoto(t *testing.T) {
	f := newRecordHooksFixture(t)
	author := f.localActor("photographer")
	trail := f.save("trails", map[string]any{"name": "Trail", "author": author.Id})
	photo := f.photo(author)
	linked := false
	f.app.OnRecordDelete("assets").BindFunc(func(e *core.RecordEvent) error {
		// Commit through another connection after the orphan check, before delete.
		_, err := f.app.ConcurrentDB().Insert("trail_assets", dbx.Params{
			"id": "latephotolink01", "asset": photo.Id, "trail": trail.Id,
		}).Execute()
		if err != nil {
			return err
		}
		linked = true
		return e.Next()
	})
	deleted, _ := util.DeleteAssetIfOrphaned(f.app, photo.Id)
	if !linked {
		t.Fatal("concurrent link was not committed")
	}
	if deleted || !f.exists(photo) {
		t.Fatal("photo linked after the orphan check was deleted")
	}
	link, err := f.app.FindRecordById("trail_assets", "latephotolink01")
	if err != nil || link.GetString("asset") != photo.Id {
		t.Fatalf("concurrent photo link did not survive: %v", err)
	}
}
