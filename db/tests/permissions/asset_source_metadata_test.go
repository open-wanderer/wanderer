package permissions_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pocketbase/hooks"
	"pocketbase/util"
)

func TestAssetAPICannotSetLegacySourceMetadata(t *testing.T) {
	f := newOwnershipFixture(t)
	f.app.OnRecordCreateRequest("assets").BindFunc(hooks.PreserveAssetSourceMetadata())
	f.app.OnRecordUpdateRequest("assets").BindFunc(hooks.PreserveAssetSourceMetadata())
	f.app.OnRecordCreate("assets").BindFunc(hooks.StoreAssetSourceMetadata())
	f.app.OnRecordUpdate("assets").BindFunc(hooks.StoreAssetSourceMetadata())
	trusted := map[string]any{"source_collection": "trails", "source_record": f.trail.Id, "source_file": "original.png"}
	asset := saveRulesTestRecord(t, f.app, "assets", map[string]any{
		"author": f.ownerActor.Id, "type": "photo", "storage_mode": "copy", "metadata": trusted,
	})
	for _, format := range []string{"json", "multipart-json"} {
		response := ownershipRequest(t, f.handler, asset, f.owner, http.MethodPatch, format, map[string]any{
			"metadata": map[string]any{"source_collection": "summit_logs", "source_record": f.log.Id, "source_file": "other.png", "label": "editable"},
		})
		if response.Code != http.StatusOK {
			t.Fatalf("%s update: %d %s", format, response.Code, response.Body)
		}
		fresh, err := f.app.FindRecordById("assets", asset.Id)
		if err != nil {
			t.Fatal(err)
		}
		metadata := util.AssetMetadataWithProvenance(fresh)
		for field, value := range trusted {
			if metadata[field] != value {
				t.Fatalf("%s changed by client: %#v", field, metadata)
			}
		}
		if metadata["label"] != "editable" {
			t.Fatal("unrelated metadata was discarded")
		}
	}
	data, err := json.Marshal(map[string]any{
		"author": f.ownerActor.Id, "type": "photo", "storage_mode": "copy",
		"metadata": map[string]any{"source_collection": "trails", "source_record": f.otherTrail.Id, "source_file": "other.png", "label": "editable"},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/collections/assets/records", bytes.NewReader(data))
	token, err := f.owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("create: %d %s", response.Code, response.Body)
	}
	result := struct {
		Metadata map[string]any `json:"metadata"`
	}{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"source_collection", "source_record", "source_file"} {
		if _, exists := result.Metadata[field]; exists {
			t.Fatalf("client established trusted source %s: %#v", field, result.Metadata)
		}
	}
	if result.Metadata["label"] != "editable" {
		t.Fatal("create lost unrelated metadata")
	}
}
