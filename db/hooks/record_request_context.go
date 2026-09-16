package hooks

import (
	"context"

	"github.com/pocketbase/pocketbase/core"
)

// UseTrailRequestContext carries cancellation into the record hooks and final
// database write. PocketBase's collection API initializes its upsert form with
// a background context, but applies the request event's App before submitting.
func UseTrailRequestContext(e *core.RecordRequestEvent) error {
	ctx := e.Request.Context()
	if err := ctx.Err(); err != nil {
		return err
	}
	app := e.App
	e.App = requestContextApp{App: app, ctx: ctx}
	defer func() { e.App = app }()
	return e.Next()
}

type requestContextApp struct {
	core.App
	ctx context.Context
}

func (app requestContextApp) SaveWithContext(_ context.Context, model core.Model) error {
	return app.App.SaveWithContext(app.ctx, model)
}
