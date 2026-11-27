package hooks

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestAssetOwnershipMemoStaysHiddenInCustomAuthExports(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	auth := core.NewRecord(users)
	auth.Id = "user00000000001"
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "user"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(actors)
	actor.Set("user", auth.Id)
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	if !assetAuthorOwnedForRequest(app, &core.RequestInfo{Auth: auth}, actor.Id) {
		t.Fatal("owned actor was not resolved")
	}
	auth.Set("publicExtra", "visible")
	auth.WithCustomData(true)
	data := auth.PublicExport()
	if data["publicExtra"] != "visible" {
		t.Fatal("custom export positive control missing")
	}
	if _, exists := data[assetMetadataOwnerMemoKey]; exists {
		t.Fatal("private request state entered custom auth export")
	}
}
