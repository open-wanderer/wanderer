package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Empty actor.user values (remote actors) and empty share back-relations can
// match an anonymous request's empty auth id. Require authentication for every
// ownership/share comparison while preserving public and share-token access.
func init() {
	m.Register(func(app core.App) error {
		return setAnonymousReadRules1789200002(app, false)
	}, func(app core.App) error {
		return setAnonymousReadRules1789200002(app, true)
	})
}

func setAnonymousReadRules1789200002(app core.App, rollback bool) error {
	for _, rules := range []struct {
		collection       string
		oldList, oldView string
		list, view       string
		lockedList       bool
		lockedView       bool
	}{
		{
			collection: "trails",
			oldList:    `author.user = @request.auth.id || public = true || (@request.auth.id != "" && trail_share_via_trail.actor.user ?= @request.auth.id) || (trail_link_share_via_trail.token != "" && trail_link_share_via_trail.token = @request.query.share)`,
			oldView:    `author.user = @request.auth.id || public = true || (@request.auth.id != "" && trail_share_via_trail.actor.user ?= @request.auth.id) || (trail_link_share_via_trail.token != "" && trail_link_share_via_trail.token = @request.query.share)`,
			list:       `public = true || (@request.auth.id != "" && (author.user = @request.auth.id || trail_share_via_trail.actor.user ?= @request.auth.id)) || (trail_link_share_via_trail.token != "" && trail_link_share_via_trail.token = @request.query.share)`,
			view:       `public = true || (@request.auth.id != "" && (author.user = @request.auth.id || trail_share_via_trail.actor.user ?= @request.auth.id)) || (trail_link_share_via_trail.token != "" && trail_link_share_via_trail.token = @request.query.share)`,
		},
		{
			collection: "lists",
			oldList:    `author.user = @request.auth.id || public = true || (@request.auth.id != "" && list_share_via_list.actor.user ?= @request.auth.id)`,
			oldView:    `author.user = @request.auth.id || public = true || (@request.auth.id != "" && list_share_via_list.actor.user ?= @request.auth.id)`,
			list:       `public = true || (@request.auth.id != "" && (author.user = @request.auth.id || list_share_via_list.actor.user ?= @request.auth.id))`,
			view:       `public = true || (@request.auth.id != "" && (author.user = @request.auth.id || list_share_via_list.actor.user ?= @request.auth.id))`,
		},
		{
			collection: "waypoints",
			oldList:    "author = @request.auth.id || trail.author.user ?= @request.auth.id || trail.public ?= true || trail.trail_share_via_trail.actor.user ?= @request.auth.id\n|| \n(trail.trail_link_share_via_trail.token != \"\" && trail.trail_link_share_via_trail.token = @request.query.share)",
			oldView:    "author = @request.auth.id || trail.author.user ?= @request.auth.id || trail.public ?= true || trail.trail_share_via_trail.actor.user ?= @request.auth.id\n|| \n(trail.trail_link_share_via_trail.token != \"\" && trail.trail_link_share_via_trail.token = @request.query.share)",
			list:       `trail.public ?= true || (@request.auth.id != "" && (author = @request.auth.id || trail.author.user ?= @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id)) || (trail.trail_link_share_via_trail.token != "" && trail.trail_link_share_via_trail.token = @request.query.share)`,
			view:       `trail.public ?= true || (@request.auth.id != "" && (author = @request.auth.id || trail.author.user ?= @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id)) || (trail.trail_link_share_via_trail.token != "" && trail.trail_link_share_via_trail.token = @request.query.share)`,
		},
		{
			collection: "summit_logs",
			oldList:    `(@request.auth.id != "" && (author.user = @request.auth.id || trail.author.user = @request.auth.id)) || trail.public = true || trail.trail_share_via_trail.actor.user ?= @request.auth.id`,
			oldView:    `(@request.auth.id != "" && (author.user = @request.auth.id || trail.author.user = @request.auth.id)) || trail.public = true || trail.trail_share_via_trail.actor.user ?= @request.auth.id`,
			list:       `(@request.auth.id != "" && (author.user = @request.auth.id || trail.author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id)) || trail.public = true`,
			view:       `(@request.auth.id != "" && (author.user = @request.auth.id || trail.author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id)) || trail.public = true`,
		},
		{
			collection: "comments",
			oldList:    `((@request.auth.id != "" && trail.author.user = @request.auth.id) || trail.public = true) || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id`,
			oldView:    `((@request.auth.id != "" && trail.author.user = @request.auth.id) || trail.public = true) || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id`,
			list:       `trail.public = true || (@request.auth.id != "" && (trail.author.user = @request.auth.id || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id))`,
			view:       `trail.public = true || (@request.auth.id != "" && (trail.author.user = @request.auth.id || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id))`,
		},
		{
			collection: "trail_share",
			oldList:    `trail.author.user = @request.auth.id || actor.user = @request.auth.id`,
			oldView:    `trail.author.user = @request.auth.id || actor.user = @request.auth.id`,
			list:       `(@request.auth.id != "" && (trail.author.user = @request.auth.id || actor.user = @request.auth.id))`,
			view:       `(@request.auth.id != "" && (trail.author.user = @request.auth.id || actor.user = @request.auth.id))`,
		},
		{
			collection: "list_share",
			oldList:    `list.author.user = @request.auth.id || actor.user = @request.auth.id`,
			oldView:    `list.author.user = @request.auth.id || actor.user = @request.auth.id`,
			list:       `(@request.auth.id != "" && (list.author.user = @request.auth.id || actor.user = @request.auth.id))`,
			view:       `(@request.auth.id != "" && (list.author.user = @request.auth.id || actor.user = @request.auth.id))`,
		},
		{
			collection: "trail_like",
			oldList:    `trail.author.user = @request.auth.id || trail.public = true || trail.trail_share_via_trail.actor.user ?= @request.auth.id || actor.user = @request.auth.id`,
			oldView:    `actor.user = @request.auth.id`,
			list:       `trail.public = true || (@request.auth.id != "" && (trail.author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id || actor.user = @request.auth.id))`,
			view:       `(@request.auth.id != "" && (actor.user = @request.auth.id))`,
		},
		{
			collection: "trails_bounding_box",
			oldView:    `@request.auth.id = user`,
			view:       `@request.auth.id != "" && @request.auth.id = user`,
			lockedList: true,
		},
		{
			collection: "trails_filter",
			oldView:    `@request.auth.id = user`,
			view:       `@request.auth.id != "" && @request.auth.id = user`,
			lockedList: true,
		},
		{
			collection: "trail_link_share",
			oldList:    `trail.author.user = @request.auth.id`,
			oldView:    `trail.author.user = @request.auth.id`,
			list:       `@request.auth.id != "" && trail.author.user = @request.auth.id`,
			view:       `@request.auth.id != "" && trail.author.user = @request.auth.id`,
		},
		{
			collection: "settings",
			oldList:    `user = @request.auth.id`,
			oldView:    `user = @request.auth.id`,
			list:       `@request.auth.id != "" && user = @request.auth.id`,
			view:       `@request.auth.id != "" && user = @request.auth.id`,
		},
		{
			collection: "feed",
			oldList:    `actor.user = @request.auth.id`,
			list:       `@request.auth.id != "" && actor.user = @request.auth.id`,
			lockedView: true,
		},
		{
			collection: "api_tokens",
			oldList:    `user = @request.auth.id`,
			oldView:    `user = @request.auth.id`,
			list:       `@request.auth.id != "" && user = @request.auth.id`,
			view:       `@request.auth.id != "" && user = @request.auth.id`,
		},
	} {
		collection, err := app.FindCollectionByNameOrId(rules.collection)
		if err != nil {
			return err
		}
		if rollback {
			collection.ListRule = types.Pointer(rules.oldList)
			collection.ViewRule = types.Pointer(rules.oldView)
		} else {
			collection.ListRule = types.Pointer(rules.list)
			collection.ViewRule = types.Pointer(rules.view)
		}
		// nil locks a rule to superusers; an empty string would open access.
		if rules.lockedList {
			collection.ListRule = nil
		}
		if rules.lockedView {
			collection.ViewRule = nil
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
