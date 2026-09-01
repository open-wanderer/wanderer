package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	h3 "github.com/uber/h3-go/v4"
)

const (
	// Sampling at half an average edge length keeps successive cells close
	// enough for GridPath while limiting divergence between the grid path and
	// the geographic segment it represents.
	h3DensifyEdgeFraction = 0.5
	h3GridSafetyRings     = 2
	h3MaxSegmentSamples   = 1_000_000
	h3MaxCellsPerTrail    = 500_000
	h3MaxQueryCells       = 100_000
	h3MaxSubdivisionDepth = 16
)

// h3FieldsForTrail rasterizes every part of a trail independently at every
// requested resolution. A new rasterization starts for each part so the gap
// between two LineStrings in a MultiLineString is never indexed as a segment.
func h3FieldsForTrail(trail Trail, resolutions []int) (map[string][]string, error) {
	return h3FieldsForTrailContext(context.Background(), trail, resolutions)
}

func h3FieldsForTrailContext(ctx context.Context, trail Trail, resolutions []int) (map[string][]string, error) {
	resolutions, err := normalizedH3Resolutions(resolutions)
	if err != nil {
		return nil, err
	}

	for partIndex, part := range trail.Parts {
		for coordinateIndex, coordinate := range part {
			if err := validateH3Coordinate(coordinate); err != nil {
				return nil, fmt.Errorf("trail %q part %d coordinate %d: %w", trail.ID, partIndex, coordinateIndex, err)
			}
		}
	}

	fields := make(map[string][]string, len(resolutions))
	for _, resolution := range resolutions {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cells, err := h3CellsForTrailAtResolution(ctx, trail, resolution)
		if err != nil {
			return nil, fmt.Errorf("trail %q at H3 resolution %d: %w", trail.ID, resolution, err)
		}

		values := make([]string, 0, len(cells))
		for cell := range cells {
			values = append(values, cell.String())
		}
		sort.Strings(values)
		fields[h3FieldName(resolution)] = values
	}

	return fields, nil
}

func h3CellsForTrailAtResolution(ctx context.Context, trail Trail, resolution int) (map[h3.Cell]struct{}, error) {
	edgeLength, err := h3.HexagonEdgeLengthAvgM(resolution)
	if err != nil {
		return nil, fmt.Errorf("get average edge length: %w", err)
	}
	if edgeLength <= 0 || math.IsNaN(edgeLength) || math.IsInf(edgeLength, 0) {
		return nil, fmt.Errorf("invalid average edge length %v", edgeLength)
	}

	maxStepMeters := edgeLength * h3DensifyEdgeFraction
	cells := make(map[h3.Cell]struct{})
	for partIndex, part := range trail.Parts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(part) == 0 {
			continue
		}

		firstCell, err := coordinateToH3Cell(part[0], resolution)
		if err != nil {
			return nil, fmt.Errorf("part %d first coordinate: %w", partIndex, err)
		}
		cells[firstCell] = struct{}{}

		for segmentIndex := 1; segmentIndex < len(part); segmentIndex++ {
			if err := rasterizeH3Segment(
				ctx,
				part[segmentIndex-1],
				part[segmentIndex],
				resolution,
				maxStepMeters,
				cells,
			); err != nil {
				return nil, fmt.Errorf("part %d segment %d: %w", partIndex, segmentIndex-1, err)
			}
			if len(cells) > h3MaxCellsPerTrail {
				return nil, fmt.Errorf("trail exceeds %d cells", h3MaxCellsPerTrail)
			}
		}
	}

	return cells, nil
}

func rasterizeH3Segment(
	ctx context.Context,
	start Coordinate,
	end Coordinate,
	resolution int,
	maxStepMeters float64,
	cells map[h3.Cell]struct{},
) error {
	distance := h3.GreatCircleDistanceM(toH3LatLng(start), toH3LatLng(end))
	if math.IsNaN(distance) || math.IsInf(distance, 0) || distance < 0 {
		return fmt.Errorf("invalid segment distance %v", distance)
	}

	steps := int(math.Ceil(distance / maxStepMeters))
	if steps < 1 {
		steps = 1
	}
	if steps > h3MaxSegmentSamples {
		return fmt.Errorf("segment needs %d samples, maximum is %d", steps, h3MaxSegmentSamples)
	}
	if steps > h3MaxCellsPerTrail-len(cells) {
		return fmt.Errorf("segment may exceed the %d-cell trail budget", h3MaxCellsPerTrail)
	}

	previousCoordinate := start
	previousCell, err := coordinateToH3Cell(start, resolution)
	if err != nil {
		return err
	}
	cells[previousCell] = struct{}{}

	for step := 1; step <= steps; step++ {
		if step%256 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		fraction := float64(step) / float64(steps)
		currentCoordinate := interpolateGreatCircle(start, end, fraction)
		currentCell, err := coordinateToH3Cell(currentCoordinate, resolution)
		if err != nil {
			return err
		}

		if err := addH3GridSpan(
			ctx,
			previousCoordinate,
			currentCoordinate,
			previousCell,
			currentCell,
			resolution,
			cells,
			0,
		); err != nil {
			return err
		}

		previousCoordinate = currentCoordinate
		previousCell = currentCell
	}

	return nil
}

// addH3GridSpan uses GridPath for the ordinary case. GridPath can fail around
// pentagons or for cells that are too far apart, so a failing span is split at
// its geographic midpoint until local GridPath calls succeed.
func addH3GridSpan(
	ctx context.Context,
	start Coordinate,
	end Coordinate,
	startCell h3.Cell,
	endCell h3.Cell,
	resolution int,
	cells map[h3.Cell]struct{},
	depth int,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cells[startCell] = struct{}{}
	cells[endCell] = struct{}{}
	if startCell == endCell {
		return nil
	}

	path, err := h3.GridPath(startCell, endCell)
	if err == nil {
		for _, cell := range path {
			if cell.IsValid() {
				cells[cell] = struct{}{}
			}
		}
		return nil
	}

	if depth >= h3MaxSubdivisionDepth {
		return fmt.Errorf("grid path from %s to %s: %w", startCell, endCell, err)
	}

	midpoint := interpolateGreatCircle(start, end, 0.5)
	midpointCell, midpointErr := coordinateToH3Cell(midpoint, resolution)
	if midpointErr != nil {
		return midpointErr
	}
	if midpointCell == startCell || midpointCell == endCell {
		// The geographic span is already local. Retaining both endpoint cells is
		// conservative and avoids dropping a segment solely due to pentagon
		// distortion in GridPath.
		return nil
	}

	if err := addH3GridSpan(ctx, start, midpoint, startCell, midpointCell, resolution, cells, depth+1); err != nil {
		return err
	}
	return addH3GridSpan(ctx, midpoint, end, midpointCell, endCell, resolution, cells, depth+1)
}

// h3QueryFilter chooses a radius-appropriate indexed resolution and returns a
// Meilisearch IN filter containing a conservative H3 disk around the query.
func h3QueryFilter(
	point Coordinate,
	radiusM float64,
	resolutions []int,
) (field string, filter string, cells int, err error) {
	if err := validateH3Coordinate(point); err != nil {
		return "", "", 0, fmt.Errorf("query coordinate: %w", err)
	}
	if radiusM < 0 || math.IsNaN(radiusM) || math.IsInf(radiusM, 0) {
		return "", "", 0, fmt.Errorf("radius must be a finite non-negative number, got %v", radiusM)
	}

	resolutions, err = normalizedH3Resolutions(resolutions)
	if err != nil {
		return "", "", 0, err
	}
	resolution := h3ResolutionForRadius(radiusM, resolutions)
	field = h3FieldName(resolution)

	origin, err := coordinateToH3Cell(point, resolution)
	if err != nil {
		return "", "", 0, fmt.Errorf("query origin: %w", err)
	}
	edgeLength, err := h3.HexagonEdgeLengthAvgM(resolution)
	if err != nil {
		return "", "", 0, fmt.Errorf("get average edge length for resolution %d: %w", resolution, err)
	}
	if edgeLength <= 0 || math.IsNaN(edgeLength) || math.IsInf(edgeLength, 0) {
		return "", "", 0, fmt.Errorf("invalid average edge length %v", edgeLength)
	}

	// Dividing by one edge rather than the larger center-to-center spacing
	// intentionally overestimates the required grid distance. Two extra rings
	// cover the query cell and route-cell boundary effects.
	ringsFloat := math.Ceil(radiusM/edgeLength) + h3GridSafetyRings
	if ringsFloat > float64(math.MaxInt) {
		return "", "", 0, fmt.Errorf("radius %.0f produces an unsupported grid distance", radiusM)
	}
	rings := int(ringsFloat)
	estimatedCells := 3*float64(rings)*(float64(rings)+1) + 1
	if estimatedCells > h3MaxQueryCells {
		return "", "", 0, fmt.Errorf(
			"H3 query at resolution %d needs about %.0f cells, maximum is %d",
			resolution,
			estimatedCells,
			h3MaxQueryCells,
		)
	}
	disk, err := h3.GridDisk(origin, rings)
	if err != nil {
		return "", "", 0, fmt.Errorf("H3 grid disk at resolution %d with %d rings: %w", resolution, rings, err)
	}

	unique := make(map[string]struct{}, len(disk))
	for _, cell := range disk {
		if cell.IsValid() {
			unique[cell.String()] = struct{}{}
		}
	}
	values := make([]string, 0, len(unique))
	for value := range unique {
		values = append(values, value)
	}
	sort.Strings(values)

	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = strconv.Quote(value)
	}
	filter = fmt.Sprintf("%s IN [%s]", field, strings.Join(quoted, ", "))
	return field, filter, len(values), nil
}

func h3ResolutionForRadius(radiusM float64, resolutions []int) int {
	last := len(resolutions) - 1
	switch {
	case radiusM <= 500:
		return resolutions[last]
	case radiusM <= 5_000:
		return resolutions[max(last-1, 0)]
	case radiusM <= 25_000:
		return resolutions[max(last-2, 0)]
	default:
		return resolutions[0]
	}
}

func normalizedH3Resolutions(resolutions []int) ([]int, error) {
	if len(resolutions) == 0 {
		return nil, fmt.Errorf("at least one H3 resolution is required")
	}

	unique := make(map[int]struct{}, len(resolutions))
	for _, resolution := range resolutions {
		if resolution < 0 || resolution > h3.MaxResolution {
			return nil, fmt.Errorf("H3 resolution %d is outside 0..%d", resolution, h3.MaxResolution)
		}
		unique[resolution] = struct{}{}
	}

	normalized := make([]int, 0, len(unique))
	for resolution := range unique {
		normalized = append(normalized, resolution)
	}
	sort.Ints(normalized)
	return normalized, nil
}

func h3FieldName(resolution int) string {
	return fmt.Sprintf("h3_r%d", resolution)
}

func coordinateToH3Cell(coordinate Coordinate, resolution int) (h3.Cell, error) {
	cell, err := h3.LatLngToCell(toH3LatLng(coordinate), resolution)
	if err != nil {
		return h3.Cell(0), err
	}
	if !cell.IsValid() {
		return h3.Cell(0), fmt.Errorf("H3 returned invalid cell for (%v, %v)", coordinate.Lat, coordinate.Lon)
	}
	return cell, nil
}

func toH3LatLng(coordinate Coordinate) h3.LatLng {
	return h3.NewLatLng(coordinate.Lat, coordinate.Lon)
}

func validateH3Coordinate(coordinate Coordinate) error {
	if math.IsNaN(coordinate.Lat) || math.IsInf(coordinate.Lat, 0) ||
		math.IsNaN(coordinate.Lon) || math.IsInf(coordinate.Lon, 0) {
		return fmt.Errorf("latitude and longitude must be finite")
	}
	if coordinate.Lat < -90 || coordinate.Lat > 90 {
		return fmt.Errorf("latitude %v is outside -90..90", coordinate.Lat)
	}
	if coordinate.Lon < -180 || coordinate.Lon > 180 {
		return fmt.Errorf("longitude %v is outside -180..180", coordinate.Lon)
	}
	return nil
}

func interpolateGreatCircle(start, end Coordinate, fraction float64) Coordinate {
	if fraction <= 0 {
		return start
	}
	if fraction >= 1 {
		return end
	}

	lat1 := start.Lat * math.Pi / 180
	lon1 := start.Lon * math.Pi / 180
	lat2 := end.Lat * math.Pi / 180
	lon2 := end.Lon * math.Pi / 180

	dot := math.Sin(lat1)*math.Sin(lat2) + math.Cos(lat1)*math.Cos(lat2)*math.Cos(lon2-lon1)
	dot = math.Max(-1, math.Min(1, dot))
	angle := math.Acos(dot)
	sinAngle := math.Sin(angle)
	if math.Abs(sinAngle) < 1e-12 {
		// The great-circle arc is undefined for exact antipodes. Trail segments
		// are normally far shorter; shortest wrapped linear interpolation is a
		// deterministic and finite fallback for this degenerate case.
		longitudeDelta := math.Mod(end.Lon-start.Lon+540, 360) - 180
		return Coordinate{
			Lat: start.Lat + fraction*(end.Lat-start.Lat),
			Lon: normalizeLongitude(start.Lon + fraction*longitudeDelta),
		}
	}

	startWeight := math.Sin((1-fraction)*angle) / sinAngle
	endWeight := math.Sin(fraction*angle) / sinAngle
	x := startWeight*math.Cos(lat1)*math.Cos(lon1) + endWeight*math.Cos(lat2)*math.Cos(lon2)
	y := startWeight*math.Cos(lat1)*math.Sin(lon1) + endWeight*math.Cos(lat2)*math.Sin(lon2)
	z := startWeight*math.Sin(lat1) + endWeight*math.Sin(lat2)

	return Coordinate{
		Lat: math.Atan2(z, math.Hypot(x, y)) * 180 / math.Pi,
		Lon: normalizeLongitude(math.Atan2(y, x) * 180 / math.Pi),
	}
}
