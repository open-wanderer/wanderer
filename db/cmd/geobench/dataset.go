package main

import (
	"context"
	"fmt"
	"math"
	"math/rand"
)

const (
	scenarioAlpineDense = "alpine_dense"
	scenarioBackground  = "background"
	scenarioLongOutlier = "long_outlier"

	angularEpsilon = 1e-14
)

var datasetScenarios = [...]string{
	scenarioAlpineDense,
	scenarioBackground,
	scenarioLongOutlier,
}

// generateDataset creates a deterministic mixture of densely overlapping
// Alpine trails, geographically dispersed background trails, and a small set
// of very long trails. At least three trails and query points are required so
// every scenario is represented.
func generateDataset(trails, queries int, seed int64) (Dataset, error) {
	return generateDatasetContext(context.Background(), trails, queries, seed)
}

func generateDatasetContext(ctx context.Context, trails, queries int, seed int64) (Dataset, error) {
	if trails < len(datasetScenarios) {
		return Dataset{}, fmt.Errorf("trail count must be at least %d", len(datasetScenarios))
	}
	if queries < len(datasetScenarios) {
		return Dataset{}, fmt.Errorf("query count must be at least %d", len(datasetScenarios))
	}

	rng := rand.New(rand.NewSource(seed))
	// Long routes are deliberately rare outliers. Keeping them at roughly
	// 0.2% still stresses each strategy without making a fine-resolution H3 run
	// consist mostly of millions of cells from synthetic transcontinental data.
	trailCounts := allocateScenarioCounts(trails, [3]float64{0.68, 0.318, 0.002})
	queryCounts := allocateScenarioCounts(queries, [3]float64{0.50, 0.25, 0.25})
	multiLineOffset := rng.Intn(10)

	dataset := Dataset{
		Seed:        seed,
		Trails:      make([]Trail, 0, trails),
		QueryPoints: make([]QueryPoint, 0, queries),
	}
	byScenario := make(map[string][]Trail, len(datasetScenarios))

	for scenarioIndex, scenario := range datasetScenarios {
		for i := 0; i < trailCounts[scenarioIndex]; i++ {
			if err := ctx.Err(); err != nil {
				return Dataset{}, err
			}
			globalIndex := len(dataset.Trails)
			multiLine := (globalIndex+multiLineOffset)%10 == 0
			var parts [][]Coordinate
			switch scenario {
			case scenarioAlpineDense:
				parts = generateAlpineTrail(rng, i, multiLine)
			case scenarioBackground:
				parts = generateBackgroundTrail(rng, i, multiLine)
			case scenarioLongOutlier:
				parts = generateLongOutlierTrail(rng, i, multiLine)
			default:
				return Dataset{}, fmt.Errorf("unknown scenario %q", scenario)
			}

			trail := Trail{
				ID:       fmt.Sprintf("trail-%06d", globalIndex),
				Scenario: scenario,
				Parts:    parts,
			}
			dataset.Trails = append(dataset.Trails, trail)
			byScenario[scenario] = append(byScenario[scenario], trail)
		}
	}

	queryIndex := 0
	for scenarioIndex, scenario := range datasetScenarios {
		candidates := byScenario[scenario]
		if len(candidates) == 0 {
			return Dataset{}, fmt.Errorf("scenario %q has no trails", scenario)
		}
		for i := 0; i < queryCounts[scenarioIndex]; i++ {
			if err := ctx.Err(); err != nil {
				return Dataset{}, err
			}
			trail := candidates[rng.Intn(len(candidates))]
			anchor, err := queryAnchor(rng, trail, scenario == scenarioLongOutlier)
			if err != nil {
				return Dataset{}, err
			}

			offset := queryOffsetMeters(queryIndex, rng)
			point := anchor
			if offset != 0 {
				point = destinationPoint(anchor, offset, rng.Float64()*360)
			}
			dataset.QueryPoints = append(dataset.QueryPoints, QueryPoint{
				ID:       fmt.Sprintf("query-%06d", queryIndex),
				Scenario: scenario,
				Point:    point,
			})
			queryIndex++
		}
	}

	return dataset, nil
}

// allocateScenarioCounts applies the supplied weights while reserving one item
// for every scenario. The final bucket absorbs rounding so the sum is exact.
func allocateScenarioCounts(total int, weights [3]float64) [3]int {
	counts := [3]int{1, 1, 1}
	remaining := total - len(counts)
	if remaining <= 0 {
		return counts
	}

	counts[0] += int(math.Floor(float64(remaining) * weights[0]))
	counts[1] += int(math.Floor(float64(remaining) * weights[1]))
	counts[2] = total - counts[0] - counts[1]
	return counts
}

func generateAlpineTrail(rng *rand.Rand, index int, multiLine bool) [][]Coordinate {
	center := Coordinate{Lat: 46.82, Lon: 8.23}
	corridor := index % 6
	corridorBearing := []float64{24, 58, 91, 127, 166, 305}[corridor]
	corridorOrigin := destinationPoint(center, float64(corridor)*650, float64(corridor)*60)
	start := destinationPoint(corridorOrigin, math.Abs(rng.NormFloat64())*900, rng.Float64()*360)
	start = destinationPoint(start, rng.NormFloat64()*220, corridorBearing+90)

	vertices := 36 + rng.Intn(25)
	stepMeters := 230 + rng.Float64()*260
	path := make([]Coordinate, 0, vertices)
	current := start
	heading := corridorBearing + rng.NormFloat64()*4
	for i := 0; i < vertices; i++ {
		path = append(path, current)
		curve := 2.4 * math.Sin(float64(i)/7+float64(corridor))
		heading += curve*0.18 + rng.NormFloat64()*0.55
		current = destinationPoint(current, stepMeters*(0.88+rng.Float64()*0.24), heading)
	}
	return pathParts(path, multiLine)
}

func generateBackgroundTrail(rng *rand.Rand, index int, multiLine bool) [][]Coordinate {
	centers := [...]Coordinate{
		{Lat: 37.77, Lon: -122.42},
		{Lat: 40.71, Lon: -74.01},
		{Lat: 35.68, Lon: 139.69},
		{Lat: -33.87, Lon: 151.21},
		{Lat: 59.91, Lon: 10.75},
		{Lat: -22.91, Lon: -43.17},
		{Lat: 28.61, Lon: 77.21},
		{Lat: 0.35, Lon: 32.58},
	}
	center := centers[index%len(centers)]
	start := destinationPoint(center, rng.Float64()*180_000, rng.Float64()*360)
	vertices := 20 + rng.Intn(31)
	stepMeters := 650 + rng.Float64()*1_800
	path := make([]Coordinate, 0, vertices)
	current := start
	heading := rng.Float64() * 360
	for i := 0; i < vertices; i++ {
		path = append(path, current)
		heading += rng.NormFloat64()*2.8 + 0.35*math.Sin(float64(i)/4)
		current = destinationPoint(current, stepMeters*(0.75+rng.Float64()*0.5), heading)
	}
	return pathParts(path, multiLine)
}

func generateLongOutlierTrail(rng *rand.Rand, index int, multiLine bool) [][]Coordinate {
	starts := [...]Coordinate{
		{Lat: 46.8, Lon: 8.2},
		{Lat: 39.0, Lon: 164.0}, // deliberately crosses the antimeridian
		{Lat: -31.0, Lon: 119.0},
		{Lat: 52.0, Lon: -128.0},
	}
	bearings := [...]float64{86, 78, 102, 52}
	start := destinationPoint(starts[index%len(starts)], rng.Float64()*30_000, rng.Float64()*360)
	vertices := 105 + rng.Intn(56)
	stepMeters := 24_000 + rng.Float64()*18_000
	path := make([]Coordinate, 0, vertices)
	current := start
	heading := bearings[index%len(bearings)] + rng.NormFloat64()*2
	for i := 0; i < vertices; i++ {
		path = append(path, current)
		heading += rng.NormFloat64()*0.22 + 0.08*math.Sin(float64(i)/11)
		current = destinationPoint(current, stepMeters*(0.9+rng.Float64()*0.2), heading)
	}
	return pathParts(path, multiLine)
}

// pathParts drops a short section around the split. Keeping the two remaining
// sections as independent parts prevents a synthetic segment across the gap.
func pathParts(path []Coordinate, multiLine bool) [][]Coordinate {
	if !multiLine || len(path) < 8 {
		return [][]Coordinate{path}
	}

	gap := max(2, len(path)/12)
	leftEnd := len(path)/2 - gap/2
	rightStart := leftEnd + gap
	if leftEnd < 2 || len(path)-rightStart < 2 {
		return [][]Coordinate{path}
	}

	left := append([]Coordinate(nil), path[:leftEnd]...)
	right := append([]Coordinate(nil), path[rightStart:]...)
	return [][]Coordinate{left, right}
}

func queryAnchor(rng *rand.Rand, trail Trail, preferFarFromStart bool) (Coordinate, error) {
	points := flattenTrailPoints(trail)
	if len(points) == 0 {
		return Coordinate{}, fmt.Errorf("trail %q has no coordinates", trail.ID)
	}

	first := 0
	if preferFarFromStart && len(points) > 2 {
		first = (2 * len(points)) / 3
	}
	return points[first+rng.Intn(len(points)-first)], nil
}

func queryOffsetMeters(index int, rng *rand.Rand) float64 {
	// These bands exercise all requested radii and include exact on-trail points.
	bands := [...]float64{0, 150, 650, 2_500, 8_000, 35_000, 85_000, 140_000}
	base := bands[index%len(bands)]
	if base == 0 {
		return 0
	}
	return base * (0.85 + rng.Float64()*0.3)
}

func flattenTrailPoints(trail Trail) []Coordinate {
	count := 0
	for _, part := range trail.Parts {
		count += len(part)
	}
	points := make([]Coordinate, 0, count)
	for _, part := range trail.Parts {
		points = append(points, part...)
	}
	return points
}

func summarizeDataset(dataset Dataset) DatasetSummary {
	summary := DatasetSummary{
		Trails:      len(dataset.Trails),
		QueryPoints: len(dataset.QueryPoints),
		Scenarios: map[string]int{
			scenarioAlpineDense: 0,
			scenarioBackground:  0,
			scenarioLongOutlier: 0,
		},
	}

	for _, trail := range dataset.Trails {
		summary.Scenarios[trail.Scenario]++
		if len(trail.Parts) > 1 {
			summary.MultiLineTrails++
		}
		for _, part := range trail.Parts {
			summary.Vertices += len(part)
			if len(part) > 1 {
				summary.Segments += len(part) - 1
			}
		}
	}

	return summary
}

// mutateTrails returns up to n geometry-changing updates without modifying the
// input. Evenly spaced source trails include both dataset ends, so a small
// incremental batch covers the rare long-outlier scenario as well.
func mutateTrails(trails []Trail, n int) []Trail {
	if n <= 0 || len(trails) == 0 {
		return nil
	}
	if n > len(trails) {
		n = len(trails)
	}

	mutated := make([]Trail, 0, n)
	for i := 0; i < n; i++ {
		index := len(trails) - 1
		if n > 1 {
			index = i * (len(trails) - 1) / (n - 1)
		}
		source := trails[index]
		update := Trail{
			ID:       source.ID,
			Scenario: source.Scenario,
			Parts:    make([][]Coordinate, len(source.Parts)),
		}
		distance := 45 + float64(i%7)*17
		bearing := math.Mod(float64(i)*137.507764, 360)
		for partIndex, part := range source.Parts {
			update.Parts[partIndex] = make([]Coordinate, len(part))
			for pointIndex, point := range part {
				update.Parts[partIndex][pointIndex] = destinationPoint(point, distance, bearing)
			}
		}
		mutated = append(mutated, update)
	}
	return mutated
}

// pointToTrailDistanceMeters returns the shortest great-circle distance to any
// segment in any part. Parts are deliberately evaluated independently, so a
// MultiLineString gap never becomes an implicit segment.
func pointToTrailDistanceMeters(point Coordinate, trail Trail) float64 {
	distance, _ := pointToTrailDistanceMetersContext(context.Background(), point, trail)
	return distance
}

func pointToTrailDistanceMetersContext(ctx context.Context, point Coordinate, trail Trail) (float64, error) {
	minimum := math.Inf(1)
	processed := 0
	for _, part := range trail.Parts {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		switch len(part) {
		case 0:
			continue
		case 1:
			minimum = math.Min(minimum, angularDistance(point, part[0])*earthRadiusMeters)
			processed++
		default:
			for i := 1; i < len(part); i++ {
				if processed%256 == 0 {
					if err := ctx.Err(); err != nil {
						return 0, err
					}
				}
				minimum = math.Min(minimum, pointToSegmentDistanceMeters(point, part[i-1], part[i]))
				processed++
			}
		}
	}
	return minimum, nil
}

func pointToSegmentDistanceMeters(point, start, end Coordinate) float64 {
	p := coordinateVector(point)
	a := coordinateVector(start)
	b := coordinateVector(end)

	normalRaw := a.cross(b)
	normalLength := normalRaw.norm()
	arcLength := vectorAngle(a, b)
	if normalLength < angularEpsilon || arcLength < angularEpsilon {
		return math.Min(vectorAngle(p, a), vectorAngle(p, b)) * earthRadiusMeters
	}

	normal := normalRaw.scale(1 / normalLength)
	projectionRaw := p.subtract(normal.scale(p.dot(normal)))
	projectionLength := projectionRaw.norm()
	if projectionLength < angularEpsilon {
		// The point is a pole of this great circle. Every point on the great
		// circle is pi/2 away, so endpoint distance is a valid minimum too.
		return math.Min(vectorAngle(p, a), vectorAngle(p, b)) * earthRadiusMeters
	}
	projection := projectionRaw.scale(1 / projectionLength)

	startToProjection := vectorAngle(a, projection)
	projectionToEnd := vectorAngle(projection, b)
	if math.Abs((startToProjection+projectionToEnd)-arcLength) <= 1e-10 {
		return vectorAngle(p, projection) * earthRadiusMeters
	}

	return math.Min(vectorAngle(p, a), vectorAngle(p, b)) * earthRadiusMeters
}

// destinationPoint solves the direct geodesic problem on the benchmark's
// spherical Earth model. Bearing is expressed in degrees clockwise from north.
func destinationPoint(origin Coordinate, distanceMeters, bearingDegrees float64) Coordinate {
	angularDistance := distanceMeters / earthRadiusMeters
	bearing := degreesToRadians(bearingDegrees)
	lat1 := degreesToRadians(origin.Lat)
	lon1 := degreesToRadians(origin.Lon)

	sinLat1, cosLat1 := math.Sincos(lat1)
	sinDistance, cosDistance := math.Sincos(angularDistance)
	sinLat2 := sinLat1*cosDistance + cosLat1*sinDistance*math.Cos(bearing)
	lat2 := math.Asin(clamp(sinLat2, -1, 1))
	lon2 := lon1 + math.Atan2(
		math.Sin(bearing)*sinDistance*cosLat1,
		cosDistance-sinLat1*math.Sin(lat2),
	)

	return Coordinate{
		Lat: radiansToDegrees(lat2),
		Lon: normalizeLongitude(radiansToDegrees(lon2)),
	}
}

func angularDistance(a, b Coordinate) float64 {
	return vectorAngle(coordinateVector(a), coordinateVector(b))
}

type unitVector struct {
	x float64
	y float64
	z float64
}

func coordinateVector(coordinate Coordinate) unitVector {
	lat := degreesToRadians(coordinate.Lat)
	lon := degreesToRadians(coordinate.Lon)
	sinLat, cosLat := math.Sincos(lat)
	sinLon, cosLon := math.Sincos(lon)
	return unitVector{x: cosLat * cosLon, y: cosLat * sinLon, z: sinLat}
}

func (v unitVector) dot(other unitVector) float64 {
	return v.x*other.x + v.y*other.y + v.z*other.z
}

func (v unitVector) cross(other unitVector) unitVector {
	return unitVector{
		x: v.y*other.z - v.z*other.y,
		y: v.z*other.x - v.x*other.z,
		z: v.x*other.y - v.y*other.x,
	}
}

func (v unitVector) subtract(other unitVector) unitVector {
	return unitVector{x: v.x - other.x, y: v.y - other.y, z: v.z - other.z}
}

func (v unitVector) scale(factor float64) unitVector {
	return unitVector{x: v.x * factor, y: v.y * factor, z: v.z * factor}
}

func (v unitVector) norm() float64 {
	return math.Sqrt(v.dot(v))
}

func vectorAngle(a, b unitVector) float64 {
	return math.Atan2(a.cross(b).norm(), clamp(a.dot(b), -1, 1))
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func radiansToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}

func normalizeLongitude(longitude float64) float64 {
	normalized := math.Mod(longitude+180, 360)
	if normalized < 0 {
		normalized += 360
	}
	return normalized - 180
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Max(minimum, math.Min(maximum, value))
}
