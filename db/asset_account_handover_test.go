package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestContributedAssetsSurviveContributorAccountDeletion(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, editor := f.localActor("owner"), f.localActor("editor")
	trail := f.save("trails", map[string]any{"name": "Shared trail", "author": owner.Id, "public": true})
	point := f.save("waypoints", map[string]any{"name": "Owner point", "author": owner.Id, "trail": trail.Id})
	entry := f.save("summit_logs", map[string]any{"author": editor.Id, "trail": trail.Id, "date": "2026-01-01"})
	f.save("trail_share", map[string]any{"trail": trail.Id, "actor": editor.Id, "permission": "edit"})
	photoBytes := handoverPhotoBytes(t)
	photos := []*core.Record{}
	for _, target := range []struct {
		collection, field, id string
		thumbnail             bool
	}{
		{"trail_assets", "trail", trail.Id, true}, {"waypoint_assets", "waypoint", point.Id, false},
	} {
		file, err := filesystem.NewFileFromBytes(photoBytes, "contribution.png")
		if err != nil {
			t.Fatal(err)
		}
		photo := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": editor.Id, "file": file})
		f.save(target.collection, map[string]any{"asset": photo.Id, target.field: target.id, "is_thumbnail": target.thumbnail})
		photos = append(photos, photo)
	}
	ownPhoto := f.photo(editor)
	f.save("summit_log_assets", map[string]any{"asset": ownPhoto.Id, "summit_log": entry.Id})
	user, err := f.app.FindRecordById("users", editor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	updated := 0
	f.app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("author") != e.Record.Original().GetString("author") {
			updated++
			if e.App.TxInfo() == nil {
				t.Error("photo handover ran outside the deletion transaction")
			}
		}
		return e.Next()
	})
	if err := f.app.Delete(user); err != nil {
		t.Fatal(err)
	}
	if f.exists(user) || f.exists(editor) {
		t.Fatal("contributor account survived deletion")
	}
	if !f.exists(trail) || !f.exists(point) {
		t.Fatal("another owner's target was deleted")
	}
	if f.exists(entry) || f.exists(ownPhoto) {
		t.Fatal("the contributor's own entry and unshared photo must cascade")
	}
	if updated != len(photos) {
		t.Fatalf("handed over %d photos, want %d", updated, len(photos))
	}
	fsys, err := f.app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	for _, photo := range photos {
		stored, err := f.app.FindRecordById("assets", photo.Id)
		if err != nil || stored.GetString("author") != owner.Id {
			t.Fatalf("contributed photo was lost or has another owner: %v", err)
		}
		reader, err := fsys.GetReader(stored.BaseFilesPath() + "/" + stored.GetString("file"))
		if err != nil {
			t.Fatal(err)
		}
		actual, err := io.ReadAll(reader)
		reader.Close()
		if err != nil || !bytes.Equal(actual, photoBytes) {
			t.Fatalf("photo bytes changed: %v", err)
		}
	}
	title, err := f.app.FindFirstRecordByData("trail_assets", "trail", trail.Id)
	if err != nil || title.GetString("asset") != photos[0].Id || !title.GetBool("is_thumbnail") {
		t.Fatalf("title photo link was lost: %v", err)
	}
	links, err := f.app.FindRecordsByFilter("waypoint_assets", "waypoint={:id}", "", -1, 0, dbx.Params{"id": point.Id})
	if err != nil || len(links) != 1 || links[0].GetString("asset") != photos[1].Id {
		t.Fatalf("waypoint photo link was lost: %v", err)
	}
}

func TestAssetHandoverUsesTargetOwners(t *testing.T) {
	f := newRecordHooksFixture(t)
	trailOwner, childAuthor := f.localActor("trailowner"), f.localActor("childauthor")
	trail := f.save("trails", map[string]any{"name": "Trail", "author": trailOwner.Id})
	point := f.save("waypoints", map[string]any{"name": "Point", "author": childAuthor.Id, "trail": trail.Id})
	cases := []struct {
		kind, id, want     string
		contributor, asset *core.Record
	}{}
	for _, target := range []struct{ kind, id, want string }{
		{"trail", trail.Id, trailOwner.Id}, {"waypoint", point.Id, trailOwner.Id}, {"summit_log", "", childAuthor.Id},
	} {
		contributor := f.localActor("contributor" + target.kind)
		asset := f.photo(contributor)
		var entryTrail *core.Record
		if target.kind == "summit_log" {
			// The contributor can attach this photo while owning the trail.
			entryTrail = f.save("trails", map[string]any{"name": "Entry trail", "author": contributor.Id})
			entry := f.save("summit_logs", map[string]any{"author": childAuthor.Id, "trail": entryTrail.Id, "date": "2026-01-01"})
			target.id = entry.Id
		} else {
			f.save("trail_share", map[string]any{"trail": trail.Id, "actor": contributor.Id, "permission": "edit"})
		}
		f.save(target.kind+"_assets", map[string]any{"asset": asset.Id, target.kind: target.id})
		if entryTrail != nil {
			// A later ownership transfer leaves the entry and its existing link alive.
			entryTrail.Set("author", trailOwner.Id)
			if err := f.app.Save(entryTrail); err != nil {
				t.Fatal(err)
			}
		}
		cases = append(cases, struct {
			kind, id, want     string
			contributor, asset *core.Record
		}{target.kind, target.id, target.want, contributor, asset})
	}
	f.bindHooks()
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			if err := f.app.Delete(tc.contributor); err != nil {
				t.Fatal(err)
			}
			asset, err := f.app.FindRecordById("assets", tc.asset.Id)
			if err != nil || asset.GetString("author") != tc.want {
				t.Fatalf("target owner was not selected: %v", err)
			}
		})
	}
}

func TestAssetHandoverSkipsWaypointsDeletedWithContributor(t *testing.T) {
	f := newRecordHooksFixture(t)
	firstOwner, survivingOwner, contributor := f.localActor("firstowner"), f.localActor("survivingowner"), f.localActor("contributor")
	firstTrail := f.save("trails", map[string]any{"name": "First trail", "author": firstOwner.Id})
	survivingTrail := f.save("trails", map[string]any{"name": "Surviving trail", "author": survivingOwner.Id})
	for _, trail := range []*core.Record{firstTrail, survivingTrail} {
		f.save("trail_share", map[string]any{"trail": trail.Id, "actor": contributor.Id, "permission": "edit"})
	}
	point := f.save("waypoints", map[string]any{"name": "Contributor point", "trail": firstTrail.Id, "author": contributor.Id})
	photo, orphan := f.photo(contributor), f.photo(contributor)
	oldLink := saveDatedHandoverLink(t, f.app, "waypoint_assets", map[string]any{"asset": photo.Id, "waypoint": point.Id}, "2025-01-01 00:00:00.000Z")
	f.save("waypoint_assets", map[string]any{"asset": orphan.Id, "waypoint": point.Id})
	survivingLink := saveDatedHandoverLink(t, f.app, "trail_assets", map[string]any{"asset": photo.Id, "trail": survivingTrail.Id, "is_thumbnail": true}, "2026-01-01 00:00:00.000Z")
	user, err := f.app.FindRecordById("users", contributor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	orphanUpdates := 0
	f.app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == orphan.Id {
			orphanUpdates++
		}
		return e.Next()
	})
	if err := f.app.Delete(user); err != nil {
		t.Fatal(err)
	}
	if f.exists(user) || f.exists(contributor) || f.exists(point) || f.exists(oldLink) {
		t.Fatal("contributor waypoint or its link survived the account cascade")
	}
	if !f.exists(firstTrail) || !f.exists(survivingTrail) {
		t.Fatal("another owner's trail was deleted")
	}
	stored, err := f.app.FindRecordById("assets", photo.Id)
	if err != nil || stored.GetString("author") != survivingOwner.Id {
		t.Fatalf("photo did not reach its surviving target's owner: %v", err)
	}
	link, err := f.app.FindRecordById("trail_assets", survivingLink.Id)
	if err != nil || link.GetString("asset") != photo.Id || link.GetString("trail") != survivingTrail.Id || !link.GetBool("is_thumbnail") {
		t.Fatalf("surviving title link changed: %v", err)
	}
	if f.exists(orphan) || orphanUpdates != 0 {
		t.Fatalf("photo without a surviving target should cascade without a save; updates=%d", orphanUpdates)
	}
}

func TestAssetHandoverSkipsEntriesOnDeletedContributorTrail(t *testing.T) {
	f := newRecordHooksFixture(t)
	contributor, survivingOwner := f.localActor("contributor"), f.localActor("survivingowner")
	visitor := f.save("activitypub_actors", map[string]any{"preferred_username": "visitor", "iri": "https://b.example/users/visitor", "is_local": false})
	ownTrail := f.save("trails", map[string]any{"name": "Contributor trail", "author": contributor.Id, "public": true})
	survivingTrail := f.save("trails", map[string]any{"name": "Surviving trail", "author": survivingOwner.Id})
	f.save("trail_share", map[string]any{"trail": survivingTrail.Id, "actor": contributor.Id, "permission": "edit"})
	entry := f.save("summit_logs", map[string]any{"author": visitor.Id, "trail": ownTrail.Id, "date": "2026-01-01"})
	photo, orphan := f.photo(contributor), f.photo(contributor)
	oldLink := saveDatedHandoverLink(t, f.app, "summit_log_assets", map[string]any{"asset": photo.Id, "summit_log": entry.Id}, "2025-01-01 00:00:00.000Z")
	f.save("summit_log_assets", map[string]any{"asset": orphan.Id, "summit_log": entry.Id})
	survivingLink := saveDatedHandoverLink(t, f.app, "trail_assets", map[string]any{"asset": photo.Id, "trail": survivingTrail.Id}, "2026-01-01 00:00:00.000Z")
	user, err := f.app.FindRecordById("users", contributor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	orphanUpdates := 0
	f.app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Id == orphan.Id {
			orphanUpdates++
		}
		return e.Next()
	})
	if err := f.app.Delete(user); err != nil {
		t.Fatal(err)
	}
	if f.exists(user) || f.exists(contributor) || f.exists(ownTrail) || f.exists(entry) || f.exists(oldLink) {
		t.Fatal("contributor trail or its entry survived the account cascade")
	}
	if !f.exists(visitor) || visitor.GetString("user") != "" || !f.exists(survivingTrail) {
		t.Fatal("visitor or another owner's trail changed")
	}
	stored, err := f.app.FindRecordById("assets", photo.Id)
	if err != nil || stored.GetString("author") != survivingOwner.Id {
		t.Fatalf("photo did not reach its surviving target's owner: %v", err)
	}
	link, err := f.app.FindRecordById("trail_assets", survivingLink.Id)
	if err != nil || link.GetString("asset") != photo.Id || link.GetString("trail") != survivingTrail.Id {
		t.Fatalf("surviving trail link changed: %v", err)
	}
	if f.exists(orphan) || orphanUpdates != 0 {
		t.Fatalf("photo without a surviving target should cascade without a save; updates=%d", orphanUpdates)
	}
}

func TestAssetHandoverMergesExternalIdentityLinks(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, other, contributor := f.localActor("owner"), f.localActor("other"), f.localActor("contributor")
	trail := f.save("trails", map[string]any{"name": "Owner trail", "author": owner.Id})
	otherTrail := f.save("trails", map[string]any{"name": "Other trail", "author": other.Id})
	point := f.save("waypoints", map[string]any{"name": "Point", "trail": otherTrail.Id, "author": other.Id})
	source := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": contributor.Id, "external_provider": "provider", "external_id": "same-photo", "metadata": map[string]any{"content_hash": "same-bytes"}})
	destination := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": owner.Id, "external_provider": "provider", "external_id": "same-photo", "metadata": map[string]any{"content_hash": "same-bytes"}})
	oldest := "2025-01-01 00:00:00.000Z"
	oldTitle := saveDatedHandoverLink(t, f.app, "trail_assets", map[string]any{"asset": source.Id, "trail": trail.Id, "is_thumbnail": true}, oldest)
	duplicate := saveDatedHandoverLink(t, f.app, "trail_assets", map[string]any{"asset": destination.Id, "trail": trail.Id}, "2026-01-01 00:00:00.000Z")
	moved := saveDatedHandoverLink(t, f.app, "waypoint_assets", map[string]any{"asset": source.Id, "waypoint": point.Id}, "2026-02-01 00:00:00.000Z")
	f.bindHooks()
	if err := f.app.Delete(contributor); err != nil {
		t.Fatal(err)
	}
	if f.exists(source) || f.exists(oldTitle) {
		t.Fatal("superseded asset or duplicate link survived")
	}
	title, err := f.app.FindRecordById("trail_assets", duplicate.Id)
	if err != nil || title.GetString("asset") != destination.Id || !title.GetBool("is_thumbnail") || title.GetString("created") != oldest {
		t.Fatalf("combined title/order was not retained: %v", err)
	}
	links, err := f.app.FindRecordsByFilter("trail_assets", "trail={:id}", "", -1, 0, dbx.Params{"id": trail.Id})
	if err != nil || len(links) != 1 {
		t.Fatalf("duplicate title links: %d %v", len(links), err)
	}
	moved, err = f.app.FindRecordById("waypoint_assets", moved.Id)
	if err != nil || moved.GetString("asset") != destination.Id {
		t.Fatalf("another owner's link was not retained: %v", err)
	}
}

func TestAssetHandoverChoosesOldestLinkAcrossBatches(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, oldestOwner, contributor := f.localActor("owner"), f.localActor("oldest"), f.localActor("contributor")
	trail := f.save("trails", map[string]any{"name": "Trail", "author": owner.Id})
	otherTrail := f.save("trails", map[string]any{"name": "Other trail", "author": oldestOwner.Id})
	entry := f.save("summit_logs", map[string]any{"trail": trail.Id, "author": owner.Id, "date": "2026-01-01"})
	assets := []*core.Record{}
	if err := f.app.RunInTransaction(func(tx core.App) error {
		for i := 0; i < 153; i++ {
			collection, err := tx.FindCollectionByNameOrId("assets")
			if err != nil {
				return err
			}
			asset := core.NewRecord(collection)
			asset.Id = fmt.Sprintf("asset%010d", i)
			asset.Load(map[string]any{"type": "photo", "storage_mode": "copy", "author": contributor.Id})
			if err := tx.SaveNoValidate(asset); err != nil {
				return err
			}
			saveDatedHandoverLink(t, tx, "trail_assets", map[string]any{"id": fmt.Sprintf("link%011d", i), "asset": asset.Id, "trail": trail.Id}, "2026-01-01 00:00:00.000Z")
			assets = append(assets, asset)
		}
		for i := 0; i < 153; i++ {
			pointCollection, err := tx.FindCollectionByNameOrId("waypoints")
			if err != nil {
				return err
			}
			targetTrail, created := trail.Id, "2026-01-01 00:00:00.000Z"
			if i == 152 {
				targetTrail, created = otherTrail.Id, "2024-01-01 00:00:00.000Z"
			}
			point := core.NewRecord(pointCollection)
			point.Load(map[string]any{"name": "Point", "trail": targetTrail, "author": owner.Id})
			if err := tx.SaveNoValidate(point); err != nil {
				return err
			}
			saveDatedHandoverLink(t, tx, "waypoint_assets", map[string]any{"id": fmt.Sprintf("link%011d", i), "asset": assets[len(assets)-1].Id, "waypoint": point.Id}, created)
		}
		saveDatedHandoverLink(t, tx, "summit_log_assets", map[string]any{"asset": assets[len(assets)-1].Id, "summit_log": entry.Id}, "2025-01-01 00:00:00.000Z")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	if err := f.app.Delete(contributor); err != nil {
		t.Fatal(err)
	}
	for i, asset := range assets {
		want := owner.Id
		if i == len(assets)-1 {
			want = oldestOwner.Id
		}
		stored, err := f.app.FindRecordById("assets", asset.Id)
		if err != nil || stored.GetString("author") != want {
			t.Fatalf("asset %d did not reach its oldest foreign target's owner: %v", i, err)
		}
	}
}

func TestAssetHandoverRollsBackWithActorDeletion(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, contributor := f.localActor("owner"), f.localActor("contributor")
	trail := f.save("trails", map[string]any{"name": "Trail", "author": owner.Id})
	asset := f.photo(contributor)
	link := f.save("trail_assets", map[string]any{"asset": asset.Id, "trail": trail.Id})
	f.bindHooks()
	f.app.OnRecordDeleteExecute("activitypub_actors").BindFunc(func(e *core.RecordEvent) error { return errors.New("actor deletion stopped") })
	if err := f.app.Delete(contributor); err == nil {
		t.Fatal("expected the actor deletion to stop")
	}
	if !f.exists(contributor) || !f.exists(link) {
		t.Fatal("failed deletion changed actor or link")
	}
	stored, err := f.app.FindRecordById("assets", asset.Id)
	if err != nil || stored.GetString("author") != contributor.Id {
		t.Fatalf("ownership update escaped rollback: %v", err)
	}
}

func TestAssetFileUpdatesRequireSuperuserWhileMetadataRemainsEditable(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, editor := f.localActor("owner"), f.localActor("editor")
	trail := f.save("trails", map[string]any{"name": "Shared trail", "author": owner.Id, "public": true})
	share := f.save("trail_share", map[string]any{"trail": trail.Id, "actor": editor.Id, "permission": "edit"})
	file, err := filesystem.NewFileFromBytes(handoverPhotoBytes(t), "original.png")
	if err != nil {
		t.Fatal(err)
	}
	asset := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": editor.Id, "file": file})
	f.save("trail_assets", map[string]any{"asset": asset.Id, "trail": trail.Id})
	if err := f.app.Delete(share); err != nil {
		t.Fatal(err)
	}
	f.allowAssetReads()
	collection, err := f.app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	updateRule := `@request.auth.id != "" && author.user = @request.auth.id`
	collection.UpdateRule = &updateRule
	if err := f.app.Save(collection); err != nil {
		t.Fatal(err)
	}
	admin := f.save("_superusers", map[string]any{"email": "admin@a.example", "password": "long-test-password"})
	f.bindHooks()
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	user, err := f.app.FindRecordById("users", editor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := admin.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	original := asset.GetString("file")
	path := "/api/collections/assets/records/" + asset.Id
	for _, field := range []string{"file", "file+", "+file", "file-"} {
		t.Run(field, func(t *testing.T) {
			response := patchHandoverPhoto(t, mux, path, token, field, original)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("file change returned %d: %s", response.Code, response.Body.String())
			}
			stored, err := f.app.FindRecordById("assets", asset.Id)
			if err != nil || stored.GetString("file") != original {
				t.Fatalf("file changed after rejected update: %v", err)
			}
		})
	}
	request := httptest.NewRequest(http.MethodPatch, path, strings.NewReader(`{"metadata":{"label":"Author edit"}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("metadata update returned %d: %s", response.Code, response.Body.String())
	}
	stored, err := f.app.FindRecordById("assets", asset.Id)
	if err != nil {
		t.Fatal(err)
	}
	metadata := map[string]any{}
	if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["label"] != "Author edit" {
		t.Fatal("metadata-only author update was lost")
	}
	replacement := append(handoverPhotoBytes(t), []byte("internal replacement")...)
	file, err = filesystem.NewFileFromBytes(replacement, "internal.png")
	if err != nil {
		t.Fatal(err)
	}
	stored.Set("file", file)
	if err := f.app.Save(stored); err != nil {
		t.Fatal(err)
	}
	if stored.GetString("file") == original {
		t.Fatal("internal file replacement was blocked")
	}
	if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata["content_hash"] != fmt.Sprintf("%x", sha256.Sum256(replacement)) {
		t.Fatal("internal file replacement was not rehashed")
	}
	response = patchHandoverPhoto(t, mux, path, adminToken, "file", stored.GetString("file"))
	if response.Code != http.StatusOK {
		t.Fatalf("superuser file update returned %d: %s", response.Code, response.Body.String())
	}
}

func handoverPhotoBytes(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func saveDatedHandoverLink(t *testing.T, app core.App, collection string, values map[string]any, created string) *core.Record {
	t.Helper()
	table, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(table)
	record.Load(values)
	record.SetRaw("created", created)
	if err := app.SaveNoValidate(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func patchHandoverPhoto(t *testing.T, handler http.Handler, path, token, field, original string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if field == "file-" {
		if err := writer.WriteField(field, original); err != nil {
			t.Fatal(err)
		}
	} else {
		part, err := writer.CreateFormFile(field, "replacement.png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(handoverPhotoBytes(t)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPatch, path, &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if !json.Valid(response.Body.Bytes()) {
		t.Fatal("record API did not return JSON")
	}
	return response
}

func TestAssetHandoverKeepsDifferentOrUnknownPhotoBytes(t *testing.T) {
	for _, kind := range []string{"different bytes", "unknown source hash", "unknown destination hash"} {
		t.Run(kind, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			owner, contributor := f.localActor("owner"), f.localActor("contributor")
			trail := f.save("trails", map[string]any{"name": "Shared trail", "author": owner.Id})
			photos := []*core.Record{}
			payloads := [][]byte{handoverPhotoBytes(t), append(handoverPhotoBytes(t), []byte("different photo")...)}
			for i, author := range []*core.Record{contributor, owner} {
				file, err := filesystem.NewFileFromBytes(payloads[i], "photo.png")
				if err != nil {
					t.Fatal(err)
				}
				hash := fmt.Sprintf("%x", sha256.Sum256(payloads[i]))
				if kind == "unknown source hash" && i == 0 || kind == "unknown destination hash" && i == 1 {
					hash = ""
				}
				photo := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": author.Id, "file": file, "external_provider": "provider", "external_id": "shared-identity", "metadata": map[string]any{"content_hash": hash, "label": "preserved"}})
				photos = append(photos, photo)
			}
			link := f.save("trail_assets", map[string]any{"asset": photos[0].Id, "trail": trail.Id, "is_thumbnail": true})
			f.bindHooks()
			if err := f.app.Delete(contributor); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById("assets", photos[0].Id)
			if err != nil || stored.GetString("author") != owner.Id {
				t.Fatalf("contributed photo was not handed over: %v", err)
			}
			linked, err := f.app.FindRecordById("trail_assets", link.Id)
			if err != nil || linked.GetString("asset") != photos[0].Id || !linked.GetBool("is_thumbnail") {
				t.Fatalf("photo link was redirected: %v", err)
			}
			if stored.GetString("external_provider") != "" || stored.GetString("external_id") != "" {
				t.Fatal("conflicting external identity was retained")
			}
			provenance := map[string]any{}
			if err := stored.UnmarshalJSONField("source_metadata", &provenance); err != nil || provenance["source_external_id"] != "shared-identity" || provenance["source_external_provider"] != "provider" {
				t.Fatalf("original identity was not preserved: %v", err)
			}
			fsys, err := f.app.NewFilesystem()
			if err != nil {
				t.Fatal(err)
			}
			defer fsys.Close()
			for i, photo := range photos {
				photo, err = f.app.FindRecordById("assets", photo.Id)
				if err != nil {
					t.Fatal(err)
				}
				reader, err := fsys.GetReader(photo.BaseFilesPath() + "/" + photo.GetString("file"))
				if err != nil {
					t.Fatal(err)
				}
				actual, err := io.ReadAll(reader)
				reader.Close()
				if err != nil || !bytes.Equal(actual, payloads[i]) {
					t.Fatalf("photo bytes changed: %v", err)
				}
			}
		})
	}
}
