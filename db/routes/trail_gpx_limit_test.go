package routes

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func newGpxLimitTestApp(t *testing.T, field core.Field) core.App {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunSystemMigrations(); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(field)
	if err := app.Save(trails); err != nil {
		t.Fatalf("save trails: %v", err)
	}
	return app
}

func TestTrailGpxMaxSizeFollowsTheField(t *testing.T) {
	app := newGpxLimitTestApp(t, &core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 2 << 30})
	got, err := trailGpxMaxSize(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2<<30 {
		t.Fatalf("maxSize = %d", got)
	}
}

func TestTrailGpxMaxSizeUsesPocketBaseDefaultWhenUnset(t *testing.T) {
	app := newGpxLimitTestApp(t, &core.FileField{Name: "gpx", MaxSelect: 1})
	got, err := trailGpxMaxSize(app)
	if err != nil {
		t.Fatal(err)
	}
	if got != core.DefaultFileFieldMaxSize {
		t.Fatalf("maxSize = %d", got)
	}
}

func TestTrailGpxMaxSizeNeedsAFileField(t *testing.T) {
	app := newGpxLimitTestApp(t, &core.TextField{Name: "gpx"})
	if _, err := trailGpxMaxSize(app); err == nil {
		t.Fatal("expected an error")
	}
}
