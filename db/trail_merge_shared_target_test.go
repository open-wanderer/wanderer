package main

import (
	"errors"
	"testing"

	"pocketbase/services/trailmerge"
)

func TestManualMergeSuggestionsAllowEditSharedTargetsButOnlyOwnedSources(t *testing.T) {
	f := newRecordHooksFixture(t)
	merger, owner := f.localActor("merger"), f.localActor("owner")
	source := f.save("trails", map[string]any{"name": "Own source", "author": merger.Id})
	// Bias target ranking toward the source. Permissions must still force
	// the shared record to be the target, since it cannot be deleted.
	f.save("summit_logs", map[string]any{"trail": source.Id, "author": merger.Id, "text": "Valuable"})
	target := f.save("trails", map[string]any{"name": "Shared target", "author": owner.Id})
	share := f.save("trail_share", map[string]any{"trail": target.Id, "actor": merger.Id, "permission": "edit"})
	if !trailmerge.CanMerge(f.app, merger.Id, source, target, true) {
		t.Fatal("edit-shared target should be mergeable")
	}
	request := trailmerge.SuggestRequest{Mode: trailmerge.SuggestModeManualSelection, TrailIDs: []string{source.Id, target.Id}}
	suggestion, err := trailmerge.Suggest(f.app, merger.Id, request)
	if err != nil {
		t.Fatal(err)
	}
	if suggestion.TargetTrailID != target.Id {
		t.Fatalf("suggested undeletable source as merge source: %+v", suggestion)
	}
	for _, candidate := range suggestion.Candidates {
		if candidate.Selectable != (candidate.TrailID == target.Id) {
			t.Fatalf("candidate does not match source deletion permission: %+v", candidate)
		}
	}
	share.Set("permission", "view")
	if err := f.app.SaveNoValidate(share); err != nil {
		t.Fatal(err)
	}
	if _, err := trailmerge.Suggest(f.app, merger.Id, request); !errors.Is(err, trailmerge.ErrSourceActorMismatch) {
		t.Fatalf("read-only target accepted: %v", err)
	}
}
