package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Allow summit-log expansion through trail share links, after the preceding
// security migration has guarded ownership and actor-share comparisons.
func init() {
	m.Register(func(app core.App) error {
		return setSummitLogReadRule1789200003(app, `(@request.auth.id != "" && (author.user = @request.auth.id || trail.author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id)) || trail.public = true || (trail.trail_link_share_via_trail.token != "" && trail.trail_link_share_via_trail.token = @request.query.share)`)
	}, func(app core.App) error {
		return setSummitLogReadRule1789200003(app, `(@request.auth.id != "" && (author.user = @request.auth.id || trail.author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id)) || trail.public = true`)
	})
}

func setSummitLogReadRule1789200003(app core.App, rule string) error {
	collection, err := app.FindCollectionByNameOrId("dd2l9a4vxpy2ni8")
	if err != nil {
		return err
	}
	collection.ListRule = types.Pointer(rule)
	collection.ViewRule = types.Pointer(rule)
	return app.Save(collection)
}
