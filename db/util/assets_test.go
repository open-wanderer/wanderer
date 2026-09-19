package util

import (
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func TestPhotoAssetLinkTargetsTreatsTrailAsContextForNestedTargets(t *testing.T) {
	tests := []struct {
		name      string
		trail     string
		waypoint  string
		summitLog string
		want      []AssetLinkTarget
	}{
		{
			name:  "direct trail asset",
			trail: "trail_id",
			want: []AssetLinkTarget{
				{Collection: "trail_assets", Field: "trail", ID: "trail_id"},
			},
		},
		{
			name:     "waypoint asset with trail context",
			trail:    "trail_id",
			waypoint: "waypoint_id",
			want: []AssetLinkTarget{
				{Collection: "waypoint_assets", Field: "waypoint", ID: "waypoint_id"},
			},
		},
		{
			name:      "summit log asset with trail context",
			trail:     "trail_id",
			summitLog: "summit_log_id",
			want: []AssetLinkTarget{
				{Collection: "summit_log_assets", Field: "summit_log", ID: "summit_log_id"},
			},
		},
		{
			name:     "waypoint asset without explicit trail context",
			waypoint: "waypoint_id",
			want: []AssetLinkTarget{
				{Collection: "waypoint_assets", Field: "waypoint", ID: "waypoint_id"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PhotoAssetLinkTargets(tt.trail, tt.waypoint, tt.summitLog)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("PhotoAssetLinkTargets() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCreatePhotoAssetReusesConcurrentExternalAsset(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(
		&core.TextField{Name: "type", Required: true},
		&core.TextField{Name: "storage_mode", Required: true},
		&core.TextField{Name: "remote_status"},
		&core.TextField{Name: "author", Required: true},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.DateField{Name: "taken_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	assets.Indexes = append(assets.Indexes, "CREATE UNIQUE INDEX idx_assets_external ON assets (author, external_provider, external_id, type) WHERE external_provider != '' AND external_id != ''")
	if err := app.Save(assets); err != nil {
		t.Fatal(err)
	}

	injectedCompetingAsset := false
	app.OnRecordCreate("assets").BindFunc(func(e *core.RecordEvent) error {
		if injectedCompetingAsset || e.Record.GetString("external_id") != "photo-1" {
			return e.Next()
		}
		injectedCompetingAsset = true
		competing := core.NewRecord(e.Record.Collection())
		competing.Set("type", "photo")
		competing.Set("storage_mode", "link_private")
		competing.Set("remote_status", "available")
		competing.Set("author", "actor-1")
		competing.Set("external_provider", "immich")
		competing.Set("external_id", "photo-1")
		if err := e.App.Save(competing); err != nil {
			return err
		}
		return e.Next()
	})

	asset, err := CreatePhotoAsset(app, PhotoAssetInput{
		Author:           "actor-1",
		StorageMode:      "link_private",
		ExternalProvider: "immich",
		ExternalID:       "photo-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if asset == nil || asset.GetString("external_id") != "photo-1" {
		t.Fatalf("unexpected reused asset: %#v", asset)
	}

	records, err := app.FindAllRecords("assets")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d external assets, want 1", len(records))
	}
}
