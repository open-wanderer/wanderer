package routes

import (
	"pocketbase/util"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestAssetPreviewsRecognizeGeneratedRouteImages(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{
			name: "metadata kind",
			values: map[string]any{
				"file": "photo.jpg",
				"metadata": map[string]any{
					"generated": map[string]any{"kind": "route-preview"},
				},
			},
			want: true,
		},
		{
			name:   "migrated server placeholder",
			values: map[string]any{"file": "wanderer_route_placeholder_abcdefghij.svg", "metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}}},
			want:   true,
		},
		{
			name:   "legacy wanderer route preview filename",
			values: map[string]any{"file": "wanderer-route-preview.webp"},
			want:   true,
		},
		{
			name:   "unmarked route upload",
			values: map[string]any{"file": "route_abcdef123.webp"},
			want:   false,
		},
		{
			name: "external provider with matching filename is a regular photo",
			values: map[string]any{
				"file":              "route_abcdef123.webp",
				"external_provider": "immich",
			},
			want: false,
		},
		{
			name: "taken at with matching filename is a regular photo",
			values: map[string]any{
				"file":     "route_abcdef123.webp",
				"taken_at": time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			want: false,
		},
		{
			name:   "regular photo",
			values: map[string]any{"file": "photo.jpg"},
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := newAssetPreviewRecord(tc.values)
			if got := util.IsGeneratedRoutePreviewAsset(record); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func newAssetPreviewRecord(values map[string]any) *core.Record {
	collection := core.NewBaseCollection("assets")
	collection.Fields.Add(
		&core.TextField{Name: "file"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.DateField{Name: "taken_at"},
		&core.JSONField{Name: "metadata"},
	)

	record := core.NewRecord(collection)
	for key, value := range values {
		record.Set(key, value)
	}
	return record
}
