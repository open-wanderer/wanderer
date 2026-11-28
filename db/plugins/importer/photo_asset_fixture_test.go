package importer

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

func newPhotoAssetImportTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	assets := core.NewBaseCollection("assets")
	assets.Fields.Add(
		&core.TextField{Name: "type"},
		&core.TextField{Name: "storage_mode"},
		&core.TextField{Name: "remote_status"},
		&core.TextField{Name: "author"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.JSONField{Name: "metadata"},
		&core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1024},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	if err := app.Save(assets); err != nil {
		t.Fatal(err)
	}
	return app
}
