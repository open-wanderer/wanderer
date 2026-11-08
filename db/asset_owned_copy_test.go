package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"net/http"
	"net/http/httptest"
	"pocketbase/routes"
	"pocketbase/util"
	"testing"
)

func TestOwnedPhotoCopyLookupChecksSourceAccessAndPreservesTrustedProvenance(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("owner")
	caller := f.localActor("caller")
	other := f.localActor("other")
	public := f.save("trails", map[string]any{"author": owner.Id, "name": "Public source", "public": true})
	private := f.save("trails", map[string]any{"author": owner.Id, "name": "Private source"})
	f.bindHooks()
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	file := func() *filesystem.File {
		photo, err := filesystem.NewFileFromBytes(raw, "photo.png")
		if err != nil {
			t.Fatal(err)
		}
		return photo
	}
	source := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": file()})
	inaccessible := f.save("assets", map[string]any{"author": owner.Id, "type": "photo", "storage_mode": "copy", "file": file()})
	f.save("trail_assets", map[string]any{"trail": public.Id, "asset": source.Id})
	f.save("trail_assets", map[string]any{"trail": private.Id, "asset": inaccessible.Id})
	ownCopy := f.save("assets", map[string]any{"author": caller.Id, "type": "photo", "storage_mode": "copy", "file": file(), "metadata": map[string]any{"copied_from": source.Id, "original_filename": "original.png"}})
	f.save("assets", map[string]any{"author": other.Id, "type": "photo", "storage_mode": "copy", "file": file(), "metadata": map[string]any{"copied_from": source.Id}})
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	api.GET("/assets/{id}/owned-copy", routes.AssetOwnedCopy)
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.app.FindRecordById("users", caller.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path string, payload map[string]any, authenticated bool) *httptest.ResponseRecorder {
		data, _ := json.Marshal(payload)
		req := httptest.NewRequest(method, path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		if authenticated {
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	path := "/assets/" + source.Id + "/owned-copy"
	if response := request(http.MethodGet, path, nil, false); response.Code != 401 {
		t.Fatalf("anonymous reusable copy lookup: %d %s", response.Code, response.Body.String())
	}
	response := request(http.MethodGet, path, nil, true)
	var data struct {
		ID       string
		Metadata map[string]any
	}
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || data.ID != ownCopy.Id || data.Metadata["copied_from"] != source.Id {
		t.Fatalf("owned source reuse: %d %s", response.Code, response.Body.String())
	}
	response = request(http.MethodPost, "/api/collections/assets/records", map[string]any{"author": caller.Id, "type": "photo", "storage_mode": "copy", "metadata": map[string]any{"copied_from": inaccessible.Id, "original_filename": "secret.png"}}, true)
	if response.Code != 404 {
		t.Fatalf("inaccessible copy provenance accepted: %d %s", response.Code, response.Body.String())
	}
	response = request(http.MethodPost, "/api/collections/assets/records", map[string]any{"author": caller.Id, "type": "photo", "storage_mode": "copy", "source_metadata": map[string]any{"source_record": private.Id, "copied_from": inaccessible.Id}, "metadata": map[string]any{"source_record": private.Id, "copied_from": source.Id, "original_filename": "legitimate.png", "label": "copied"}}, true)
	if response.Code != 200 {
		t.Fatalf("authorized source copy metadata rejected: %d %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	created, err := f.app.FindRecordById("assets", data.ID)
	if err != nil {
		t.Fatal(err)
	}
	provenance := util.AssetProvenanceMetadata(created)
	if provenance["copied_from"] != source.Id || provenance["original_filename"] != "legitimate.png" || provenance["source_record"] != nil {
		t.Fatalf("client supplied hidden/legacy provenance: %v", provenance)
	}
	response = request(http.MethodPatch, "/api/collections/assets/records/"+ownCopy.Id, map[string]any{"source_metadata": map[string]any{"copied_from": inaccessible.Id}, "metadata": map[string]any{"copied_from": inaccessible.Id, "original_filename": "supplied.png", "label": "editable"}}, true)
	if response.Code != 200 {
		t.Fatalf("ordinary metadata edit failed: %d %s", response.Code, response.Body.String())
	}
	stored, err := f.app.FindRecordById("assets", ownCopy.Id)
	if err != nil {
		t.Fatal(err)
	}
	if util.AssetProvenanceMetadata(stored)["copied_from"] != source.Id || util.AssetProvenanceMetadata(stored)["original_filename"] != "original.png" {
		t.Fatalf("existing trusted provenance changed: %v", stored.PublicExport())
	}
	public.Set("public", false)
	if err := f.app.Save(public); err != nil {
		t.Fatal(err)
	}
	if response := request(http.MethodGet, path, nil, true); response.Code != 404 {
		t.Fatalf("reuse ignored revoked source access: %d %s", response.Code, response.Body.String())
	}
}
