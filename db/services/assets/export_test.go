package assets

import (
	"context"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/util"
)

func SetRemotePhotoMediaFetcherForTest(fetch func(context.Context, pluginsystem.Photo, importer.Options, int64) (*util.SafeFetchResult, error)) func() {
	original := fetchRemotePhotoMedia
	fetchRemotePhotoMedia = fetch
	return func() { fetchRemotePhotoMedia = original }
}

func MaterializeTrailWithByteLimitForTest(ctx context.Context, app core.App, trailID string, maxBytes int64) error {
	return materializePrivateRemotePluginAssetsForTrail(ctx, app, trailID, maxBytes)
}
