package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestTrailCardPreviewsBoundPhotosPrioritizeRealThumbnailAndCheckShares(t *testing.T) {
	f := setupHandleLookupTest(t, "https://a.example", http.StatusOK)
	author := f.remoteActor(t, "alice", "b.example")
	trails, _ := f.app.FindCollectionByNameOrId("trails")
	shares := core.NewBaseCollection("trail_link_share")
	shares.Fields.Add(&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1}, &core.TextField{Name: "token"})
	if err := f.app.Save(shares); err != nil {
		t.Fatal(err)
	}
	trails.ViewRule = types.Pointer(`public=true || (@request.query.share!="" && trail_link_share_via_trail.token ?= @request.query.share)`)
	if err := f.app.Save(trails); err != nil {
		t.Fatal(err)
	}
	assets := core.NewBaseCollection("assets")
	assets.ViewRule = types.Pointer("")
	assets.Fields.Add(&core.TextField{Name: "type"}, &core.TextField{Name: "author"}, &core.TextField{Name: "file"}, &core.TextField{Name: "storage_mode"}, &core.JSONField{Name: "metadata"})
	if err := f.app.Save(assets); err != nil {
		t.Fatal(err)
	}
	links := core.NewBaseCollection("trail_assets")
	links.ViewRule = types.Pointer(`trail.public=true || (@request.query.share!="" && trail.trail_link_share_via_trail.token ?= @request.query.share)`)
	links.Fields.Add(&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1}, &core.RelationField{Name: "asset", CollectionId: assets.Id, MaxSelect: 1}, &core.BoolField{Name: "is_thumbnail"}, &core.AutodateField{Name: "created", OnCreate: true}, &core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	if err := f.app.Save(links); err != nil {
		t.Fatal(err)
	}
	populated := f.object(t, "trails", author.Id, "https://b.example", "many00000000001", false, false)
	empty := f.object(t, "trails", author.Id, "https://b.example", "empty0000000001", false, false)
	private := f.object(t, "trails", author.Id, "https://b.example", "private00000001", false, false)
	private.Set("public", false)
	if err := f.app.Save(private); err != nil {
		t.Fatal(err)
	}
	mixedPlaceholder := f.object(t, "trails", author.Id, "https://b.example", "mixed0000000001", false, false)
	generatedOnly := f.object(t, "trails", author.Id, "https://b.example", "route0000000001", false, false)
	f.record(t, "trail_link_share", map[string]any{"trail": private.Id, "token": "valid-share"})
	titleID := ""
	if err := f.app.RunInTransaction(func(app core.App) error {
		for i := 0; i < 1501; i++ {
			asset := core.NewRecord(assets)
			asset.Set("type", "photo")
			asset.Set("author", author.Id)
			asset.Set("storage_mode", "link_public")
			metadata := map[string]any{"unrelated": strings.Repeat("metadata", 500)}
			if i == 0 {
				metadata["generated"] = map[string]any{"kind": "route-preview"}
			}
			asset.Set("metadata", metadata)
			if err := app.Save(asset); err != nil {
				return err
			}
			link := core.NewRecord(links)
			link.Set("trail", populated.Id)
			link.Set("asset", asset.Id)
			link.Set("is_thumbnail", i == 0 || i == 1500)
			if err := app.Save(link); err != nil {
				return err
			}
			if i == 1500 {
				asset.Set("file", "clip.mp4")
				asset.Set("storage_mode", "copy")
				if err := app.Save(asset); err != nil {
					return err
				}
				titleID = asset.Id
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	generated := f.record(t, "assets", map[string]any{"type": "photo", "author": author.Id, "storage_mode": "copy", "file": "wanderer_route_placeholder_abcdefghij.svg", "metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}}})
	f.record(t, "trail_assets", map[string]any{"trail": generatedOnly.Id, "asset": generated.Id, "is_thumbnail": true})
	f.record(t, "trail_assets", map[string]any{"trail": mixedPlaceholder.Id, "asset": generated.Id, "is_thumbnail": true})
	real := f.record(t, "assets", map[string]any{"type": "photo", "author": author.Id, "storage_mode": "copy", "file": "real.jpg"})
	f.record(t, "trail_assets", map[string]any{"trail": mixedPlaceholder.Id, "asset": real.Id})
	request := func(query url.Values) (int, map[string]trailCardPreview, string) {
		t.Helper()
		event := &core.RequestEvent{App: f.app}
		event.Request = httptest.NewRequest(http.MethodGet, "/assets/trail-previews?"+query.Encode(), nil)
		response := httptest.NewRecorder()
		event.Response = response
		if err := AssetTrailPreviews(event); err != nil {
			t.Fatal(err)
		}
		var body struct {
			Previews map[string]trailCardPreview `json:"previews"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return response.Code, body.Previews, response.Body.String()
	}
	ids := strings.Join([]string{populated.Id, empty.Id, private.Id, generatedOnly.Id, mixedPlaceholder.Id, "unknown00000001"}, ",")
	for _, share := range []string{"", "invalid-share", "valid-share"} {
		status, previews, wire := request(url.Values{"ids": {ids}, "share": {share}})
		if status != http.StatusOK {
			t.Fatalf("previewstatus=%d", status)
		}
		many := previews[populated.Id]
		if len(many.Photos) != 3 || !strings.Contains(many.Photos[many.Thumbnail], titleID) {
			t.Fatalf("late real thumbnail was lost: %v", many)
		}
		videoURL, err := url.Parse(many.Photos[many.Thumbnail])
		if err != nil || !strings.HasSuffix(videoURL.Path, "/"+titleID+"/clip.mp4") || !strings.HasPrefix(videoURL.Path, "/api/v1/files/"+assets.Id+"/") {
			t.Fatalf("stored video card URL lost its filename: %s", many.Photos[many.Thumbnail])
		}
		placeholderURL, _ := url.Parse(previews[generatedOnly.Id].Photos[0])
		if !strings.HasSuffix(placeholderURL.Path, "/wanderer_route_placeholder_abcdefghij.svg") {
			t.Fatalf("placeholder-only trail lost its stored fallback URL: %s", placeholderURL)
		}
		mixed := previews[mixedPlaceholder.Id]
		if len(mixed.Photos) != 2 || !strings.Contains(mixed.Photos[0], real.Id+"/real.jpg") || !strings.Contains(mixed.Photos[mixed.Thumbnail], real.Id) {
			t.Fatalf("real photo did not outrank the migrated placeholder: %v", mixed)
		}
		if len(previews[empty.Id].Photos) != 0 || len(previews[generatedOnly.Id].Photos) != 1 {
			t.Fatal("empty/generated-only preview fallback lost")
		}
		_, privatePresent := previews[private.Id]
		if privatePresent != (share == "valid-share") {
			t.Fatalf("private trail previewpresent=%v share=%q", privatePresent, share)
		}
		if _, found := previews["unknown00000001"]; found {
			t.Fatal("unknown record was exposed")
		}
		if len(wire) > 1400 || strings.Contains(wire, "metadata") || strings.Contains(wire, "generated") {
			t.Fatalf("card payload expanded unrelated metadata: %d bytes", len(wire))
		}
		if share != "" && !strings.Contains(many.Photos[0], "share="+share) {
			t.Fatal("share context lost in photo URL")
		}
	}
	for _, invalid := range []string{"wrong/id", func() string {
		ids := []string{}
		for i := 0; i < 101; i++ {
			ids = append(ids, fmt.Sprintf("%015d", i))
		}
		return strings.Join(ids, ",")
	}()} {
		event := &core.RequestEvent{App: f.app}
		event.Request = httptest.NewRequest(http.MethodGet, "/assets/trail-previews?ids="+url.QueryEscape(invalid), nil)
		event.Response = httptest.NewRecorder()
		if err := AssetTrailPreviews(event); err == nil {
			t.Fatal("invalid/unbounded preview selection accepted")
		}
	}
}
