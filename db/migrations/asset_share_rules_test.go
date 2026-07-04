package migrations

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAssetReadRuleIncludesTrailSharing(t *testing.T) {
	for _, want := range []string{
		"trail_assets_via_asset.trail.trail_share_via_trail.actor.user ?= @request.auth.id",
		"trail_assets_via_asset.trail.trail_link_share_via_trail.token ?= @request.query.share",
		"waypoint_assets_via_asset.waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id",
		"waypoint_assets_via_asset.waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share",
		"summit_log_assets_via_asset.summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id",
		"summit_log_assets_via_asset.summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share",
	} {
		if !strings.Contains(assetReadRule, want) {
			t.Fatalf("assetReadRule does not contain %q", want)
		}
	}
}

func TestAssetReadRuleUsesAnyMatchForAssetBackRelations(t *testing.T) {
	for _, want := range []string{
		"trail_assets_via_asset.trail.author.user ?= @request.auth.id",
		"trail_assets_via_asset.trail.public ?= true",
		"waypoint_assets_via_asset.waypoint.trail.author.user ?= @request.auth.id",
		"waypoint_assets_via_asset.waypoint.author.user ?= @request.auth.id",
		"waypoint_assets_via_asset.waypoint.trail.public ?= true",
		"summit_log_assets_via_asset.summit_log.author.user ?= @request.auth.id",
		"summit_log_assets_via_asset.summit_log.trail.author.user ?= @request.auth.id",
		"summit_log_assets_via_asset.summit_log.trail.public ?= true",
	} {
		if !strings.Contains(assetReadRule, want) {
			t.Fatalf("assetReadRule does not contain any-match condition %q", want)
		}
	}
	for _, forbidden := range []string{
		"trail_assets_via_asset.trail.author.user = @request.auth.id",
		"trail_assets_via_asset.trail.public = true",
		"waypoint_assets_via_asset.waypoint.trail.author.user = @request.auth.id",
		"waypoint_assets_via_asset.waypoint.author.user = @request.auth.id",
		"waypoint_assets_via_asset.waypoint.trail.public = true",
		"summit_log_assets_via_asset.summit_log.author.user = @request.auth.id",
		"summit_log_assets_via_asset.summit_log.trail.author.user = @request.auth.id",
		"summit_log_assets_via_asset.summit_log.trail.public = true",
	} {
		if strings.Contains(assetReadRule, forbidden) {
			t.Fatalf("assetReadRule still contains all-match condition %q", forbidden)
		}
	}
}

func TestAssetLinkRulesSeparateReadAndWriteSharing(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"trail_assets", trailAssetLinkCollectionJSON()},
		{"waypoint_assets", waypointAssetLinkCollectionJSON()},
		{"summit_log_assets", summitLogAssetLinkCollectionJSON()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var collection map[string]any
			if err := json.Unmarshal([]byte(tt.data), &collection); err != nil {
				t.Fatalf("invalid collection JSON: %v", err)
			}

			for _, field := range []string{"listRule", "viewRule"} {
				rule, _ := collection[field].(string)
				if !strings.Contains(rule, "trail_share_via_trail.actor.user ?= @request.auth.id") {
					t.Fatalf("%s %s does not allow view shares: %s", tt.name, field, rule)
				}
				if !strings.Contains(rule, "trail_link_share_via_trail.token ?= @request.query.share") {
					t.Fatalf("%s %s does not allow link shares: %s", tt.name, field, rule)
				}
				if strings.Contains(rule, `permission = "edit"`) {
					t.Fatalf("%s %s still requires edit permission for read access: %s", tt.name, field, rule)
				}
			}

			for _, field := range []string{"createRule", "deleteRule", "updateRule"} {
				rule, _ := collection[field].(string)
				if !strings.Contains(rule, `permission = "edit"`) {
					t.Fatalf("%s %s does not require edit permission for writes: %s", tt.name, field, rule)
				}
				if strings.Contains(rule, "trail_link_share_via_trail") {
					t.Fatalf("%s %s allows link shares to write: %s", tt.name, field, rule)
				}
			}
		})
	}
}
