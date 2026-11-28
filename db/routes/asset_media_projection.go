package routes

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"pocketbase/util"
	"strings"
)

type assetFileProjectionApp struct {
	core.App
	assetID      string
	collectionID string
}

func (app assetFileProjectionApp) FindRecordById(collection any, id string, filters ...func(*dbx.SelectQuery) error) (*core.Record, error) {
	var matches bool
	switch value := collection.(type) {
	case string:
		matches = value == "assets" || value == app.collectionID
	case *core.Collection:
		matches = value.Id == app.collectionID
	}
	if matches && id == app.assetID && len(filters) == 0 {
		return util.FindAssetMediaRecord(app.App, id)
	}
	return app.App.FindRecordById(collection, id, filters...)
}

// Keep PocketBase's complete file handler (tokens, thumbnails, range/cache and
// download hooks), substituting only its initial asset row projection.
func ProjectAssetFileDownloadRecord(e *core.RequestEvent) error {
	if (e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead) || !strings.HasPrefix(e.Request.URL.Path, "/api/files/") {
		return e.Next()
	}
	collection, err := e.App.FindCachedCollectionByNameOrId(e.Request.PathValue("collection"))
	if err != nil || collection.Name != "assets" {
		return e.Next()
	}
	original := e.App
	e.App = assetFileProjectionApp{App: original, assetID: e.Request.PathValue("recordId"), collectionID: collection.Id}
	defer func() { e.App = original }()
	return e.Next()
}
