package permissions_test

import (
	"net/http"
	"testing"
)

// author, external_provider and external_id say whose photo an asset is and
// which remote asset it stands for, and federation reuses an asset matched on
// them. Like the owner fields guarded by 1791110001, they must not change
// through the records API.
func TestAssetIdentityIsImmutableThroughAPI(t *testing.T) {
	f := newOwnershipFixture(t)
	asset := saveRulesTestRecord(t, f.app, "assets", map[string]any{
		"author": f.ownerActor.Id, "type": "photo", "storage_mode": "link_private",
		"external_provider": "immich", "external_id": "immich-asset-1",
	})
	for _, change := range []struct {
		field string
		value string
	}{
		{"author", f.strangerActor.Id},
		{"external_provider", "federation"},
		{"external_id", "https://b.example/api/v1/assets/abcdefghijklmno"},
	} {
		for _, format := range []string{"json", "multipart", "multipart-json"} {
			t.Run(change.field+"/"+format, func(t *testing.T) {
				response := ownershipRequest(t, f.handler, asset, f.owner, http.MethodPatch, format, map[string]any{change.field: change.value})
				if response.Code != http.StatusNotFound {
					t.Errorf("status = %d; want 404: %s", response.Code, response.Body.String())
				}
				fresh, err := f.app.FindRecordById("assets", asset.Id)
				if err != nil {
					t.Fatal(err)
				}
				if got, want := fresh.GetString(change.field), asset.GetString(change.field); got != want {
					t.Fatalf("%s changed from %q to %q (HTTP %d)", change.field, want, got, response.Code)
				}
			})
		}
	}
	t.Run("other fields stay editable", func(t *testing.T) {
		response := ownershipRequest(t, f.handler, asset, f.owner, http.MethodPatch, "json", map[string]any{"lat": 1.5, "lon": 2.5})
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d; want 200: %s", response.Code, response.Body.String())
		}
	})
}
