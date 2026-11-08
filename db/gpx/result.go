package gpx

import "time"

// firstPoint is the first stored point of the measured source in file order.
func firstPoint(tracks []measuredTrack) (LatLon, bool) {
	for ti := range tracks {
		if len(tracks[ti].segments) > 0 {
			seg := &tracks[ti].segments[0]
			return LatLon{Lat: seg.lat[0], Lon: seg.lon[0]}, true
		}
	}
	return LatLon{}, false
}

// firstNonEmpty mirrors the TypeScript `||` chain: an empty string falls
// through to the next candidate.
func firstNonEmpty(candidates ...string) string {
	for _, c := range candidates {
		if c != "" {
			return c
		}
	}
	return ""
}

// deriveTrail fills the trail fields of r that come from the document's text
// and from the first point, mirroring gpx2trail in web/src/lib/util/gpx_util.ts.
// The measurement fields are already set.
//
// The final name fallback is the empty string: a pure package never invents a
// timestamp name, so the caller decides what an unnamed trail is called.
func (b *builder) deriveTrail(r *Result, fallbackName string) {
	r.Name = firstNonEmpty(b.metaText.name, b.firstTrkName, b.firstRteName, fallbackName)
	r.Description = b.metaText.desc

	// A4: the first stored point of the measured source, in file order.
	r.Start, _ = firstPoint(r.tracks)

	// The date needs the first segment's first and last time. firstTimes was
	// recorded for the first <trkseg> of the first <trk>, or for the first <rte>.
	if r.firstTimes.start != noTime && r.firstTimes.end != noTime {
		r.StartDate = time.UnixMilli(r.firstTimes.start).UTC().Format("2006-01-02")
	}

	for _, w := range b.waypoints {
		r.Waypoints = append(r.Waypoints, Waypoint{
			Lat:         w.lat,
			Lon:         w.lon,
			Name:        w.text.name,
			Description: w.text.desc,
			Icon:        iconFor(w.text.sym),
		})
	}
}
