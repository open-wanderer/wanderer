package hooks

import (
	"context"
	"database/sql"
	"errors"
	"pocketbase/services/routeimage"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type placeholderWriteKey struct{}

// RoutePlaceholderHandler keeps a route's generated asset in step with the
// saved GPX. Other metadata edits leave the image alone.
func RoutePlaceholderHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Context.Value(placeholderWriteKey{}) == true {
			return e.Next()
		}
		changed := e.Record.IsNew() || len(e.Record.GetUnsavedFiles("gpx")) > 0 || e.Record.GetString("gpx") != e.Record.Original().GetString("gpx")
		originalApp := e.App
		return originalApp.RunInTransaction(func(tx core.App) error {
			e.App = tx
			defer func() { e.App = originalApp }()
			if err := e.Next(); err != nil {
				return err
			}
			logPlaceholderError(tx, e.Record, syncRoutePlaceholder(e.Context, tx, e.Record, changed))
			return nil
		})
	}
}

// RoutePlaceholderLinkHandler handles attachments independently of a later
// target save, including moves between targets and removal of the last photo.
func RoutePlaceholderLinkHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Context.Value(placeholderWriteKey{}) == true {
			return e.Next()
		}
		collection, field := placeholderTarget(e.Record.Collection().Name)
		if collection == "" {
			return e.Next()
		}
		ids := util.UniqueNonEmptyStrings([]string{e.Record.Original().GetString(field), e.Record.GetString(field)})
		originalApp := e.App
		return originalApp.RunInTransaction(func(tx core.App) error {
			e.App = tx
			defer func() { e.App = originalApp }()
			if err := e.Next(); err != nil {
				return err
			}
			// Asset deletion cascades remove links before their targets.
			if _, err := tx.FindRecordById("assets", e.Record.GetString("asset")); errors.Is(err, sql.ErrNoRows) {
				return nil
			} else if err != nil {
				return err
			}
			for _, id := range ids {
				target, err := tx.FindRecordById(collection, id)
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				if err != nil {
					return err
				}
				ctx := e.Context
				if util.DeferTargetSync(tx, collection+"/"+id, func(app core.App) error {
					fresh, err := app.FindRecordById(collection, id)
					if errors.Is(err, sql.ErrNoRows) {
						return nil
					}
					if err != nil {
						return err
					}
					logPlaceholderError(app, fresh, syncRoutePlaceholder(ctx, app, fresh, false))
					return nil
				}) {
					continue
				}
				logPlaceholderError(tx, target, syncRoutePlaceholder(e.Context, tx, target, false))
			}
			return nil
		})
	}
}

func placeholderTarget(links string) (string, string) {
	switch links {
	case "trail_assets":
		return "trails", "trail"
	case "summit_log_assets":
		return "summit_logs", "summit_log"
	}
	return "", ""
}

func logPlaceholderError(app core.App, record *core.Record, err error) {
	if err != nil {
		app.Logger().Error("could not generate route placeholder", "collection", record.Collection().Name, "id", record.Id, "error", err)
	}
}

func syncRoutePlaceholder(ctx context.Context, app core.App, target *core.Record, changed bool) error {
	// The account cascade may have removed the author before this target.
	if _, err := app.FindRecordById("activitypub_actors", target.GetString("author")); errors.Is(err, sql.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	linksName, field := "trail_assets", "trail"
	if target.Collection().Name == "summit_logs" {
		linksName, field = "summit_log_assets", "summit_log"
	}
	links, err := app.FindRecordsByFilter(linksName, field+"={:target}", "created,id", -1, 0, dbx.Params{"target": target.Id})
	if err != nil {
		return err
	}
	var generated []*core.Record
	real := false
	for _, link := range links {
		asset, err := app.FindRecordById("assets", link.GetString("asset"))
		if err != nil {
			return err
		}
		if util.IsGeneratedRoutePreviewAsset(asset) {
			generated = append(generated, link)
		} else {
			real = true
		}
	}
	ctx = context.WithValue(ctx, placeholderWriteKey{}, true)
	drop := func() error {
		for _, link := range generated {
			if err := app.DeleteWithContext(ctx, link); err != nil {
				return err
			}
			if _, err := util.DeleteAssetIfOrphaned(app, link.GetString("asset")); err != nil {
				return err
			}
		}
		return nil
	}
	if real {
		return drop()
	}
	if target.GetString("gpx") == "" || (!changed && len(generated) > 0) {
		return nil
	}
	file, err := routeimage.PlaceholderFile(app, target)
	if err != nil || file == nil {
		return err
	}
	// A failed render leaves the previous image intact. A successful render
	// replaces only this target's generated links; shared assets survive.
	assets, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return err
	}
	asset := core.NewRecord(assets)
	asset.Load(map[string]any{"author": target.GetString("author"), "type": "photo", "storage_mode": "copy", "remote_status": "available", "file": file,
		"metadata": map[string]any{"generated": map[string]any{"kind": "route-preview"}}})
	if err := app.SaveWithContext(ctx, asset); err != nil {
		return err
	}
	if err := drop(); err != nil {
		return err
	}
	collection, err := app.FindCollectionByNameOrId(linksName)
	if err != nil {
		return err
	}
	link := core.NewRecord(collection)
	link.Set("asset", asset.Id)
	link.Set(field, target.Id)
	if linksName == "trail_assets" {
		link.Set("is_thumbnail", true)
	}
	return app.SaveWithContext(ctx, link)
}
