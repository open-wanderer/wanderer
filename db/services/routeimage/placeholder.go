package routeimage

import (
	"fmt"
	"hash/fnv"
	"io"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/tkrajina/gpxgo/gpx"
)

// placeholderName is the stored name of a generated photo up to PocketBase's
// random suffix, so generated photos can be told apart from uploaded ones.
const placeholderName = "wanderer_route_placeholder"

func IsPlaceholder(filename string) bool {
	return strings.HasPrefix(filename, placeholderName+"_")
}

// PlaceholderFile renders the current route. Routes without usable points do
// not get an image. The caller owns the asset and target-link writes.
func PlaceholderFile(app core.App, record *core.Record) (*filesystem.File, error) {
	content, err := readGPX(app, record)
	if err != nil {
		return nil, err
	}
	data, err := gpx.ParseBytes(content)
	if err != nil {
		return nil, fmt.Errorf("parse gpx: %w", err)
	}
	segments := Segments(data)
	if len(segments) == 0 {
		return nil, nil
	}
	return filesystem.NewFileFromBytes(Render(segments, seed(content)), placeholderName+".svg")
}

// readGPX reads the route uploaded with this save, or else the stored one.
func readGPX(app core.App, record *core.Record) ([]byte, error) {
	var reader io.ReadCloser
	if uploads := record.GetUnsavedFiles("gpx"); len(uploads) > 0 {
		r, err := uploads[len(uploads)-1].Reader.Open()
		if err != nil {
			return nil, fmt.Errorf("open uploaded gpx: %w", err)
		}
		reader = r
	} else {
		fsys, err := app.NewFilesystem()
		if err != nil {
			return nil, fmt.Errorf("open filesystem: %w", err)
		}
		defer fsys.Close()
		r, err := fsys.GetReader(record.BaseFilesPath() + "/" + record.GetString("gpx"))
		if err != nil {
			return nil, fmt.Errorf("open gpx: %w", err)
		}
		reader = r
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read gpx: %w", err)
	}
	return content, nil
}

// Segments returns the GPX tracks, or its routes if it has no track points.
func Segments(data *gpx.GPX) [][]Point {
	var segments [][]Point
	for _, trk := range data.Tracks {
		for _, seg := range trk.Segments {
			segments = appendSegment(segments, seg.Points)
		}
	}
	if len(segments) == 0 {
		for _, rte := range data.Routes {
			segments = appendSegment(segments, rte.Points)
		}
	}
	return segments
}

func appendSegment(segments [][]Point, points []gpx.GPXPoint) [][]Point {
	if len(points) == 0 {
		return segments
	}
	seg := make([]Point, len(points))
	for i, p := range points {
		seg[i] = Point{Lat: p.Latitude, Lon: p.Longitude}
	}
	return append(segments, seg)
}

// seed gives every route its own contour pattern that stays the same when
// the placeholder is rendered again.
func seed(gpxContent []byte) float64 {
	h := fnv.New32a()
	h.Write(gpxContent)
	return float64(h.Sum32()%10000) / 100
}
