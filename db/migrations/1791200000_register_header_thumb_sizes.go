package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The trail and list headers render wider than 600px, and twice that on HiDPI
// screens, so they offer a 1200px variant through srcset next to 600x0.
var headerThumbSizes = map[string]map[string][]string{
	"trails": {"photos": {"600x0", "1200x0"}},
	"lists":  {"avatar": {"300x300", "600x0", "1200x0"}},
}

var previousHeaderThumbSizes = map[string]map[string][]string{
	"trails": {"photos": {"600x0"}},
	"lists":  {"avatar": {"300x300", "600x0"}},
}

func init() {
	m.Register(func(app core.App) error {
		return applyThumbSizes(app, headerThumbSizes)
	}, func(app core.App) error {
		return applyThumbSizes(app, previousHeaderThumbSizes)
	})
}
