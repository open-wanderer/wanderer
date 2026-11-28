package main

import (
	"io"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/hooks"
)

type placeholderReadCounter struct {
	source filesystem.FileReader
	opens  int
}

func (r *placeholderReadCounter) Open() (io.ReadSeekCloser, error) { r.opens++; return r.source.Open() }

func TestRoutePlaceholderHookSkipsCollectionsWithoutLegacyPhotoFields(t *testing.T) {
	app := newRealSchemaApp(t)
	for _, name := range []string{"trails", "summit_logs"} {
		t.Run(name, func(t *testing.T) {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				t.Fatal(err)
			}
			if collection.Fields.GetByName("photos") != nil {
				t.Fatal("legacy photos field still exists")
			}
			record := core.NewRecord(collection)
			record.Set("thumbnail", 4)
			file, err := filesystem.NewFileFromBytes([]byte(`<?xml version="1.0"?><gpx version="1.1"><trk><trkseg><trkpt lat="46.6" lon="7.2"/><trkpt lat="46.7" lon="7.3"/></trkseg></trk></gpx>`), "route.gpx")
			if err != nil {
				t.Fatal(err)
			}
			reader := &placeholderReadCounter{source: file.Reader}
			file.Reader = reader
			record.Set("gpx", file)
			event := &core.RecordEvent{App: app}
			event.Record = record
			if err := hooks.RoutePlaceholderHandler()(event); err != nil {
				t.Fatal(err)
			}
			if reader.opens != 0 {
				t.Fatalf("placeholder hook opened GPX %d times without a photos field", reader.opens)
			}
			if record.GetInt("thumbnail") != 4 {
				t.Fatal("placeholder hook changed the thumbnail")
			}
			if record.Get("photos") != nil {
				t.Fatal("placeholder hook wrote an absent photos field")
			}
		})
	}
}
