package assets_test

import (
	"strings"
	"testing"
)

func TestAssetAuthorQueriesUseIndexesWithCurrentSchema(t *testing.T) {
	app := newAssetFilterTestApp(t)
	for _, tc := range []struct{ name, index, query string }{
		{"external identity", "idx_assets_author", "SELECT id FROM assets WHERE author='actor' AND type='photo' AND external_provider='provider' AND external_id='photo'"},
		{"capture order", "idx_assets_author_taken_at", "SELECT id FROM assets WHERE author='actor' ORDER BY taken_at,id LIMIT 26"},
		{"library order", "idx_assets_author_library", "SELECT id FROM assets WHERE author='actor' AND type='photo' ORDER BY taken_at DESC,created DESC,id LIMIT 51"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows := []struct{ Detail string }{}
			if err := app.DB().NewQuery("EXPLAIN QUERY PLAN " + tc.query).All(&rows); err != nil {
				t.Fatal(err)
			}
			usesIndex := false
			for _, row := range rows {
				if strings.Contains(row.Detail, "INDEX "+tc.index+" (") {
					usesIndex = true
				}
				if strings.Contains(row.Detail, "SCAN assets") || strings.Contains(row.Detail, "TEMP B-TREE") {
					t.Fatalf("unindexed query: %s", row.Detail)
				}
			}
			if !usesIndex {
				t.Fatalf("exact index %s not used: %v", tc.index, rows)
			}
		})
	}
}
