// Package gpx parses a GPX document into the measurements Wanderer stores for
// a trail.
//
// The package is PocketBase-free: it imports nothing from PocketBase or from
// db/util, so it can be used from any caller and tested in isolation. It
// streams its input through a byte cap and a tolerant stdlib XML decoder into
// compact struct-of-arrays storage, so memory stays bounded by the size limit
// rather than by the shape of the document.
//
// The oracle for every measurement is the corrected TypeScript implementation
// in web/src/lib/models/gpx plus the shared corpus in fixtures/gpx-corpus,
// never a third-party GPX library. Go is a third corpus-exact measurer beside
// TypeScript and Dart.
//
// Parse takes the byte limit as a required argument and has no default. A
// limit of zero or less is a zero-byte budget: any non-empty input is too
// large.
//
// Parse returns (nil, err) on every failure and never a partial result. The
// errors callers can test with errors.Is are ErrTruncated, ErrNoTrack,
// ErrTooLarge, ErrMalformed and ErrTooComplex.
package gpx

import (
	"errors"
	"io"
)

var (
	// ErrTruncated reports a document that ends before its root element closes.
	ErrTruncated = errors.New("gpx: truncated document")
	// ErrNoTrack reports a document without a usable point in its measured
	// source (its tracks, or its routes when it has no track element).
	ErrNoTrack = errors.New("gpx: no usable track or route point")
	// ErrTooLarge reports input over the byte limit, or over a structural cap.
	ErrTooLarge = errors.New("gpx: input exceeds size limit")
	// ErrMalformed reports a document the decoder refuses for any other reason.
	ErrMalformed = errors.New("gpx: malformed document")
	// ErrTooComplex reports a track whose simplification exceeds its work budget.
	ErrTooComplex = errors.New("gpx: track too complex to simplify")
)

// LatLon is a position in degrees, exactly as the file spells it.
type LatLon struct{ Lat, Lon float64 }

// Bounds is the bounding box of the measured points. Its values are the file's
// own and are never clipped or wrapped.
type Bounds struct{ MinLat, MaxLat, MinLon, MaxLon float64 }

// Waypoint is a named point of interest from a <wpt> element.
type Waypoint struct {
	Lat, Lon                float64
	Name, Description, Icon string
}

// Result is the measurement of one GPX document.
type Result struct {
	// Name and Description come from the document's metadata and first track.
	Name, Description string
	// Start is the first usable point of the measured source.
	Start LatLon
	// StartDate is the UTC date ("YYYY-MM-DD") of the first segment's first
	// point, or "" when the first segment has no usable start and end time.
	StartDate string
	// Distance is the raw haversine sum in metres.
	Distance float64
	// ElevationGain and ElevationLoss are the final, filtered totals in metres.
	ElevationGain, ElevationLoss float64
	// DurationMs is the summed leg duration of the measured source.
	DurationMs int64
	// PointCount is the number of usable points of the measured source.
	PointCount int
	Bounds     Bounds
	Centroid   LatLon
	// HasElevation is false when fewer than half of the measured points carry a
	// usable elevation, or when every usable elevation is identical. Gain and
	// loss are never zeroed by it.
	HasElevation bool
	// Waypoints are the usable <wpt> elements, in file order.
	Waypoints []Waypoint

	// tracks is the measured source in file order: the file's tracks, or its
	// routes promoted one segment each when the file has no <trk>.
	tracks []measuredTrack
	// firstTimes is the first and last point time of the first segment.
	firstTimes segmentTimes
}

// Parse reads one GPX document. maxBytes is required and has no default.
// fallbackName names the result when the document names nothing.
func Parse(r io.Reader, maxBytes int64, fallbackName string) (*Result, error) {
	res, err := decode(newPipeline(r, maxBytes), fallbackName)
	if err != nil {
		return nil, err
	}
	return res, nil
}
