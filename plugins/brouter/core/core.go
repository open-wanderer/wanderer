package core

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/open-wanderer/wanderer/plugins/sdk/polyline"
)

const (
	EncodedPolylineFormat    = "encoded_polyline"
	EncodedPolylinePrecision = 6
	EncodedPolylineScale     = 1e6
	MaxRouteCandidates       = 4
	RoundTripTolerance       = 0.10
	RoundTripMaxAttempts     = 3
	RoundTripPointCount      = 5
	RoundTripMinRadius       = 250.0
	RoundTripMaxRadius       = 100000.0
)

type RouteRequest struct {
	RoutingMode         string         `json:"routingMode"`
	Anchors             []Anchor       `json:"anchors"`
	Mode                string         `json:"mode,omitempty"`
	Profile             RoutingProfile `json:"profile"`
	Preferences         map[string]any `json:"preferences,omitempty"`
	RequiredPreferences []string       `json:"requiredPreferences,omitempty"`
	Options             RouteOptions   `json:"options,omitempty"`
}

type Anchor struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type RoutingProfile struct {
	ID            string         `json:"id,omitempty"`
	PluginID      string         `json:"pluginId,omitempty"`
	Key           string         `json:"key"`
	Kind          string         `json:"kind,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	ContentBase64 string         `json:"contentBase64,omitempty"`
	ContentType   string         `json:"contentType,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	NativeConfig  map[string]any `json:"nativeConfig,omitempty"`
	PreparedKey   string         `json:"preparedKey,omitempty"`
}

type RouteOptions struct {
	Alternatives     int  `json:"alternatives,omitempty"`
	IncludeElevation bool `json:"includeElevation,omitempty"`
}

type RouteOutput struct {
	Candidates []RouteCandidate `json:"candidates,omitempty"`
	Error      any              `json:"error,omitempty"`
}

type RoundTripRequest struct {
	Start               Anchor         `json:"start"`
	TargetDistance      float64        `json:"targetDistance"`
	Direction           *float64       `json:"direction,omitempty"`
	Seed                string         `json:"seed,omitempty"`
	Mode                string         `json:"mode,omitempty"`
	Profile             RoutingProfile `json:"profile"`
	Preferences         map[string]any `json:"preferences,omitempty"`
	RequiredPreferences []string       `json:"requiredPreferences,omitempty"`
	Options             RouteOptions   `json:"options,omitempty"`
}

type RoundTripMetadata struct {
	TargetDistance float64 `json:"targetDistance"`
	ActualDistance float64 `json:"actualDistance"`
	Direction      *int    `json:"direction,omitempty"`
	Attempts       int     `json:"attempts"`
	Tolerance      float64 `json:"tolerance"`
}

type RouteCandidate struct {
	ID               string             `json:"id"`
	ProfileKey       string             `json:"profileKey,omitempty"`
	Geometry         *Geometry          `json:"geometry,omitempty"`
	Elevation        *Elevation         `json:"elevation,omitempty"`
	Summary          Summary            `json:"summary"`
	Segments         []Segment          `json:"segments"`
	SnappedAnchors   []Anchor           `json:"snappedAnchors,omitempty"`
	SuggestedAnchors []Anchor           `json:"suggestedAnchors,omitempty"`
	RoundTrip        *RoundTripMetadata `json:"roundTrip,omitempty"`
	Warnings         []string           `json:"warnings,omitempty"`
}

type Elevation struct {
	Heights []float64 `json:"heights,omitempty"`
	Status  string    `json:"status,omitempty"`
	Source  string    `json:"source,omitempty"`
}

type Geometry struct {
	Format      string `json:"format"`
	Precision   int    `json:"precision"`
	Coordinates string `json:"coordinates"`
}

type Summary struct {
	Distance      float64 `json:"distance"`
	Duration      float64 `json:"duration"`
	ElevationGain float64 `json:"elevationGain,omitempty"`
	ElevationLoss float64 `json:"elevationLoss,omitempty"`
}

type Segment struct {
	FromAnchor int      `json:"fromAnchor"`
	ToAnchor   int      `json:"toAnchor"`
	Geometry   Geometry `json:"geometry"`
	Distance   float64  `json:"distance"`
	Duration   float64  `json:"duration"`
}

type FeatureCollection struct {
	Features []Feature `json:"features"`
}

type RawFeatureCollection struct {
	Features []json.RawMessage `json:"features"`
}

type Feature struct {
	Properties Properties `json:"properties"`
	Geometry   LineString `json:"geometry"`
}

type Properties struct {
	TrackLength string      `json:"track-length"`
	TotalTime   string      `json:"total-time"`
	Ascend      string      `json:"filtered ascend"`
	Times       []float64   `json:"times"`
	Messages    [][]string  `json:"messages"`
	Raw         interface{} `json:"-"`
}

type LineString struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

func ParseRoundTripFeatures(features []json.RawMessage, start Anchor) (Feature, []Anchor, error) {
	var route Feature
	foundRoute := false
	suggestions := []Anchor{}
	for _, raw := range features {
		var header struct {
			Geometry struct {
				Type string `json:"type"`
			} `json:"geometry"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			continue
		}
		switch header.Geometry.Type {
		case "LineString":
			if foundRoute {
				continue
			}
			if err := json.Unmarshal(raw, &route); err != nil {
				return Feature{}, nil, err
			}
			foundRoute = true
		case "Point":
			var point struct {
				Properties map[string]any `json:"properties"`
				Geometry   struct {
					Coordinates []float64 `json:"coordinates"`
				} `json:"geometry"`
			}
			if err := json.Unmarshal(raw, &point); err != nil || len(point.Geometry.Coordinates) < 2 {
				continue
			}
			name, _ := point.Properties["name"].(string)
			name = strings.ToLower(strings.TrimSpace(name))
			candidate := Anchor{Lat: point.Geometry.Coordinates[1], Lon: point.Geometry.Coordinates[0]}
			// BRouter names the input and synthetic closing point "from" and
			// "to_rt" respectively; neither is an edit anchor.
			if name == "from" || name == "to_rt" || DistanceMeters([2]float64{start.Lat, start.Lon}, [2]float64{candidate.Lat, candidate.Lon}) < 1 {
				continue
			}
			suggestions = append(suggestions, candidate)
		}
	}
	if !foundRoute {
		return Feature{}, nil, fmt.Errorf("BRouter returned no round-trip line feature")
	}
	return route, suggestions, nil
}

func CandidateFromFeature(req RouteRequest, profileKey string, feature Feature, parseFloat func(string) float64) (RouteCandidate, error) {
	coords := Coordinates(feature.Geometry.Coordinates)
	if len(coords) < 2 {
		return RouteCandidate{}, fmt.Errorf("BRouter returned an empty route geometry")
	}
	heights := Heights(feature.Geometry.Coordinates)
	segments := SegmentsFromCoordinates(req, coords, feature.Properties)
	routeGeometry := Geometry{
		Format:      EncodedPolylineFormat,
		Precision:   EncodedPolylinePrecision,
		Coordinates: encodePolyline(coords, EncodedPolylineScale),
	}
	distance := parseFloat(feature.Properties.TrackLength)
	if distance == 0 {
		for _, segment := range segments {
			distance += segment.Distance
		}
	}
	duration := parseFloat(feature.Properties.TotalTime)
	if duration == 0 {
		for _, segment := range segments {
			duration += segment.Duration
		}
	}
	candidate := RouteCandidate{
		ID:         "primary",
		ProfileKey: profileKey,
		Geometry:   &routeGeometry,
		Elevation:  RouteElevation(heights),
		Summary: Summary{
			Distance:      distance,
			Duration:      duration,
			ElevationGain: parseFloat(feature.Properties.Ascend),
		},
		Segments:       segments,
		SnappedAnchors: SnappedAnchors(req.Anchors, coords),
	}
	return candidate, nil
}

func RoundTripCandidateFromFeature(profileKey string, feature Feature, suggested []Anchor, targetDistance float64, direction int, attempts int, parseFloat func(string) float64) (RouteCandidate, error) {
	coords := Coordinates(feature.Geometry.Coordinates)
	if len(coords) < 3 {
		return RouteCandidate{}, fmt.Errorf("BRouter returned an empty round-trip geometry")
	}
	distance := parseFloat(feature.Properties.TrackLength)
	if distance == 0 {
		distance = PolylineDistance(coords)
	}
	geometry := Geometry{
		Format:      EncodedPolylineFormat,
		Precision:   EncodedPolylinePrecision,
		Coordinates: encodePolyline(coords, EncodedPolylineScale),
	}
	var effectiveDirection *int
	if direction >= 0 {
		effectiveDirection = &direction
	}
	return RouteCandidate{
		ID:         "primary",
		ProfileKey: profileKey,
		Geometry:   &geometry,
		Elevation:  RouteElevation(Heights(feature.Geometry.Coordinates)),
		Summary: Summary{
			Distance:      distance,
			Duration:      parseFloat(feature.Properties.TotalTime),
			ElevationGain: parseFloat(feature.Properties.Ascend),
		},
		SuggestedAnchors: append([]Anchor(nil), suggested...),
		RoundTrip: &RoundTripMetadata{
			TargetDistance: targetDistance,
			ActualDistance: distance,
			Direction:      effectiveDirection,
			Attempts:       attempts,
			Tolerance:      RoundTripTolerance,
		},
	}, nil
}

func InitialRoundTripRadius(targetDistance float64) float64 {
	return clampRoundTripRadius(targetDistance / (2 * math.Pi))
}

func NextRoundTripRadius(radius float64, targetDistance float64, actualDistance float64) float64 {
	if radius <= 0 || targetDistance <= 0 || actualDistance <= 0 || math.IsNaN(actualDistance) || math.IsInf(actualDistance, 0) {
		return clampRoundTripRadius(radius)
	}
	return clampRoundTripRadius(radius * targetDistance / actualDistance)
}

func RoundTripWithinTolerance(targetDistance float64, actualDistance float64) bool {
	if targetDistance <= 0 || actualDistance <= 0 {
		return false
	}
	return math.Abs(actualDistance-targetDistance)/targetDistance <= RoundTripTolerance
}

func CalibrateRoundTrip(targetDistance float64, generate func(radius float64, attempt int) (RouteCandidate, error)) (RouteCandidate, error) {
	radius := InitialRoundTripRadius(targetDistance)
	bestError := math.Inf(1)
	var best *RouteCandidate
	successfulAttempts := 0
	for attempt := 1; attempt <= RoundTripMaxAttempts; attempt++ {
		candidate, err := generate(radius, attempt)
		if err != nil {
			if best == nil {
				return RouteCandidate{}, err
			}
			best.Warnings = append(best.Warnings, "round_trip_distance_adjustment_incomplete")
			break
		}
		successfulAttempts++
		relativeError := math.Abs(candidate.Summary.Distance-targetDistance) / targetDistance
		if relativeError < bestError {
			candidateCopy := candidate
			best = &candidateCopy
			bestError = relativeError
		}
		if RoundTripWithinTolerance(targetDistance, candidate.Summary.Distance) {
			break
		}
		radius = NextRoundTripRadius(radius, targetDistance, candidate.Summary.Distance)
	}
	if best == nil {
		return RouteCandidate{}, fmt.Errorf("round-trip calibration returned no candidate")
	}
	if best.RoundTrip != nil {
		best.RoundTrip.Attempts = successfulAttempts
	}
	if !RoundTripWithinTolerance(targetDistance, best.Summary.Distance) {
		best.Warnings = append(best.Warnings, "round_trip_target_tolerance_not_met")
	}
	return *best, nil
}

func RoundTripDirection(direction *float64, seed string) int {
	if direction != nil {
		value := math.Mod(*direction, 360)
		if value < 0 {
			value += 360
		}
		return int(math.Floor(value+0.5)) % 360
	}
	if seed == "" {
		return -1
	}
	// FNV-1a keeps seeded requests stable without relying on provider state.
	var hash uint32 = 2166136261
	for i := 0; i < len(seed); i++ {
		hash ^= uint32(seed[i])
		hash *= 16777619
	}
	return int(hash % 360)
}

func clampRoundTripRadius(radius float64) float64 {
	if radius < RoundTripMinRadius {
		return RoundTripMinRadius
	}
	if radius > RoundTripMaxRadius {
		return RoundTripMaxRadius
	}
	return radius
}

func RouteElevation(heights []float64) *Elevation {
	if len(heights) == 0 {
		return nil
	}
	return &Elevation{Heights: heights, Status: "included", Source: "brouter"}
}

func SegmentsFromCoordinates(req RouteRequest, coords [][2]float64, properties Properties) []Segment {
	segments := make([]Segment, 0, len(req.Anchors)-1)
	splits := SplitCoordinatesByAnchors(req.Anchors, coords)
	for i := 0; i < len(req.Anchors)-1; i++ {
		start := splits[i]
		end := splits[i+1]
		if end <= start {
			end = start + 1
		}
		if end >= len(coords) {
			end = len(coords) - 1
		}
		part := append([][2]float64(nil), coords[start:end+1]...)
		duration := 0.0
		if len(properties.Times) == len(coords) {
			duration = properties.Times[end] - properties.Times[start]
		}
		segments = append(segments, Segment{
			FromAnchor: i,
			ToAnchor:   i + 1,
			Geometry: Geometry{
				Format:      EncodedPolylineFormat,
				Precision:   EncodedPolylinePrecision,
				Coordinates: encodePolyline(part, EncodedPolylineScale),
			},
			Distance: PolylineDistance(part),
			Duration: duration,
		})
	}
	return segments
}

func SplitCoordinatesByAnchors(anchors []Anchor, coords [][2]float64) []int {
	splits := make([]int, len(anchors))
	last := 0
	for i, a := range anchors {
		best := last
		bestDistance := DistanceMeters(coords[last], [2]float64{a.Lat, a.Lon})
		for j := last; j < len(coords); j++ {
			d := DistanceMeters(coords[j], [2]float64{a.Lat, a.Lon})
			if d < bestDistance {
				best = j
				bestDistance = d
			}
		}
		splits[i] = best
		last = best
	}
	splits[0] = 0
	splits[len(splits)-1] = len(coords) - 1
	return splits
}

func SnappedAnchors(anchors []Anchor, coords [][2]float64) []Anchor {
	splits := SplitCoordinatesByAnchors(anchors, coords)
	result := make([]Anchor, len(anchors))
	for i, index := range splits {
		result[i] = Anchor{Lat: coords[index][0], Lon: coords[index][1]}
	}
	return result
}

func Coordinates(raw [][]float64) [][2]float64 {
	coords := make([][2]float64, 0, len(raw))
	for _, point := range raw {
		if len(point) < 2 {
			continue
		}
		coords = append(coords, [2]float64{point[1], point[0]})
	}
	return coords
}

func Heights(raw [][]float64) []float64 {
	heights := make([]float64, 0, len(raw))
	for _, point := range raw {
		if len(point) < 3 {
			return nil
		}
		heights = append(heights, point[2])
	}
	return heights
}

func encodePolyline(coords [][2]float64, precision float64) string {
	encoded, _ := polyline.Encode(coords, precision)
	return encoded
}

func PolylineDistance(coords [][2]float64) float64 {
	total := 0.0
	for i := 1; i < len(coords); i++ {
		total += DistanceMeters(coords[i-1], coords[i])
	}
	return total
}

func DistanceMeters(a [2]float64, b [2]float64) float64 {
	const earthRadius = 6371000
	lat1 := a[0] * math.Pi / 180
	lat2 := b[0] * math.Pi / 180
	dLat := (b[0] - a[0]) * math.Pi / 180
	dLon := (b[1] - a[1]) * math.Pi / 180
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return earthRadius * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}
