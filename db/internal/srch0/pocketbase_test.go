package srch0_test

import (
	"pocketbase/internal/srch0"
	_ "pocketbase/migrations"
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestSRCH0DatabaseBoundaryUsesProductionConstraints(t *testing.T) {
	app := srch0.App(t, srch0.Data(t))
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	if !users.IsAuth() {
		t.Fatal("users lost its production auth type")
	}
	trail := srch0.Record(t, app, "trails", "public-alpine")
	if trail.GetDateTime("created").Time().Unix() != 1767225621 {
		t.Fatalf("fixture timestamp not preserved: %v", trail.Get("created"))
	}
	if len(trail.Id) != 15 || srch0.Readable(trail.Id) != "public-alpine" {
		t.Fatalf("invalid identity mapping: %s", trail.Id)
	}
	field := trail.Collection().Fields.GetByName("difficulty").(*core.SelectField)
	unsupported := "__srch0_unlisted_enum_value__"
	for slices.Contains(field.Values, unsupported) {
		unsupported += "_"
	}
	trail.Set("difficulty", unsupported)
	if err := app.Save(trail); err == nil {
		t.Fatal("production schema accepted a value outside its declared enum")
	}
	trail.Set("difficulty", "")
	if err := app.Save(trail); err != nil {
		t.Fatalf("optional empty difficulty rejected: %v", err)
	}
	trail.Set("photos", []string{"first.jpg", "second.jpg"})
	trail.Set("thumbnail", 9)
	if err := app.Validate(trail); err != nil {
		return
	}
	t.Fatal("production file field accepted filenames that were never uploaded")
}
