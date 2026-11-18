package main

import (
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/pocketbase/pocketbase/core"
)

// https://github.com/open-wanderer/wanderer/issues/1515
func TestDeleteUserWithTrailLikes(t *testing.T) {
	for _, tc := range []struct {
		name           string
		likers         []string
		wantLikeCount  int
		ownerIsDeleted bool
	}{
		{name: "liked another user's trail", likers: []string{"leaver"}, wantLikeCount: 0},
		{name: "liked a trail others also liked", likers: []string{"leaver", "other"}, wantLikeCount: 1},
		{name: "liked own trail", likers: []string{"owner"}, ownerIsDeleted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "fde406459dc1f6ca6f348e1f44a9a2af")
			f := newRecordHooksFixture(t)
			f.bindHooks()

			actors := map[string]*core.Record{}
			for _, name := range []string{"owner", "leaver", "other"} {
				actors[name] = f.registeredActor(name)
			}
			trail := f.save("trails", map[string]any{"name": "Trail", "author": actors["owner"].Id, "public": true})
			for _, name := range tc.likers {
				f.save("trail_like", map[string]any{"trail": trail.Id, "actor": actors[name].Id})
			}

			deleted := "leaver"
			if tc.ownerIsDeleted {
				deleted = "owner"
			}
			user, err := f.app.FindRecordById("users", actors[deleted].GetString("user"))
			if err != nil {
				t.Fatal(err)
			}
			if err := f.app.Delete(user); err != nil {
				t.Fatalf("delete user: %v", err)
			}

			if n, err := f.app.CountRecords("trail_like", nil); err != nil || n != int64(tc.wantLikeCount) {
				t.Errorf("trail_like count = %d, %v; want %d", n, err, tc.wantLikeCount)
			}
			if tc.ownerIsDeleted {
				return
			}
			trail, err = f.app.FindRecordById("trails", trail.Id)
			if err != nil {
				t.Fatal(err)
			}
			if got := trail.GetInt("like_count"); got != tc.wantLikeCount {
				t.Errorf("like_count = %d; want %d", got, tc.wantLikeCount)
			}
		})
	}
}

func TestUnlikeTrailFederatesUndoForLocalActorsOnly(t *testing.T) {
	for _, tc := range []struct {
		name      string
		local     bool
		wantUndos int64
	}{
		{name: "local actor", local: true, wantUndos: 1},
		{name: "remote actor", local: false, wantUndos: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "fde406459dc1f6ca6f348e1f44a9a2af")
			f := newRecordHooksFixture(t)
			f.bindHooks()

			owner := f.registeredActor("owner")
			liker := f.registeredActor("liker")
			if !tc.local {
				liker = f.save("activitypub_actors", map[string]any{"preferred_username": "remote",
					"iri": "https://remote.example/users/remote", "inbox": "https://remote.example/inbox"})
			}
			trail := f.save("trails", map[string]any{"name": "Trail", "author": owner.Id})
			like := f.save("trail_like", map[string]any{"trail": trail.Id, "actor": liker.Id})

			if err := f.app.Delete(like); err != nil {
				t.Fatalf("unlike: %v", err)
			}

			undos, err := f.app.CountRecords("activitypub_activities", dbx.HashExp{"type": "Undo"})
			if err != nil || undos != tc.wantUndos {
				t.Errorf("Undo activities = %d, %v; want %d", undos, err, tc.wantUndos)
			}
		})
	}
}

// registeredActor returns the actor provisioned by the user create hook.
func (f *recordHooksFixture) registeredActor(name string) *core.Record {
	f.t.Helper()
	user := f.save("users", map[string]any{"email": name + "@example.com", "username": name})
	actor, err := f.app.FindFirstRecordByData("activitypub_actors", "user", user.Id)
	if err != nil {
		f.t.Fatal(err)
	}
	return actor
}
