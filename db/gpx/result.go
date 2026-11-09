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

// hasElevation is the PARSE-06 rule. The population is every stored point of
// the measured source (the same set as PointCount); a usable elevation is a
// non-NaN ele, which already passed parseNumber, so a literal 0 counts. The
// flag is false when fewer than half of the points carry one (exactly half
// passes) or when every usable elevation is identical, which includes a single
// usable value. It only drives the GeoJSON marker: gain and loss are never
// altered by it.
func hasElevation(tracks []measuredTrack) bool {
	population, usable := 0, 0
	var lo, hi float64
	for ti := range tracks {
		for si := range tracks[ti].segments {
			for _, ele := range tracks[ti].segments[si].ele {
				population++
				if ele != ele { // NaN: no usable elevation
					continue
				}
				if usable == 0 || ele < lo {
					lo = ele
				}
				if usable == 0 || ele > hi {
					hi = ele
				}
				usable++
			}
		}
	}
	if usable*2 < population {
		return false
	}
	return usable > 0 && lo != hi
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
	r.HasElevation = hasElevation(r.tracks)

	// A4: the first stored point of the measured source, in file order.
	r.Start, _ = firstPoint(r.tracks)

	// The date needs the first segment's first and last time. firstTimes was
	// recorded for the first <trkseg> of the first <trk>, or for the first <rte>.
	//
	// StartDate is a four-digit-year date or nothing: the corpus and the stored
	// date field take YYYY-MM-DD or null, and TS's toISOString().substring(0, 10)
	// of an expanded-year string such as "+012345-01-01" is not a date (WR-01).
	if r.firstTimes.start != noTime && r.firstTimes.end != noTime {
		if t := time.UnixMilli(r.firstTimes.start).UTC(); t.Year() >= 0 && t.Year() <= 9999 {
			r.StartDate = t.Format("2006-01-02")
		}
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
