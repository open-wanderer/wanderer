package migrations

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestOwnedAssetCopyLookupUsesHiddenLineageIndex(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(&core.TextField{Name: "author"}, &core.TextField{Name: "type"}, &core.JSONField{Name: "source_metadata", Hidden: true})
	schema := core.NewBaseCollection("activitypub_actors", "pbc_1295301207")
	if err := app.Save(schema); err != nil {
		t.Fatal(err)
	}
	// Use the production schema's expression index with the slim test table.
	production, err := createAssetsCollection(app)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range production.Indexes {
		if strings.Contains(index, "idx_assets_author_copy_lineage") {
			assets.Indexes = append(assets.Indexes, index)
		}
	}
	if err := app.Delete(production); err != nil {
		t.Fatal(err)
	}
	if err := app.Save(assets); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB().NewQuery(`WITH RECURSIVE copies(n) AS(SELECT 1 UNION ALL SELECT n+1 FROM copies WHERE n<50000) INSERT INTO assets(id,author,type,source_metadata) SELECT printf('a%014d',n),'owner','photo',json_object('copied_from',printf('source-%d',n)) FROM copies`).Execute(); err != nil {
		t.Fatal(err)
	}
	captured := ""
	db := app.ConcurrentDB().(*dbx.DB)
	previous := db.QueryLogFunc
	db.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
		if strings.HasPrefix(query, "SELECT") && strings.Contains(query, "FROM `assets`") {
			captured = query
		}
	}
	record, err := app.FindFirstRecordByFilter("assets", "author='owner' && type='photo' && source_metadata.copied_from='source-25000'")
	db.QueryLogFunc = previous
	if err != nil || record.Id != "a00000000025000" || captured == "" {
		t.Fatalf("real hidden-lineage lookup failed: %v query%s", err, captured)
	}
	t.Log(captured)
	rows := []struct{ Detail string }{}
	if err := app.DB().NewQuery("EXPLAIN QUERY PLAN " + captured).All(&rows); err != nil {
		t.Fatal(err)
	}
	used := false
	for _, row := range rows {
		t.Log(row.Detail)
		if strings.Contains(row.Detail, "idx_assets_author_copy_lineage") && strings.Contains(row.Detail, "<expr>=?") {
			used = true
		}
		if strings.Contains(row.Detail, "SCAN assets") {
			t.Fatal("lineage lookup scanned complete library")
		}
	}
	if !used {
		t.Fatalf("lineage expression was not indexed: %v", rows)
	}
}
