package main

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"pocketbase/hooks"
)

// Include the native file policy when building this download fixture's router.
func newAssetDownloadRouter(app core.App) (*router.Router[*core.RequestEvent], error) {
	api, err := apis.NewRouter(app)
	if err != nil {
		return nil, err
	}
	api.BindFunc(hooks.PrepareAssetFileResponse)
	return api, nil
}
