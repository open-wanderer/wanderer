package migrations

import (
	"context"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/logger"
)

// Collection saves emit internal debug messages. A synchronous logger batch
// flush needs the auxiliary writer already held by the migration runner. Keep
// those messages out of the batch and restore the actual handler threshold,
// including PocketBase's development-mode threshold, after each migration.
func withPhotoMigrationLogLevel(migrate func(core.App) error) func(core.App) error {
	return func(app core.App) error {
		handler, ok := app.Logger().Handler().(*logger.BatchHandler)
		ctx := context.Background()
		if !ok || !handler.Enabled(ctx, slog.LevelInfo-1) {
			return migrate(app)
		}
		// BatchHandler exposes SetLevel but no getter. Its Enabled predicate is
		// monotonic; locate the exact threshold within PocketBase's integer range.
		low, high := int64(-1<<31), int64(slog.LevelInfo)
		for low < high {
			middle := low + (high-low)/2
			if handler.Enabled(ctx, slog.Level(middle)) {
				high = middle
			} else {
				low = middle + 1
			}
		}
		original := slog.Level(low)
		handler.SetLevel(slog.LevelInfo)
		defer handler.SetLevel(original)
		return migrate(app)
	}
}
