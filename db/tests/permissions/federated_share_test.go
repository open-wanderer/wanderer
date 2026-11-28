package permissions_test

import (
	"bytes"
	"encoding/json"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"pocketbase/hooks"
	"pocketbase/routes"
	"pocketbase/util"
	"strings"
	"testing"
)

func TestFederatedShareTokenReadsAndRevokesDocumentsChildrenAndPhotos(t *testing.T) {
	app := newRulesTestApp(t)
	app.OnFileDownloadRequest("assets").BindFunc(hooks.AuthorizeAssetFileDownload())
	user := saveRulesTestRecord(t, app, "users", map[string]any{"username": "owner", "email": "owner@a.example", "password": "test-password"})
	owner := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{"username": "owner", "preferred_username": "owner", "domain": "a.example", "user": user.Id, "public_key": "key", "is_local": true, "iri": "https://a.example/owner", "inbox": "https://a.example/owner/inbox"})
	recipient := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{"username": "riley@b.example", "preferred_username": "riley", "domain": "b.example", "public_key": "key", "iri": "https://b.example/riley", "inbox": "https://b.example/riley/inbox"})
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{"name": "Shared trail", "author": owner.Id, "public": true})
	share := saveRulesTestRecord(t, app, "trail_share", map[string]any{"trail": trail.Id, "actor": recipient.Id, "permission": "view"})
	// Existing directed shares continue working after their public trail becomes private.
	trail.Set("public", false)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	token, err := util.EnsureFederatedTrailShareToken(app, share)
	if err != nil || !util.ValidFederatedShareToken(token) {
		t.Fatalf("missing directed share token: %v", err)
	}
	ordinary := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{"trail": trail.Id, "permission": "view"})
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	paths := []string{"/api/collections/trails/records/" + trail.Id, "/api/v1/trail/" + trail.Id}
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		record := trail
		if target != "trail" {
			collection := map[string]string{"waypoint": "waypoints", "summit_log": "summit_logs"}[target]
			values := map[string]any{"trail": trail.Id, "author": owner.Id, "name": "Child", "text": "Entry"}
			record = saveRulesTestRecord(t, app, collection, values)
			paths = append(paths, "/api/collections/"+collection+"/records/"+record.Id)
		}
		file, err := filesystem.NewFileFromBytes(photo.Bytes(), target+".png")
		if err != nil {
			t.Fatal(err)
		}
		asset := saveRulesTestRecord(t, app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": owner.Id, "file": file})
		link := saveRulesTestRecord(t, app, target+"_assets", map[string]any{target: record.Id, "asset": asset.Id})
		paths = append(paths, "/api/files/"+asset.Collection().Id+"/"+asset.Id+"/"+asset.GetString("file"), "/api/collections/"+target+"_assets/records/"+link.Id)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/api/v1/trail/{id}", routes.RemoteTrailGet)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	get := func(path, credential string, want int) {
		t.Helper()
		response := httptest.NewRecorder()
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path+separator+"share="+credential, nil))
		if response.Code != want {
			t.Fatalf("%s status=%d want=%d body=%s", path, response.Code, want, response.Body.String())
		}
		if want == http.StatusOK && strings.Contains(response.Header().Get("Content-Type"), "json") && (strings.Contains(response.Body.String(), token) || strings.Contains(response.Body.String(), "federation_token") || strings.Contains(response.Body.String(), "remote_token")) {
			t.Fatal("response disclosed share credential")
		}
	}
	for _, path := range paths {
		get(path, token, http.StatusOK)
	}
	get(paths[0]+"?expand=waypoints_via_trail.waypoint_assets_via_waypoint.asset,summit_logs_via_trail.summit_log_assets_via_summit_log.asset,trail_assets_via_trail.asset,trail_share_via_trail", token, http.StatusOK)
	// The owner's link-sharing list retains only the ordinary link share.
	if rows, err := app.FindAllRecords("trail_link_share"); err != nil || len(rows) != 1 || rows[0].Id != ordinary.Id {
		t.Fatalf("directed token changed link sharing: %v %v", rows, err)
	}
	data, err := json.Marshal(share)
	if err != nil || bytes.Contains(data, []byte(token)) {
		t.Fatal("directed share response exposes token")
	}
	// A form update cannot overwrite either internal fetch credential.
	ownerAuth, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	update := httptest.NewRequest(http.MethodPatch, "/api/collections/trail_share/records/"+share.Id, strings.NewReader(`{"federation_token":"cccccccccccccccccccccccccccccccc","remote_token":"dddddddddddddddddddddddddddddddd"}`))
	update.Header.Set("Content-Type", "application/json")
	update.Header.Set("Authorization", "Bearer "+ownerAuth)
	updated := httptest.NewRecorder()
	mux.ServeHTTP(updated, update)
	if updated.Code != http.StatusOK {
		t.Fatalf("share form update status=%d body=%s", updated.Code, updated.Body.String())
	}
	current, err := app.FindRecordById("trail_share", share.Id)
	if err != nil || current.GetString("federation_token") != token || current.GetString("remote_token") != "" {
		t.Fatal("share form replaced internal credentials")
	}
	if err := app.Delete(share); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		want := http.StatusNotFound
		if strings.HasPrefix(path, "/api/v1/trail/") {
			want = http.StatusForbidden
		}
		get(path, token, want)
	}
	// Revocation remains scoped to this directed share.
	get(paths[0], ordinary.GetString("token"), http.StatusOK)
}

func TestLegacyPeerShareQueryKeepsExistingPrivateTrailForbidden(t *testing.T) {
	app := newRulesTestApp(t)
	user := saveRulesTestRecord(t, app, "users", map[string]any{"username": "owner", "email": "owner@a.example", "password": "test-password"})
	actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{"username": "owner", "preferred_username": "owner", "domain": "a.example", "user": user.Id, "is_local": true, "public_key": "key", "iri": "https://a.example/owner", "inbox": "https://a.example/inbox/owner"})
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{"name": "Private", "author": actor.Id})
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/api/v1/trail/{id}", routes.RemoteTrailGet)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	// Older peers forward their complete document query to this route.
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/trail/"+trail.Id+"?expand=author,waypoints_via_trail&fields=id,name&share="+strings.Repeat("x", 32), nil)
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("legacy peer private read status=%d want403", response.Code)
	}
	if _, err := app.FindRecordById("trails", trail.Id); err != nil {
		t.Fatal("existing private trail disappeared")
	}
}
