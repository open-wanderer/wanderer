package permissions_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestTrailsBoundingBoxViewAndIndexAreRemoved(t *testing.T) {
	app := newRulesTestApp(t)

	if _, err := app.FindCollectionByNameOrId("trails_bounding_box"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("trails_bounding_box collection: got %v; want no rows", err)
	}

	var count int
	if err := app.DB().
		NewQuery("SELECT COUNT(*) FROM sqlite_master WHERE name IN ({:view}, {:index})").
		Bind(dbx.Params{"view": "trails_bounding_box", "index": "idx_trails_public_bounds"}).
		Row(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("found %d leftover schema objects", count)
	}
}
