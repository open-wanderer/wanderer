package routeimage

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"image/jpeg"
	"io"
	"slices"
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

// Apply keeps the generated photo of a trail or summit log in step with its
// route, before the record is written. It runs inside the save so that the
// Create or Update activity sent for the record already carries the image: a
// separate, later Update could reach a remote instance before the Create,
// which would then clear the photos again.
//
// It drops the placeholder once the author adds photos of their own, and
// renders one when the record has a route but no photos: on create, when
// the route changes, and when the author removes the last photo. Records of
// remote authors are left alone; their origin instance sends its own photos.
func Apply(app core.App, record *core.Record) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	DropPlaceholder(record)
	if !needsPlaceholder(record) || !outdated(record) {
		return nil
	}
	author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	if err != nil {
		return fmt.Errorf("find author: %w", err)
	}
	if !author.GetBool("is_local") {
		return nil
	}

	content, err := readGPX(app, record)
	if err != nil {
		return err
	}
	data, err := gpx.ParseBytes(content)
	if err != nil {
		return fmt.Errorf("parse gpx: %w", err)
	}
	segments := Segments(data)
	if len(segments) == 0 {
		return nil
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, Render(segments, seed(content)), &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	file, err := filesystem.NewFileFromBytes(buf.Bytes(), placeholderName+".jpg")
	if err != nil {
		return err
	}
	record.Set("photos", []any{file})
	if hasThumbnail(record) {
		record.Set("thumbnail", 0)
	}
	return nil
}

// needsPlaceholder is true when the record has a route and either no photos
// or only an earlier placeholder.
func needsPlaceholder(record *core.Record) bool {
	if !hasGPX(record) {
		return false
	}
	photos := photoEntries(record)
	if len(photos) == 0 {
		return true
	}
	name, ok := photos[0].(string)
	return len(photos) == 1 && ok && IsPlaceholder(name)
}

// outdated is true when the record has no placeholder for its current route.
// Other edits keep the one it has.
func outdated(record *core.Record) bool {
	if record.IsNew() || len(record.GetUnsavedFiles("gpx")) > 0 {
		return true
	}
	original := record.Original()
	if record.GetString("gpx") != original.GetString("gpx") {
		return true
	}
	return len(photoEntries(record)) == 0
}

func hasGPX(record *core.Record) bool {
	return record.GetString("gpx") != "" || len(record.GetUnsavedFiles("gpx")) > 0
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

// DropPlaceholder removes a generated photo once the author adds their own.
// On trails the thumbnail keeps pointing at the same photo.
func DropPlaceholder(record *core.Record) {
	photos := photoEntries(record)
	if len(photos) < 2 {
		return
	}
	i := slices.IndexFunc(photos, func(p any) bool {
		name, ok := p.(string)
		return ok && IsPlaceholder(name)
	})
	if i < 0 {
		return
	}

	record.Set("photos", slices.Delete(photos, i, i+1))
	if !hasThumbnail(record) {
		return
	}
	thumbnail := record.GetInt("thumbnail")
	switch {
	case thumbnail == i:
		thumbnail = 0
	case thumbnail > i:
		thumbnail--
	}
	record.Set("thumbnail", thumbnail)
}

// hasThumbnail is true for trails, which pick their cover photo by index.
func hasThumbnail(record *core.Record) bool {
	return record.Collection().Fields.GetByName("thumbnail") != nil
}

// photoEntries lists the stored names and, before a save, the files uploaded
// with it. GetStringSlice drops the whole list while it holds files.
func photoEntries(record *core.Record) []any {
	switch v := record.GetRaw("photos").(type) {
	case []any:
		return slices.Clone(v)
	case []string:
		entries := make([]any, len(v))
		for i, name := range v {
			entries[i] = name
		}
		return entries
	case string:
		if v != "" {
			return []any{v}
		}
	}
	return nil
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
