package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
)

func TestAssetMetadataOwnershipLookupIsSharedOnlyWithinOneRequest(t *testing.T) {
	f := newRecordHooksFixture(t)
	f.allowAssetReads()
	owner := f.localActor("owner")
	viewer := f.localActor("viewer")
	trail := f.save("trails", map[string]any{"author": owner.Id, "name": "Public", "public": true})
	for i := 0; i < 64; i++ {
		photo := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "metadata": map[string]any{"source_record": "private-origin", "original_filename": "private.jpg"}})
		f.save("trail_assets", map[string]any{"trail": trail.Id, "asset": photo.Id})
	}
	f.bindHooks()
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	var lookups atomic.Int32
	logger := func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
		if strings.HasPrefix(query, "SELECT `activitypub_actors`.* FROM `activitypub_actors`") {
			lookups.Add(1)
		}
	}
	for _, database := range []*dbx.DB{f.app.NonconcurrentDB().(*dbx.DB), f.app.ConcurrentDB().(*dbx.DB)} {
		previous := database.QueryLogFunc
		database.QueryLogFunc = logger
		t.Cleanup(func() { database.QueryLogFunc = previous })
	}
	request := func(userID string, wantPrivate, expanded bool) {
		t.Helper()
		user, err := f.app.FindRecordById("users", userID)
		if err != nil {
			t.Fatal(err)
		}
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		lookups.Store(0)
		path := "/api/collections/assets/records?perPage=100"
		if expanded {
			path = "/api/collections/trails/records/" + trail.Id + "?expand=trail_assets_via_trail.asset"
		}
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("asset listing failed: %d %s", response.Code, response.Body.String())
		}
		type item struct {
			Metadata map[string]any `json:"metadata"`
		}
		var body struct {
			Items  []item
			Expand struct {
				Links []struct {
					Expand struct {
						Asset item `json:"asset"`
					} `json:"expand"`
				} `json:"trail_assets_via_trail"`
			} `json:"expand"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if expanded {
			for _, link := range body.Expand.Links {
				body.Items = append(body.Items, link.Expand.Asset)
			}
		}
		if len(body.Items) != 64 || lookups.Load() != 1 {
			t.Fatalf("ownership lookup repeated for each asset: items%d lookups%d", len(body.Items), lookups.Load())
		}
		for _, asset := range body.Items {
			_, private := asset.Metadata["source_record"]
			if private != wantPrivate {
				t.Fatalf("provenance ownership mismatch: %v", asset.Metadata)
			}
		}
		if strings.Contains(response.Body.String(), "__assetMetadataOwnerMemo") {
			t.Fatal("internal request memo leaked into response")
		}
	}
	request(owner.GetString("user"), true, false)
	request(owner.GetString("user"), true, true)
	request(viewer.GetString("user"), false, false)
	// The same account starts a new request after the ownership relation changes.
	previousUser := owner.GetString("user")
	owner.Set("user", "")
	if err := f.app.UnsafeWithoutHooks().Save(owner); err != nil {
		t.Fatal(err)
	}
	request(previousUser, false, false)
}
