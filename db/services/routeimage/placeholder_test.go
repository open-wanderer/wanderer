package routeimage

import (
	"bytes"
	"image/jpeg"
	"io"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/tkrajina/gpxgo/gpx"
)

const trackGPX = `<?xml version="1.0"?>
<gpx version="1.1" creator="test"><trk><trkseg>
<trkpt lat="46.60" lon="7.20"/><trkpt lat="46.61" lon="7.22"/><trkpt lat="46.63" lon="7.21"/>
</trkseg></trk></gpx>`

const otherTrackGPX = `<?xml version="1.0"?>
<gpx version="1.1" creator="test"><trk><trkseg>
<trkpt lat="47.00" lon="8.00"/><trkpt lat="47.02" lon="8.01"/>
</trkseg></trk></gpx>`

const routeGPX = `<?xml version="1.0"?>
<gpx version="1.1" creator="test"><rte>
<rtept lat="46.60" lon="7.20"/><rtept lat="46.62" lon="7.25"/>
</rte></gpx>`

func setupApp(t *testing.T) *pbtests.TestApp {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.BoolField{Name: "is_local"})
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "author"},
		&core.TextField{Name: "name"},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 10 << 20},
		&core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 10 << 20},
		&core.NumberField{Name: "thumbnail"},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	summitLogs := core.NewBaseCollection("summit_logs")
	summitLogs.Fields.Add(
		&core.TextField{Name: "author"},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 10 << 20},
		&core.FileField{Name: "photos", MaxSelect: 99, MaxSize: 10 << 20},
	)
	if err := app.Save(summitLogs); err != nil {
		t.Fatal(err)
	}

	// What hooks.RoutePlaceholderHandler does, minus logging.
	apply := func(e *core.RecordEvent) error {
		Apply(e.App, e.Record)
		return e.Next()
	}
	for _, c := range []string{"trails", "summit_logs"} {
		app.OnRecordCreate(c).BindFunc(apply)
		app.OnRecordUpdate(c).BindFunc(apply)
	}
	return app
}

func newTrail(t *testing.T, app core.App, local bool, gpxContent string, photos ...string) *core.Record {
	t.Helper()
	return newRecord(t, app, "trails", local, gpxContent, photos...)
}

func newRecord(t *testing.T, app core.App, collection string, local bool, gpxContent string, photos ...string) *core.Record {
	t.Helper()
	actors, _ := app.FindCollectionByNameOrId("activitypub_actors")
	author := core.NewRecord(actors)
	author.Set("is_local", local)
	if err := app.Save(author); err != nil {
		t.Fatal(err)
	}

	c, _ := app.FindCollectionByNameOrId(collection)
	record := core.NewRecord(c)
	record.Set("author", author.Id)
	if gpxContent != "" {
		gpxFile, _ := filesystem.NewFileFromBytes([]byte(gpxContent), "track.gpx")
		record.Set("gpx", gpxFile)
	}
	files := []any{}
	for _, name := range photos {
		files = append(files, upload(t, name))
	}
	record.Set("photos", files)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return reload(t, app, record)
}

func upload(t *testing.T, name string) *filesystem.File {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, Render(nil, 0), nil); err != nil {
		t.Fatal(err)
	}
	f, err := filesystem.NewFileFromBytes(buf.Bytes(), name)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func reload(t *testing.T, app core.App, record *core.Record) *core.Record {
	t.Helper()
	fresh, err := app.FindRecordById(record.Collection().Name, record.Id)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func storedFileExists(t *testing.T, app core.App, record *core.Record, name string) bool {
	t.Helper()
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	ok, err := fsys.Exists(record.BaseFilesPath() + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func onlyPlaceholder(t *testing.T, record *core.Record) string {
	t.Helper()
	photos := record.GetStringSlice("photos")
	if len(photos) != 1 || !IsPlaceholder(photos[0]) || !strings.HasSuffix(photos[0], ".jpg") {
		t.Fatalf("photos = %v, want one generated jpg", photos)
	}
	return photos[0]
}

func TestPlaceholderOnCreate(t *testing.T) {
	for _, collection := range []string{"trails", "summit_logs"} {
		t.Run(collection, func(t *testing.T) {
			app := setupApp(t)
			record := newRecord(t, app, collection, true, trackGPX)
			name := onlyPlaceholder(t, record)

			fsys, _ := app.NewFilesystem()
			defer fsys.Close()
			r, err := fsys.GetReader(record.BaseFilesPath() + "/" + name)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			data, _ := io.ReadAll(r)
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Width != Width || cfg.Height != Height {
				t.Fatalf("size = %dx%d, want %dx%d", cfg.Width, cfg.Height, Width, Height)
			}
		})
	}
}

func TestPlaceholderFromRouteOnlyGPX(t *testing.T) {
	app := setupApp(t)
	onlyPlaceholder(t, newTrail(t, app, true, routeGPX))
}

func TestNoPlaceholder(t *testing.T) {
	cases := []struct {
		name   string
		local  bool
		gpx    string
		photos []string
	}{
		{"remote author", false, trackGPX, nil},
		{"author photo", true, trackGPX, []string{"summit.jpg"}},
		{"no route", true, "", nil},
		{"broken route", true, "<gpx", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			app := setupApp(t)
			record := newTrail(t, app, c.local, c.gpx, c.photos...)
			for _, p := range record.GetStringSlice("photos") {
				if IsPlaceholder(p) {
					t.Fatalf("photos = %v, want no placeholder", record.GetStringSlice("photos"))
				}
			}
		})
	}
}

func TestPlaceholderFollowsRoute(t *testing.T) {
	app := setupApp(t)
	trail := newTrail(t, app, true, trackGPX)
	old := onlyPlaceholder(t, trail)

	trail.Set("name", "renamed")
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	trail = reload(t, app, trail)
	if got := onlyPlaceholder(t, trail); got != old {
		t.Fatalf("an unrelated edit rendered the placeholder again: %s -> %s", old, got)
	}

	gpxFile, _ := filesystem.NewFileFromBytes([]byte(otherTrackGPX), "track.gpx")
	trail.Set("gpx", gpxFile)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	trail = reload(t, app, trail)
	if got := onlyPlaceholder(t, trail); got == old {
		t.Fatal("a new route kept the old placeholder")
	}
	if storedFileExists(t, app, trail, old) {
		t.Fatalf("the replaced placeholder %s is still stored", old)
	}
}

// Uploading photos replaces the generated one, with the thumbnail still
// pointing at the photo it pointed at before.
func TestPlaceholderMakesWayForUploads(t *testing.T) {
	app := setupApp(t)
	trail := newTrail(t, app, true, trackGPX)
	placeholder := onlyPlaceholder(t, trail)

	trail.Set("photos+", []any{upload(t, "alpha.jpg"), upload(t, "bravo.jpg")})
	trail.Set("thumbnail", 2)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}

	trail = reload(t, app, trail)
	photos := trail.GetStringSlice("photos")
	if len(photos) != 2 || !strings.HasPrefix(photos[0], "alpha_") || !strings.HasPrefix(photos[1], "bravo_") {
		t.Fatalf("photos = %v, want the two uploads", photos)
	}
	if got := trail.GetInt("thumbnail"); got != 1 {
		t.Fatalf("thumbnail = %d, want 1 (still bravo.jpg)", got)
	}
	if storedFileExists(t, app, trail, placeholder) {
		t.Fatalf("the placeholder %s is still stored", placeholder)
	}

	trail.Set("photos", []string{})
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	onlyPlaceholder(t, reload(t, app, trail))
}

func TestSummitLogPlaceholderMakesWayForUploads(t *testing.T) {
	app := setupApp(t)
	log := newRecord(t, app, "summit_logs", true, trackGPX)
	onlyPlaceholder(t, log)

	log.Set("photos+", upload(t, "summit.jpg"))
	if err := app.Save(log); err != nil {
		t.Fatal(err)
	}
	photos := reload(t, app, log).GetStringSlice("photos")
	if len(photos) != 1 || !strings.HasPrefix(photos[0], "summit_") {
		t.Fatalf("photos = %v, want only the upload", photos)
	}
}

func TestDropPlaceholderThumbnail(t *testing.T) {
	cases := []struct{ thumbnail, want int }{{0, 0}, {1, 0}, {2, 1}}
	collection := core.NewBaseCollection("trails")
	collection.Fields.Add(
		&core.FileField{Name: "photos", MaxSelect: 99},
		&core.NumberField{Name: "thumbnail"},
	)
	for _, c := range cases {
		trail := core.NewRecord(collection)
		trail.Set("photos", []string{"a_1.jpg", placeholderName + "_x.jpg", "b_2.jpg"})
		trail.Set("thumbnail", c.thumbnail)
		DropPlaceholder(trail)
		if got := trail.GetInt("thumbnail"); got != c.want {
			t.Errorf("thumbnail %d: got %d, want %d", c.thumbnail, got, c.want)
		}
		if got := strings.Join(trail.GetStringSlice("photos"), ","); got != "a_1.jpg,b_2.jpg" {
			t.Errorf("photos = %s", got)
		}
	}
}

func TestSegmentsFallsBackToRoutes(t *testing.T) {
	data, err := gpx.ParseBytes([]byte(routeGPX))
	if err != nil {
		t.Fatal(err)
	}
	segments := Segments(data)
	if len(segments) != 1 || len(segments[0]) != 2 {
		t.Fatalf("segments = %v, want the route's two points", segments)
	}
}

func TestRenderEdgeCases(t *testing.T) {
	cases := map[string][][]Point{
		"no segments":      nil,
		"single point":     {{{46.6, 7.2}}},
		"north-south only": {{{46.6, 7.2}, {46.7, 7.2}}},
		"invalid points":   {{{91, 7.2}, {46.6, 181}}},
	}
	for name, segments := range cases {
		t.Run(name, func(t *testing.T) {
			if b := Render(segments, 1).Bounds(); b.Dx() != Width || b.Dy() != Height {
				t.Fatalf("bounds = %v", b)
			}
		})
	}
}
