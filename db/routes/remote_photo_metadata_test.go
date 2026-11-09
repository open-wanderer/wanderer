package routes

import "testing"

func TestRemotePhotoListOmitsOriginGeneratedPreviews(t *testing.T) {
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		for _, kind := range []string{"route-preview", "unrecognized"} {
			t.Run(target+"/"+kind, func(t *testing.T) {
				data := map[string]any{"expand": map[string]any{
					target + "_assets_via_" + target: []any{map[string]any{
						"is_thumbnail": true,
						"expand": map[string]any{"asset": map[string]any{
							"id": "photo1", "type": "photo", "file": "map.webp", "collectionId": "assets",
							"metadata": map[string]any{"generated": map[string]any{"kind": kind}},
						}},
					}},
				}}
				photos, authoritative := syncRecordPhotoList(target, "record1", "https://a.example", data, true)
				wantCount := 1
				if kind == "route-preview" {
					wantCount = 0
				}
				if !authoritative || len(photos) != wantCount {
					t.Fatalf("wrong photo list: photos=%v authoritative=%t", photos, authoritative)
				}
				if len(photos) > 0 && !photos[0].IsThumbnail {
					t.Fatal("real thumbnail lost")
				}

			})
		}
	}
}
