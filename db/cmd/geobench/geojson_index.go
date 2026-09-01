package main

import (
	"context"
	"fmt"
	"math"
)

const defaultGeoJSONMaxSegmentLengthMeters = 5_000

const maxGeoJSONSubsegmentsPerSourceSegment = 1_000_000

// normalizeGeoJSONTrailsContext builds only the geometry written to _geojson.
// The product polyline and every oracle continue to use the original trails.
// RDP runs first, then each remaining spherical segment is densified before
// RFC 7946 antimeridian cutting. Original MultiLineString part boundaries are
// never joined.
func normalizeGeoJSONTrailsContext(
	ctx context.Context,
	trails []Trail,
	toleranceMeters float64,
	maxSegmentLengthMeters float64,
) ([]Trail, GeoJSONIndexReport, error) {
	simplified, report, err := simplifyGeoJSONTrailsContext(ctx, trails, toleranceMeters)
	report.MaxSegmentLengthMeters = maxSegmentLengthMeters
	report.PreNormalizationVertices = report.IndexedVertices
	report.PreNormalizationSegments = report.IndexedSegments
	if err != nil {
		return nil, report, err
	}
	if math.IsNaN(maxSegmentLengthMeters) || math.IsInf(maxSegmentLengthMeters, 0) || maxSegmentLengthMeters <= 0 {
		return nil, report, fmt.Errorf("GeoJSON maximum segment length must be a finite positive value: %g", maxSegmentLengthMeters)
	}

	indexed := make([]Trail, len(simplified))
	report.IndexedVertices = 0
	report.IndexedSegments = 0
	for trailIndex, trail := range simplified {
		if err := ctx.Err(); err != nil {
			return nil, report, err
		}
		indexed[trailIndex] = Trail{
			ID:       trail.ID,
			Scenario: trail.Scenario,
			Parts:    make([][]Coordinate, 0, len(trail.Parts)),
		}
		for _, part := range trail.Parts {
			densified, inserted, err := densifyGeoJSONPartContext(ctx, part, maxSegmentLengthMeters)
			if err != nil {
				return nil, report, err
			}
			report.DensifiedVertices += inserted

			cutParts, cuts, err := splitGeoJSONPartAtAntimeridianContext(ctx, densified)
			if err != nil {
				return nil, report, err
			}
			report.AntimeridianCuts += cuts
			indexed[trailIndex].Parts = append(indexed[trailIndex].Parts, cutParts...)
			for _, indexedPart := range cutParts {
				report.IndexedVertices += len(indexedPart)
				report.IndexedSegments += segmentCount(indexedPart)
			}
		}
	}
	report.PostNormalizationVertices = report.IndexedVertices
	report.PostNormalizationSegments = report.IndexedSegments
	return indexed, report, nil
}

// densifyGeoJSONPartContext approximates the benchmark's spherical
// point-to-polyline contract with short GeoJSON Cartesian segments. Original
// endpoints are retained; only the indexed projection receives new vertices.
func densifyGeoJSONPartContext(
	ctx context.Context,
	part []Coordinate,
	maxSegmentLengthMeters float64,
) ([]Coordinate, int, error) {
	if math.IsNaN(maxSegmentLengthMeters) || math.IsInf(maxSegmentLengthMeters, 0) || maxSegmentLengthMeters <= 0 {
		return nil, 0, fmt.Errorf("GeoJSON maximum segment length must be a finite positive value: %g", maxSegmentLengthMeters)
	}
	if len(part) < 2 {
		cloned, err := cloneGeoJSONPartContext(ctx, part)
		return cloned, 0, err
	}

	result := make([]Coordinate, 0, len(part))
	result = append(result, part[0])
	inserted := 0
	checks := 0
	for index := 1; index < len(part); index++ {
		if checks%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, inserted, err
			}
		}
		start := part[index-1]
		end := part[index]
		distanceMeters := angularDistance(start, end) * earthRadiusMeters
		subsegmentEstimate := math.Ceil(distanceMeters / maxSegmentLengthMeters)
		if math.IsInf(subsegmentEstimate, 0) || subsegmentEstimate > maxGeoJSONSubsegmentsPerSourceSegment {
			return nil, inserted, fmt.Errorf(
				"GeoJSON segment requires %.0f subdivisions; maximum is %d (increase --geojson-max-segment-meters)",
				subsegmentEstimate,
				maxGeoJSONSubsegmentsPerSourceSegment,
			)
		}
		subsegments := max(1, int(subsegmentEstimate))
		for step := 1; step <= subsegments; step++ {
			if checks%256 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, inserted, err
				}
			}
			if step == subsegments {
				result = append(result, end)
			} else {
				result = append(result, interpolateGreatCircle(start, end, float64(step)/float64(subsegments)))
				inserted++
			}
			checks++
		}
	}
	return result, inserted, nil
}

// splitGeoJSONPartAtAntimeridianContext follows RFC 7946 section 3.1.9: a
// route crossing the antimeridian becomes multiple LineStrings whose seam
// endpoints are represented on their respective sides (+180/-180). A split is
// applied to one source part at a time, so a real MultiLineString gap cannot
// become a route segment.
func splitGeoJSONPartAtAntimeridianContext(
	ctx context.Context,
	part []Coordinate,
) ([][]Coordinate, int, error) {
	if len(part) == 0 {
		cloned, err := cloneGeoJSONPartContext(ctx, part)
		return [][]Coordinate{cloned}, 0, err
	}

	first := part[0]
	first.Lon = normalizeLongitude(first.Lon)
	current := []Coordinate{first}
	result := make([][]Coordinate, 0, 1)
	cuts := 0
	previous := first
	for index := 1; index < len(part); index++ {
		if index%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, cuts, err
			}
		}
		next := part[index]
		next.Lon = normalizeLongitude(next.Lon)
		delta := next.Lon - previous.Lon
		if math.Abs(delta) <= 180 {
			current = append(current, next)
			previous = next
			continue
		}

		oldBoundary := 180.0
		newBoundary := -180.0
		unwrappedEnd := next.Lon + 360
		if delta > 180 {
			oldBoundary = -180
			newBoundary = 180
			unwrappedEnd = next.Lon - 360
		}
		fraction := antimeridianCrossingFraction(previous, next, unwrappedEnd, oldBoundary)
		seam := interpolateGreatCircle(previous, next, fraction)
		oldSeam := Coordinate{Lat: seam.Lat, Lon: oldBoundary}
		newSeam := Coordinate{Lat: seam.Lat, Lon: newBoundary}

		const endpointEpsilon = 1e-12
		switch {
		case fraction <= endpointEpsilon:
			// Crossing at a vertex. If it is the first vertex there is no
			// preceding LineString to preserve; otherwise finish that line and
			// start its counterpart on the other side of the antimeridian.
			if len(current) >= 2 {
				current[len(current)-1] = oldSeam
				result = append(result, current)
				current = []Coordinate{newSeam, next}
				cuts++
			} else {
				current[0] = newSeam
				current = append(current, next)
			}
		case fraction >= 1-endpointEpsilon:
			// Keep the endpoint on the current side. If the following segment
			// continues across the seam, its fraction-zero case performs the
			// actual part split without creating a singleton LineString.
			current = append(current, oldSeam)
			previous = oldSeam
			continue
		default:
			current = append(current, oldSeam)
			result = append(result, current)
			current = []Coordinate{newSeam, next}
			cuts++
		}
		previous = next
	}
	result = append(result, current)
	return result, cuts, nil
}

func antimeridianCrossingFraction(start, end Coordinate, unwrappedEnd, boundary float64) float64 {
	if unwrappedEnd == start.Lon {
		return 0
	}
	ascending := unwrappedEnd > start.Lon
	low, high := 0.0, 1.0
	for range 52 {
		middle := (low + high) / 2
		coordinate := interpolateGreatCircle(start, end, middle)
		longitude := unwrapLongitudeNear(coordinate.Lon, start.Lon)
		if (ascending && longitude < boundary) || (!ascending && longitude > boundary) {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / 2
}

func unwrapLongitudeNear(longitude, reference float64) float64 {
	for longitude-reference > 180 {
		longitude -= 360
	}
	for longitude-reference < -180 {
		longitude += 360
	}
	return longitude
}

// simplifyGeoJSONTrailsContext prepares the geometry that is written to the
// GeoJSON index. Each part is simplified independently so gaps between track
// segments can never turn into artificial route segments.
func simplifyGeoJSONTrailsContext(
	ctx context.Context,
	trails []Trail,
	toleranceMeters float64,
) ([]Trail, GeoJSONIndexReport, error) {
	report := GeoJSONIndexReport{SimplifyToleranceMeters: toleranceMeters}
	if toleranceMeters < 0 {
		return nil, report, fmt.Errorf("GeoJSON simplification tolerance must not be negative: %g", toleranceMeters)
	}
	if err := ctx.Err(); err != nil {
		return nil, report, err
	}

	simplified := make([]Trail, len(trails))
	for trailIndex, trail := range trails {
		if err := ctx.Err(); err != nil {
			return nil, report, err
		}
		simplified[trailIndex] = Trail{
			ID:       trail.ID,
			Scenario: trail.Scenario,
			Parts:    make([][]Coordinate, len(trail.Parts)),
		}
		for partIndex, part := range trail.Parts {
			if err := ctx.Err(); err != nil {
				return nil, report, err
			}
			report.SourceVertices += len(part)
			report.SourceSegments += segmentCount(part)

			var indexedPart []Coordinate
			var err error
			if toleranceMeters == 0 {
				indexedPart, err = cloneGeoJSONPartContext(ctx, part)
			} else {
				indexedPart, err = simplifyGeoJSONPartContext(ctx, part, toleranceMeters)
			}
			if err != nil {
				return nil, report, err
			}
			simplified[trailIndex].Parts[partIndex] = indexedPart
			report.IndexedVertices += len(indexedPart)
			report.IndexedSegments += segmentCount(indexedPart)
		}
	}

	if report.SourceVertices > 0 {
		report.VertexReductionPercent = 100 * float64(report.SourceVertices-report.IndexedVertices) /
			float64(report.SourceVertices)
	}
	return simplified, report, nil
}

type geoJSONSimplifyRange struct {
	first int
	last  int
}

// simplifyGeoJSONPartContext applies Ramer-Douglas-Peucker using the same
// spherical point-to-segment distance as the benchmark's exact oracle. The
// iterative work stack avoids recursion depth depending on input geometry.
func simplifyGeoJSONPartContext(
	ctx context.Context,
	part []Coordinate,
	toleranceMeters float64,
) ([]Coordinate, error) {
	if len(part) <= 2 {
		return cloneGeoJSONPartContext(ctx, part)
	}

	keep := make([]bool, len(part))
	keep[0] = true
	keep[len(part)-1] = true
	stack := []geoJSONSimplifyRange{{first: 0, last: len(part) - 1}}
	distanceChecks := 0

	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lastRange := len(stack) - 1
		current := stack[lastRange]
		stack = stack[:lastRange]

		farthestIndex := -1
		farthestDistance := -1.0
		for index := current.first + 1; index < current.last; index++ {
			if distanceChecks%256 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			distance := pointToSegmentDistanceMeters(
				part[index],
				part[current.first],
				part[current.last],
			)
			distanceChecks++
			if distance > farthestDistance {
				farthestDistance = distance
				farthestIndex = index
			}
		}

		if farthestIndex >= 0 && farthestDistance > toleranceMeters {
			keep[farthestIndex] = true
			stack = append(stack,
				geoJSONSimplifyRange{first: farthestIndex, last: current.last},
				geoJSONSimplifyRange{first: current.first, last: farthestIndex},
			)
		}
	}

	result := make([]Coordinate, 0, len(part))
	for index, coordinate := range part {
		if index%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if keep[index] {
			result = append(result, coordinate)
		}
	}
	return result, nil
}

func cloneGeoJSONPartContext(ctx context.Context, part []Coordinate) ([]Coordinate, error) {
	if part == nil {
		return nil, nil
	}
	cloned := make([]Coordinate, len(part))
	for index, coordinate := range part {
		if index%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		cloned[index] = coordinate
	}
	return cloned, nil
}

func cloneGeoJSONPart(part []Coordinate) []Coordinate {
	if part == nil {
		return nil
	}
	cloned := make([]Coordinate, len(part))
	copy(cloned, part)
	return cloned
}

func segmentCount(part []Coordinate) int {
	if len(part) < 2 {
		return 0
	}
	return len(part) - 1
}
