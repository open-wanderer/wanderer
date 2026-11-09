package gpx

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"strconv"
)

// flushThreshold is the size at which the encoder hands its buffer to gzip. The
// buffer never holds more than this plus one position or one feature header.
const flushThreshold = 32 << 10

// geoJSONWriter streams JSON text into a gzip writer through one reused buffer.
// The first write error is kept and every later write is skipped.
type geoJSONWriter struct {
	gz  *gzip.Writer
	buf []byte
	err error
}

// flush writes the buffer to gzip and empties it.
func (g *geoJSONWriter) flush() {
	if g.err == nil && len(g.buf) > 0 {
		_, g.err = g.gz.Write(g.buf)
	}
	g.buf = g.buf[:0]
}

// maybeFlush flushes once the buffer has passed the threshold.
func (g *geoJSONWriter) maybeFlush() {
	if len(g.buf) >= flushThreshold {
		g.flush()
	}
}

func (g *geoJSONWriter) str(s string) { g.buf = append(g.buf, s...) }

func (g *geoJSONWriter) float(v float64) { g.buf = strconv.AppendFloat(g.buf, v, 'f', -1, 64) }

// appendJSONString appends s as a JSON string literal. It goes through
// encoding/json so a hostile name or description cannot leave its string, and
// leaves '<', '>' and '&' alone because the file is data, not HTML.
func appendJSONString(dst []byte, s string) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil { // cannot happen for a string
		return append(dst, `""`...)
	}
	out := b.Bytes()
	return append(dst, out[:len(out)-1]...) // drop the newline Encode adds
}

// stringProperty writes `,"key":"value"` unless value is empty.
func (g *geoJSONWriter) stringProperty(key, value string) {
	if value == "" {
		return
	}
	g.buf = append(g.buf, '"')
	g.buf = append(g.buf, key...)
	g.buf = append(g.buf, '"', ':')
	g.buf = appendJSONString(g.buf, value)
	g.buf = append(g.buf, ',')
}

// EncodeGeoJSON writes the measured tracks of r to w as one gzipped GeoJSON
// FeatureCollection in the shape of GPX.toGeoJSON() in web/src/lib/models/gpx.
//
// There is one LineString Feature per stored segment of each measured track,
// in file order. A route-only file is measured as promoted tracks, so its
// routes appear as track features named from the route; the routes of a file
// that also has a track are not part of the measured source and are not
// written. Waypoints are never written: they are separate records.
//
// A position is [lon, lat, ele] when the point has a usable elevation and
// [lon, lat] otherwise. No elevation is fabricated, not even for a track whose
// elevation marker is "none"; its real values are kept. Coordinates use the
// shortest decimal form that round-trips and are never rounded.
//
// The properties of a Feature are, in this order: name, desc and type (each
// omitted when empty), number (only when the file had a usable one), featureId
// and segmentId (both the segment's position among its track's segments, so a
// skipped empty segment does not renumber the later ones), coordinateProperties
// with times (an ISO string with milliseconds and Z, or null), and anchors.
// Years outside 0000-9999 use the six-digit signed form, exactly as toISOString
// does.
//
// anchors is always present. It lists indices into that Feature's own
// coordinates for the points marked as route anchors, at positions 1 through
// len-2 only, which are the only ones the leg duration honours. A point
// without a position is dropped, so the leg-summed duration can be rebuilt
// exactly from times plus anchors only for a segment in which no such point
// was dropped; the time of a dropped point is lost.
//
// After the features come bbox, [minLon, minLat, maxLon, maxLat], and the
// foreign member wanderer: {"v": 1, "elevation": "file"} or
// {"v": 1, "elevation": "none"}. The elevation value is "none" exactly when
// Result.HasElevation is false. The reserved third value, "dem", belongs to a
// later phase and is never written here.
//
// This is a costly contract. A later phase stores these bytes and serves them,
// so every property name and marker value is part of the on-disk format, and
// wanderer.v exists so the format can evolve. Compare the output only after
// decompression: the deflate bytes may change between Go releases.
//
// Two read-path dependencies are recorded here for the phase that serves the
// file. First (Pitfall 12), the vendored elevation profile formats
// position[2].toFixed(2) for its tooltip, which throws on a two-element
// position, so the file is consumed unchanged only for tracks in which every
// point has an elevation until the read path handles the others. Second
// (T-46-21), the file is gzipped, so whoever reads it back must cap the
// gunzipped size against a decompression bomb.
//
// EncodeGeoJSON streams through a reused 32 KiB buffer into gzip and never
// holds the document. The caller owns w: the gzip stream is finished but w is
// never closed. The first write error is returned.
func (r *Result) EncodeGeoJSON(w io.Writer) error {
	g := &geoJSONWriter{gz: gzip.NewWriter(w), buf: make([]byte, 0, flushThreshold+4096)}

	g.str(`{"type":"FeatureCollection","features":[`)
	first := true
	for ti := range r.tracks {
		t := &r.tracks[ti]
		for si := range t.segments {
			if !first {
				g.str(",")
			}
			first = false
			g.writeFeature(t, &t.segments[si])
			g.maybeFlush()
			if g.err != nil {
				break
			}
		}
		if g.err != nil {
			break
		}
	}

	g.str(`],"bbox":[`)
	g.float(r.Bounds.MinLon)
	g.str(",")
	g.float(r.Bounds.MinLat)
	g.str(",")
	g.float(r.Bounds.MaxLon)
	g.str(",")
	g.float(r.Bounds.MaxLat)
	if r.HasElevation {
		g.str(`],"wanderer":{"v":1,"elevation":"file"}}`)
	} else {
		g.str(`],"wanderer":{"v":1,"elevation":"none"}}`)
	}
	g.flush()

	// Close finishes the gzip stream. It does not close w.
	if err := g.gz.Close(); g.err == nil {
		g.err = err
	}
	return g.err
}

// writeFeature writes one LineString Feature for a stored segment.
func (g *geoJSONWriter) writeFeature(t *measuredTrack, s *measuredSegment) {
	g.str(`{"type":"Feature","geometry":{"type":"LineString","coordinates":[`)
	for i := range s.lat {
		if i > 0 {
			g.str(",")
		}
		g.str("[")
		g.float(s.lon[i])
		g.str(",")
		g.float(s.lat[i])
		if e := s.ele[i]; e == e { // not NaN: the point has an elevation
			g.str(",")
			g.float(e)
		}
		g.str("]")
		g.maybeFlush()
	}
	g.str(`]},"properties":{`)

	g.stringProperty("name", t.name)
	g.stringProperty("desc", t.desc)
	g.stringProperty("type", t.typ)
	if t.hasNumber {
		g.str(`"number":`)
		g.float(t.number)
		g.str(",")
	}
	id := strconv.Itoa(s.fileIndex)
	g.str(`"featureId":` + id + `,"segmentId":` + id + `,`)

	g.str(`"coordinateProperties":{"times":[`)
	for i, ms := range s.timeMs {
		if i > 0 {
			g.str(",")
		}
		if ms == noTime {
			g.str("null")
		} else {
			g.buf = append(g.buf, '"')
			g.buf = appendISOMillis(g.buf, ms)
			g.buf = append(g.buf, '"')
		}
		g.maybeFlush()
	}
	g.str(`]},"anchors":[`)
	n := 0
	for _, a := range s.anchors {
		if a < 1 || a > len(s.lat)-2 {
			continue
		}
		if n > 0 {
			g.str(",")
		}
		g.buf = strconv.AppendInt(g.buf, int64(a), 10)
		n++
	}
	g.str(`]}}`)
}
