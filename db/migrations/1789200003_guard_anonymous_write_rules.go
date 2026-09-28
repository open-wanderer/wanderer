package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Follow-up to 1789200002: the same empty-value match also affects the write
// rules. Remote actors have no local user, so `author.user = @request.auth.id`
// is true for an anonymous request against every federated trail, list, and
// comment. Require authentication for every ownership comparison in create,
// update, and delete rules.
func init() {
	m.Register(func(app core.App) error {
		return setAnonymousWriteRules1789200003(app, false)
	}, func(app core.App) error {
		return setAnonymousWriteRules1789200003(app, true)
	})
}

func setAnonymousWriteRules1789200003(app core.App, rollback bool) error {
	type rule struct {
		old, new string
	}
	for _, rules := range []struct {
		collection             string
		create, update, delete *rule
	}{
		{
			collection: "trails",
			update: &rule{
				`author.user = @request.auth.id || (@request.auth.id != "" && trail_share_via_trail.actor.user ?= @request.auth.id && trail_share_via_trail.permission ?= "edit")`,
				`@request.auth.id != "" && (author.user = @request.auth.id || (trail_share_via_trail.actor.user ?= @request.auth.id && trail_share_via_trail.permission ?= "edit"))`,
			},
			delete: &rule{
				`author.user = @request.auth.id `,
				`@request.auth.id != "" && author.user = @request.auth.id`,
			},
		},
		{
			collection: "lists",
			update: &rule{
				`author.user = @request.auth.id || (@request.auth.id != "" && list_share_via_list.actor.user ?= @request.auth.id && list_share_via_list.permission ?= "edit")`,
				`@request.auth.id != "" && (author.user = @request.auth.id || (list_share_via_list.actor.user ?= @request.auth.id && list_share_via_list.permission ?= "edit"))`,
			},
			delete: &rule{
				`author.user = @request.auth.id `,
				`@request.auth.id != "" && author.user = @request.auth.id`,
			},
		},
		{
			collection: "comments",
			update: &rule{
				`@request.auth.id = author.user`,
				`@request.auth.id != "" && @request.auth.id = author.user`,
			},
			delete: &rule{
				`@request.auth.id = author.user`,
				`@request.auth.id != "" && @request.auth.id = author.user`,
			},
		},
		{
			collection: "trail_share",
			create:     &rule{`trail.author.user = @request.auth.id`, `@request.auth.id != "" && trail.author.user = @request.auth.id`},
			update:     &rule{`trail.author.user = @request.auth.id`, `@request.auth.id != "" && trail.author.user = @request.auth.id`},
			delete:     &rule{`trail.author.user = @request.auth.id`, `@request.auth.id != "" && trail.author.user = @request.auth.id`},
		},
		{
			collection: "list_share",
			create:     &rule{`list.author.user = @request.auth.id`, `@request.auth.id != "" && list.author.user = @request.auth.id`},
			update:     &rule{`list.author.user = @request.auth.id`, `@request.auth.id != "" && list.author.user = @request.auth.id`},
			delete:     &rule{`list.author.user = @request.auth.id`, `@request.auth.id != "" && list.author.user = @request.auth.id`},
		},
		{
			collection: "trail_link_share",
			create:     &rule{`trail.author.user = @request.auth.id`, `@request.auth.id != "" && trail.author.user = @request.auth.id`},
			update:     &rule{`trail.author.user = @request.auth.id`, `@request.auth.id != "" && trail.author.user = @request.auth.id`},
			delete:     &rule{`trail.author.user = @request.auth.id`, `@request.auth.id != "" && trail.author.user = @request.auth.id`},
		},
		{
			collection: "trail_like",
			delete:     &rule{`actor.user = @request.auth.id`, `@request.auth.id != "" && actor.user = @request.auth.id`},
		},
		{
			collection: "follows",
			create:     &rule{`@request.auth.id = follower.user.id`, `@request.auth.id != "" && @request.auth.id = follower.user.id`},
			update:     &rule{`@request.auth.id = follower.user.id`, `@request.auth.id != "" && @request.auth.id = follower.user.id`},
			delete:     &rule{`@request.auth.id = follower.user.id`, `@request.auth.id != "" && @request.auth.id = follower.user.id`},
		},
		{
			collection: "notifications",
			update: &rule{
				`@request.auth.id = recipient.user && (@request.body.type = null||@request.body.type = type) && (@request.body.metadata = null||@request.body.metadata = metadata) && (@request.body.recipient = null||@request.body.recipient = recipient) && (@request.body.author = null||@request.body.author = author) && @request.body.seen = true`,
				`@request.auth.id != "" && @request.auth.id = recipient.user && (@request.body.type = null||@request.body.type = type) && (@request.body.metadata = null||@request.body.metadata = metadata) && (@request.body.recipient = null||@request.body.recipient = recipient) && (@request.body.author = null||@request.body.author = author) && @request.body.seen = true`,
			},
		},
		{
			collection: "settings",
			update:     &rule{`user = @request.auth.id`, `@request.auth.id != "" && user = @request.auth.id`},
		},
		{
			collection: "api_tokens",
			create:     &rule{`user = @request.auth.id`, `@request.auth.id != "" && user = @request.auth.id`},
			delete:     &rule{`user = @request.auth.id`, `@request.auth.id != "" && user = @request.auth.id`},
		},
	} {
		collection, err := app.FindCollectionByNameOrId(rules.collection)
		if err != nil {
			return err
		}
		for _, target := range []struct {
			field **string
			rule  *rule
		}{
			{&collection.CreateRule, rules.create},
			{&collection.UpdateRule, rules.update},
			{&collection.DeleteRule, rules.delete},
		} {
			if target.rule == nil {
				continue
			}
			if rollback {
				*target.field = types.Pointer(target.rule.old)
			} else {
				*target.field = types.Pointer(target.rule.new)
			}
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}
	return nil
}
