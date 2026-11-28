package migrations

import (
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		shares, err := app.FindCollectionByNameOrId("trail_share")
		if err != nil {
			return err
		}
		shares.Fields.Add(&core.TextField{Name: "federation_token", Hidden: true, Max: 32}, &core.TextField{Name: "remote_token", Hidden: true, Max: 32})
		if err := app.Save(shares); err != nil {
			return err
		}
		for collection, prefix := range map[string]string{
			"trails":            "trail_share_via_trail",
			"waypoints":         "trail.trail_share_via_trail",
			"summit_logs":       "trail.trail_share_via_trail",
			"comments":          "trail.trail_share_via_trail",
			"trail_assets":      "trail.trail_share_via_trail",
			"waypoint_assets":   "waypoint.trail.trail_share_via_trail",
			"summit_log_assets": "summit_log.trail.trail_share_via_trail",
			"assets":            "",
		} {
			c, err := app.FindCollectionByNameOrId(collection)
			if err != nil {
				return err
			}
			condition := ""
			if collection == "assets" {
				condition = `trail_assets_via_asset.trail.trail_share_via_trail.federation_token ?= @request.query.share || waypoint_assets_via_asset.waypoint.trail.trail_share_via_trail.federation_token ?= @request.query.share || summit_log_assets_via_asset.summit_log.trail.trail_share_via_trail.federation_token ?= @request.query.share`
			} else {
				condition = prefix + `.federation_token ?= @request.query.share`
			}
			for _, rule := range []**string{&c.ListRule, &c.ViewRule} {
				if *rule != nil && **rule != "" {
					*rule = types.Pointer(fmt.Sprintf(`(%s) || (@request.query.share != "" && (%s))`, **rule, condition))
				}
			}
			if err := app.Save(c); err != nil {
				return err
			}
		}
		return nil
	}, nil)
}
