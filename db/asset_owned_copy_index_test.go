package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"pocketbase/routes"
)

// Exercise the current registered schema and the HTTP lookup; a separately
// constructed expression index could pass while its migration wasn't wired.
func TestOwnedCopyHTTPUsesRegisteredLineageIndexWithCompetingIndexes(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("indexed-owner")
	source := f.photo(owner)
	user, err := f.app.FindRecordById("users", owner.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.DB().NewQuery(`WITH RECURSIVE photos(n) AS(SELECT 1 UNION ALL SELECT n+1 FROM photos WHERE n<50000) INSERT INTO assets(id,author,type,storage_mode,file,source_metadata) SELECT printf('c%014d',n),{:author},'photo','copy','copy.jpg',json_object('copied_from',CASE WHEN n=50000 THEN {:source} ELSE printf('s%014d',n) END) FROM photos`).Bind(dbx.Params{"author": owner.Id, "source": source.Id}).Execute(); err != nil {
		t.Fatal(err)
	}
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	api.GET("/assets/{id}/owned-copy", routes.AssetOwnedCopy)
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, statistics := range []bool{false, true} {
		if statistics {
			if _, err := f.app.DB().NewQuery("PRAGMA analysis_limit=400").Execute(); err != nil {
				t.Fatal(err)
			}
			if _, err := f.app.DB().NewQuery("ANALYZE").Execute(); err != nil {
				t.Fatal(err)
			}
		}
		db := f.app.ConcurrentDB().(*dbx.DB)
		previous := db.QueryLogFunc
		captured := ""
		db.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
			if strings.HasPrefix(query, "SELECT") && strings.Contains(query, "FROM `assets`") && strings.Contains(query, "$.copied_from") {
				captured = query
			}
		}
		req := httptest.NewRequest(http.MethodGet, "/assets/"+source.Id+"/owned-copy", nil)
		req.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		db.QueryLogFunc = previous
		var found struct{ ID string }
		if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
			t.Fatal(err)
		}
		if response.Code != 200 || found.ID != "c00000000050000" || captured == "" {
			t.Fatalf("actual owned-copy lookup: HTTP%d %s SQL=%s", response.Code, response.Body.String(), captured)
		}
		var plan []struct{ Detail string }
		if err := f.app.DB().NewQuery("EXPLAIN QUERY PLAN " + captured).All(&plan); err != nil {
			t.Fatal(err)
		}
		used := false
		for _, row := range plan {
			t.Logf("sampled stats=%t: %s", statistics, row.Detail)
			used = used || strings.Contains(row.Detail, "idx_assets_author_copy_lineage") && strings.Contains(row.Detail, "<expr>=?")
		}
		if !used {
			t.Fatalf("registered lineage index did not constrain the real copied_from lookup: %v", plan)
		}
	}
}
