package thumbs_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// pocketBaseDefaultThumb is served for every image field without being listed
// in the field's thumbs option.
const pocketBaseDefaultThumb = "100x100"

// requestedThumbs mirrors the sizes the web UI asks for through getFileURL.
// PocketBase silently serves the original file for any size that is neither the
// built-in default nor registered on the field, so every size used by a
// component has to appear in the field's thumbs option.
var requestedThumbs = []struct {
	collection string
	field      string
	size       string
	usedBy     string
}{
	{"trails", "photos", "600x0", "trail_info_panel header and photo grid, maplibre_util, feed_card, activity_card, photo_picker, lists/edit"},
	{"users", "avatar", "100x100", "nav_bar, trail_info_panel comment box"},
	{"users", "avatar", "300x300", "settings/profile"},
	{"lists", "avatar", "100x100", "list_search_modal, trail/edit"},
	{"lists", "avatar", "300x300", "list_card, lists/edit load"},
	{"lists", "avatar", "600x0", "list_panel, profile/[handle]"},
	{"waypoints", "photos", "300x300", "waypoint_card"},
	{"waypoints", "photos", "600x0", "trail_timeline, photo_picker"},
	{"summit_logs", "photos", "300x300", "summit_log_table_row"},
	{"summit_logs", "photos", "600x0", "summit_log_card, photo_picker, activity_card"},
}

func TestRequestedThumbSizesAreRegistered(t *testing.T) {
	app := newThumbsTestApp(t)
	for _, requested := range requestedThumbs {
		t.Run(requested.collection+"."+requested.field+"/"+requested.size, func(t *testing.T) {
			field := fileField(t, app, requested.collection, requested.field)
			if requested.size == pocketBaseDefaultThumb {
				return
			}
			if !slices.Contains(field.Thumbs, requested.size) {
				t.Fatalf("%s is requested by %s but is not in %s.%s thumbs %v, so PocketBase serves the original file",
					requested.size, requested.usedBy, requested.collection, requested.field, field.Thumbs)
			}
		})
	}
}

// TestRegisteredThumbSizeIsServed covers the mechanism the table above relies
// on: a registered size is resized, an unregistered one falls back silently.
func TestRegisteredThumbSizeIsServed(t *testing.T) {
	app := newThumbsTestApp(t)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	avatar, err := filesystem.NewFileFromBytes(testPNG(t, 1200, 900), "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.Load(map[string]any{"username": "thumbs", "password": "test-password", "email": "thumbs@example.com"})
	user.Set("avatar", avatar)
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	download := func(query string) int {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/files/users/"+user.Id+"/"+user.GetString("avatar")+query, nil)
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %q = %d, want 200", query, recorder.Code)
		}
		return recorder.Body.Len()
	}

	original := download("")
	if registered := download("?thumb=300x300"); registered >= original {
		t.Errorf("registered thumb served %d bytes, want less than the original %d", registered, original)
	}
	// 500x500 is deliberately absent from the field, which PocketBase answers
	// with the original file and no error.
	if unregistered := download("?thumb=500x500"); unregistered != original {
		t.Errorf("unregistered thumb served %d bytes, want the original %d", unregistered, original)
	}
}

func fileField(t *testing.T, app core.App, collectionName, fieldName string) *core.FileField {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatal(err)
	}
	field, ok := collection.Fields.GetByName(fieldName).(*core.FileField)
	if !ok {
		t.Fatalf("%s.%s is not a file field", collectionName, fieldName)
	}
	return field
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), uint8((x + y) % 256), 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

// newThumbsTestApp builds the current schema in a temporary database, skipping
// only the migrations that configure the external Meilisearch indexes.
func newThumbsTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].File < migrations[j].File
	})
	for _, migration := range migrations {
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}

	return app
}
