package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Missing asset back-relations and remote actors have empty user IDs, just like
// anonymous requests. Only authenticated requests may satisfy ownership checks.
func init() {
	m.Register(func(app core.App) error {
		return setAnonymousAssetReadRules1789200004(app, false)
	}, func(app core.App) error {
		return setAnonymousAssetReadRules1789200004(app, true)
	})
}

func setAnonymousAssetReadRules1789200004(app core.App, rollback bool) error {
	for _, rules := range []struct {
		collection, previous, current string
	}{
		{
			collection: "assets",
			previous:   assetReadRule,
			current: `(@request.auth.id != "" && (
				author.user = @request.auth.id ||
				trail_assets_via_asset.trail.author.user ?= @request.auth.id ||
				trail_assets_via_asset.trail.trail_share_via_trail.actor.user ?= @request.auth.id ||
				waypoint_assets_via_asset.waypoint.author.user ?= @request.auth.id ||
				waypoint_assets_via_asset.waypoint.trail.author.user ?= @request.auth.id ||
				waypoint_assets_via_asset.waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id ||
				summit_log_assets_via_asset.summit_log.author.user ?= @request.auth.id ||
				summit_log_assets_via_asset.summit_log.trail.author.user ?= @request.auth.id ||
				summit_log_assets_via_asset.summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) ||
			trail_assets_via_asset.trail.public ?= true ||
			waypoint_assets_via_asset.waypoint.trail.public ?= true ||
			summit_log_assets_via_asset.summit_log.trail.public ?= true ||
			(@request.query.share != "" && (
				trail_assets_via_asset.trail.trail_link_share_via_trail.token ?= @request.query.share ||
				waypoint_assets_via_asset.waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share ||
				summit_log_assets_via_asset.summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share
			))`,
		},
		{
			collection: "trail_assets",
			previous:   trailAssetReadRule,
			current: `(@request.auth.id != "" && (
				asset.author.user = @request.auth.id || trail.author.user = @request.auth.id ||
				trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) || trail.public = true ||
			(@request.query.share != "" && trail.trail_link_share_via_trail.token ?= @request.query.share)`,
		},
		{
			collection: "waypoint_assets",
			previous:   waypointAssetReadRule,
			current: `(@request.auth.id != "" && (
				asset.author.user = @request.auth.id || waypoint.author.user = @request.auth.id ||
				waypoint.trail.author.user = @request.auth.id ||
				waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) || waypoint.trail.public = true ||
			(@request.query.share != "" && waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share)`,
		},
		{
			collection: "summit_log_assets",
			previous:   summitLogAssetReadRule,
			current: `(@request.auth.id != "" && (
				asset.author.user = @request.auth.id || summit_log.author.user = @request.auth.id ||
				summit_log.trail.author.user = @request.auth.id ||
				summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) || summit_log.trail.public = true ||
			(@request.query.share != "" && summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share)`,
		},
	} {
		collection, err := app.FindCollectionByNameOrId(rules.collection)
		if err != nil {
			return err
		}
		rule := rules.current
		if rollback {
			rule = rules.previous
		}
		collection.ListRule = types.Pointer(rule)
		collection.ViewRule = types.Pointer(rule)
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
