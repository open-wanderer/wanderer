package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func newGpxLimitApp(t *testing.T, trails, summitLogs int64) core.App {
	t.Helper()
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunSystemMigrations(); err != nil {
		t.Fatal(err)
	}
	for name, maxSize := range map[string]int64{"trails": trails, "summit_logs": summitLogs} {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: maxSize})
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	return app
}

func gpxLimit(t *testing.T, app core.App, name string) int64 {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}
	return collection.Fields.GetByName("gpx").(*core.FileField).MaxSize
}

func TestRaiseGpxFileLimit(t *testing.T) {
	const fiveMiB, twentyFiveMiB = int64(5 << 20), int64(25 << 20)
	tests := []struct {
		name          string
		before, after int64
	}{
		{"the default", fiveMiB, twentyFiveMiB},
		{"unset", 0, twentyFiveMiB},
		{"raised by an admin", 2 << 30, 2 << 30},
		{"lowered by an admin", 1 << 20, 1 << 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newGpxLimitApp(t, tt.before, tt.before)
			if err := setGpxFileLimit1791800000(app, core.DefaultFileFieldMaxSize, gpxFileLimit1791800000); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"trails", "summit_logs"} {
				if got := gpxLimit(t, app, name); got != tt.after {
					t.Fatalf("%s: maxSize = %d, want %d", name, got, tt.after)
				}
			}
		})
	}
}

func TestRaiseGpxFileLimitHandlesEachCollectionOnItsOwn(t *testing.T) {
	app := newGpxLimitApp(t, 2<<30, 5<<20)
	if err := setGpxFileLimit1791800000(app, core.DefaultFileFieldMaxSize, gpxFileLimit1791800000); err != nil {
		t.Fatal(err)
	}
	if got := gpxLimit(t, app, "trails"); got != 2<<30 {
		t.Fatalf("trails: maxSize = %d", got)
	}
	if got := gpxLimit(t, app, "summit_logs"); got != gpxFileLimit1791800000 {
		t.Fatalf("summit_logs: maxSize = %d", got)
	}
}

func TestLowerGpxFileLimitOnlyUndoesTheRaise(t *testing.T) {
	app := newGpxLimitApp(t, 25<<20, 2<<30)
	if err := setGpxFileLimit1791800000(app, gpxFileLimit1791800000, core.DefaultFileFieldMaxSize); err != nil {
		t.Fatal(err)
	}
	if got := gpxLimit(t, app, "trails"); got != core.DefaultFileFieldMaxSize {
		t.Fatalf("trails: maxSize = %d", got)
	}
	if got := gpxLimit(t, app, "summit_logs"); got != 2<<30 {
		t.Fatalf("summit_logs: maxSize = %d", got)
	}
}
