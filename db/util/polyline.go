package util

import (
	"bytes"
	"io"

	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/twpayne/go-polyline"
)

func ComputePolyline(app core.App, r *core.Record) (string, error) {
	gpxPath := r.GetString("gpx")
	if len(gpxPath) == 0 {
		return "", nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", err
	}
	defer fsys.Close()

	gpxFile, err := fsys.GetReader(r.BaseFilesPath() + "/" + gpxPath)
	if err != nil {
		return "", err
	}
	defer gpxFile.Close()

	content := new(bytes.Buffer)
	if _, err = io.Copy(content, gpxFile); err != nil {
		return "", err
	}

	gpxData, err := gpx.Parse(content)
	if err != nil {
		return "", err
	}

	gpxData.SimplifyTracks(50)
	coordinates := make([][]float64, 0)
	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				coordinates = append(coordinates, []float64{pt.Latitude, pt.Longitude})
			}
		}
	}
	return string(polyline.EncodeCoords(coordinates)), nil
}

func SavePolyline(app core.App, r *core.Record) error {
	encoded, err := ComputePolyline(app, r)
	if err != nil {
		return err
	}
	r.Set("polyline", encoded)
	return app.UnsafeWithoutHooks().Save(r)
}
