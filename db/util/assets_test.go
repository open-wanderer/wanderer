package util

import (
	"reflect"
	"testing"
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
