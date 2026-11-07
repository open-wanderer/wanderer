package routes

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// TrailGpxLimit handles GET /trail/gpx-limit. The web server sizes track
// uploads by the gpx field of the trails collection, whose schema only a
// superuser can read.
func TrailGpxLimit(e *core.RequestEvent) error {
	maxSize, err := trailGpxMaxSize(e.App)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]int64{"maxSize": maxSize})
}

func trailGpxMaxSize(app core.App) (int64, error) {
	trails, err := app.FindCachedCollectionByNameOrId("trails")
	if err != nil {
		return 0, err
	}
	field, ok := trails.Fields.GetByName("gpx").(*core.FileField)
	if !ok {
		return 0, fmt.Errorf("trails has no gpx file field")
	}
	if field.MaxSize <= 0 {
		return core.DefaultFileFieldMaxSize, nil
	}
	return field.MaxSize, nil
}
