package core

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/open-wanderer/wanderer/plugins/sdk"
	"github.com/open-wanderer/wanderer/plugins/sdk/polyline"
)

const (
	DefaultTraceMaxShape          = 16000
	DefaultTraceMaxDistanceMeters = 200000.0
	DefaultTraceOverlap           = 10
	traceChunkSpliceTolerance     = 5.0
	traceChunkExactTolerance      = 0.5
	trackPartContinuityTolerance  = 25.0
	maxTrackPartBridgeMeters      = 50000.0
)

type ManeuverOptions struct {
	MaxShape          int
	MaxDistanceMeters float64
	Overlap           int
}

type TraceFunc func(TraceRequest) (TraceResponse, error)

type TraceRequest struct {
	Shape          []sdk.ManeuverPoint `json:"shape"`
	ShapeMatch     string              `json:"shape_match"`
	DirectionsType string              `json:"directions_type"`
	Costing        string              `json:"costing"`
	CostingOptions map[string]any      `json:"costing_options,omitempty"`
	Language       string              `json:"language,omitempty"`
	Units          string              `json:"units"`
}

type TraceResponse struct {
	Trip TraceTrip `json:"trip"`
}

type TraceTrip struct {
	Legs []TraceLeg `json:"legs"`
}

type TraceLeg struct {
	Shape     string          `json:"shape"`
	Maneuvers []TraceManeuver `json:"maneuvers"`
}

type TraceManeuver struct {
	Type                int      `json:"type"`
	Instruction         string   `json:"instruction"`
	Length              float64  `json:"length"`
	Time                float64  `json:"time"`
	BeginShapeIndex     int      `json:"begin_shape_index"`
	EndShapeIndex       int      `json:"end_shape_index"`
	BeginHeading        *float64 `json:"begin_heading,omitempty"`
	EndHeading          *float64 `json:"end_heading,omitempty"`
	RoundaboutExitCount *int     `json:"roundabout_exit_count,omitempty"`
	StreetNames         []string `json:"street_names,omitempty"`
}

type AdapterError struct {
	Code    string
	Message string
}

func (e *AdapterError) Error() string {
	return e.Message
}

func GenerateManeuvers(request sdk.ManeuverRequest, options ManeuverOptions, trace TraceFunc) (sdk.ManeuverResult, error) {
	if trace == nil {
		return sdk.ManeuverResult{}, &AdapterError{Code: "internal_error", Message: "trace adapter is unavailable"}
	}
	points, bridged, err := flattenTrackParts(request.TrackParts)
	if err != nil {
		return sdk.ManeuverResult{}, err
	}
	options = normalizedManeuverOptions(options)
	chunks := splitTracePoints(points, options)
	if len(chunks) == 0 {
		return sdk.ManeuverResult{}, &AdapterError{Code: "invalid_request", Message: "maneuver input has no usable track"}
	}
	costing := strings.TrimSpace(request.Profile.Key)
	if costing == "" {
		costing = costingForMode(request.Mode)
	}
	if costing == "" {
		return sdk.ManeuverResult{}, &AdapterError{Code: "unsupported_profile", Message: "maneuver routing profile is unavailable"}
	}
	costingOptions := valhallaCostingOptions(costing, request.Preferences)
	mergeNativeCostingOptions(costingOptions, costing, request.Profile.NativeConfig)

	var resultGeometry [][2]float64
	resultManeuvers := make([]sdk.Maneuver, 0)
	for index, chunk := range chunks {
		response, traceErr := trace(TraceRequest{
			Shape:          chunk,
			ShapeMatch:     "map_snap",
			DirectionsType: "instructions",
			Costing:        costing,
			CostingOptions: map[string]any{costing: costingOptions},
			Language:       request.Language,
			Units:          "kilometers",
		})
		if traceErr != nil {
			return sdk.ManeuverResult{}, traceErr
		}
		geometry, maneuvers, normalizeErr := normalizeTraceTrip(response.Trip, request.Limits)
		if normalizeErr != nil {
			return sdk.ManeuverResult{}, normalizeErr
		}
		resultGeometry, resultManeuvers, normalizeErr = appendTraceChunk(
			resultGeometry,
			resultManeuvers,
			geometry,
			maneuvers,
			index == len(chunks)-1,
		)
		if normalizeErr != nil {
			return sdk.ManeuverResult{}, normalizeErr
		}
	}
	if len(resultGeometry) == 0 || len(resultManeuvers) < 2 {
		return sdk.ManeuverResult{}, &AdapterError{Code: "provider_error", Message: "Valhalla returned no usable maneuvers"}
	}
	if request.Limits.MaxGeometryPoints > 0 && len(resultGeometry) > request.Limits.MaxGeometryPoints {
		return sdk.ManeuverResult{}, &AdapterError{Code: "response_limit_exceeded", Message: "matched maneuver geometry exceeds the host limit"}
	}
	if request.Limits.MaxManeuvers > 0 && len(resultManeuvers) > request.Limits.MaxManeuvers {
		return sdk.ManeuverResult{}, &AdapterError{Code: "response_limit_exceeded", Message: "maneuver count exceeds the host limit"}
	}
	if err := validateCompleteIntervals(resultManeuvers, len(resultGeometry), "Valhalla trace stitching"); err != nil {
		return sdk.ManeuverResult{}, err
	}
	setManeuverDistances(resultManeuvers, resultGeometry)
	warnings := []string{}
	if bridged {
		warnings = append(warnings, "segment_gap_bridged")
	}
	encodedGeometry, err := polyline.Encode(resultGeometry, 1e6)
	if err != nil {
		return sdk.ManeuverResult{}, &AdapterError{Code: "internal_error", Message: err.Error()}
	}
	return sdk.ManeuverResult{
		Geometry: sdk.ManeuverGeometry{
			Format:      "encoded_polyline",
			Precision:   6,
			Coordinates: encodedGeometry,
		},
		Maneuvers: resultManeuvers,
		Warnings:  warnings,
	}, nil
}

func mergeNativeCostingOptions(options map[string]any, costing string, nativeConfig map[string]any) {
	for key, value := range nativeConfig {
		if key == costing {
			if nested, ok := value.(map[string]any); ok {
				for nestedKey, nestedValue := range nested {
					options[nestedKey] = nestedValue
				}
			}
			continue
		}
		if _, nested := value.(map[string]any); !nested {
			options[key] = value
		}
	}
}

func normalizedManeuverOptions(options ManeuverOptions) ManeuverOptions {
	if options.MaxShape < 2 {
		options.MaxShape = DefaultTraceMaxShape
	}
	if options.MaxDistanceMeters <= 0 || math.IsNaN(options.MaxDistanceMeters) || math.IsInf(options.MaxDistanceMeters, 0) {
		options.MaxDistanceMeters = DefaultTraceMaxDistanceMeters
	}
	if options.Overlap < 1 {
		options.Overlap = DefaultTraceOverlap
	}
	if options.Overlap >= options.MaxShape {
		options.Overlap = options.MaxShape - 1
	}
	return options
}

func flattenTrackParts(parts []sdk.ManeuverTrackPart) ([]sdk.ManeuverPoint, bool, error) {
	points := make([]sdk.ManeuverPoint, 0)
	bridged := false
	for _, part := range parts {
		if len(part.Points) == 0 {
			continue
		}
		for _, point := range part.Points {
			if !validPoint(point) {
				return nil, false, &AdapterError{Code: "invalid_request", Message: "maneuver track contains an invalid coordinate"}
			}
		}
		if len(points) > 0 {
			gap := distanceMeters(points[len(points)-1], part.Points[0])
			if gap > maxTrackPartBridgeMeters {
				return nil, false, &AdapterError{Code: "discontinuous_track", Message: "source track parts are too far apart to bridge reliably"}
			}
			if gap > trackPartContinuityTolerance {
				bridged = true
			}
		}
		points = append(points, part.Points...)
	}
	if len(points) < 2 {
		return nil, false, &AdapterError{Code: "invalid_request", Message: "maneuver track requires at least two points"}
	}
	return points, bridged, nil
}

func splitTracePoints(points []sdk.ManeuverPoint, options ManeuverOptions) [][]sdk.ManeuverPoint {
	if len(points) < 2 {
		return nil
	}
	chunks := make([][]sdk.ManeuverPoint, 0)
	start := 0
	for start < len(points)-1 {
		end := start + 1
		distance := distanceMeters(points[start], points[end])
		for end+1 < len(points) {
			nextDistance := distance + distanceMeters(points[end], points[end+1])
			if end-start+2 > options.MaxShape || nextDistance > options.MaxDistanceMeters {
				break
			}
			end++
			distance = nextDistance
		}
		chunk := append([]sdk.ManeuverPoint(nil), points[start:end+1]...)
		chunks = append(chunks, chunk)
		if end == len(points)-1 {
			break
		}
		nextStart := end - options.Overlap + 1
		if nextStart <= start {
			nextStart = start + 1
		}
		start = nextStart
	}
	return chunks
}

func normalizeTraceTrip(trip TraceTrip, limits sdk.ManeuverLimits) ([][2]float64, []sdk.Maneuver, error) {
	if len(trip.Legs) == 0 {
		return nil, nil, &AdapterError{Code: "provider_error", Message: "Valhalla returned no trace legs"}
	}
	geometry := make([][2]float64, 0)
	maneuvers := make([]sdk.Maneuver, 0)
	for legIndex, leg := range trip.Legs {
		legGeometry, err := polyline.Decode(leg.Shape, 1e6)
		if err != nil || len(legGeometry) == 0 {
			return nil, nil, &AdapterError{Code: "provider_error", Message: "Valhalla returned invalid trace geometry"}
		}
		offset := len(geometry)
		if len(geometry) > 0 {
			if !sameCoordinate(geometry[len(geometry)-1], legGeometry[0]) {
				return nil, nil, &AdapterError{Code: "provider_error", Message: "Valhalla trace legs are discontinuous"}
			}
			offset--
			legGeometry = legGeometry[1:]
		}
		geometry = append(geometry, legGeometry...)
		for maneuverIndex, native := range leg.Maneuvers {
			if legIndex > 0 && maneuverIndex == 0 && isValhallaStart(native.Type) {
				continue
			}
			if legIndex > 0 && len(maneuvers) > 0 && maneuvers[len(maneuvers)-1].Type == "destination" {
				maneuvers = maneuvers[:len(maneuvers)-1]
			}
			maneuver := maneuverFromValhalla(native, limits)
			maneuver.BeginShapeIndex += offset
			maneuver.EndShapeIndex += offset
			maneuvers = append(maneuvers, maneuver)
		}
	}
	if len(maneuvers) < 2 {
		return nil, nil, &AdapterError{Code: "provider_error", Message: "Valhalla returned too few maneuvers"}
	}
	if err := validateCompleteIntervals(maneuvers, len(geometry), "Valhalla response"); err != nil {
		return nil, nil, err
	}
	return geometry, maneuvers, nil
}

func appendTraceChunk(existingGeometry [][2]float64, existingManeuvers []sdk.Maneuver, geometry [][2]float64, maneuvers []sdk.Maneuver, final bool) ([][2]float64, []sdk.Maneuver, error) {
	if len(existingGeometry) == 0 {
		if !final && len(maneuvers) > 0 && maneuvers[len(maneuvers)-1].Type == "destination" {
			maneuvers = maneuvers[:len(maneuvers)-1]
		}
		return append(existingGeometry, geometry...), append(existingManeuvers, maneuvers...), nil
	}
	splice, ok := findTraceGeometrySplice(existingGeometry[len(existingGeometry)-1], geometry)
	if !ok {
		return nil, nil, &AdapterError{Code: "discontinuous_track", Message: "overlapping Valhalla trace batches could not be joined"}
	}
	boundary := len(existingGeometry) - 1
	firstAppended := splice.segment + 1
	if splice.fraction >= 1-1e-9 || sameCoordinate(existingGeometry[boundary], geometry[splice.segment+1]) {
		firstAppended++
	}
	if firstAppended >= len(geometry) {
		return nil, nil, &AdapterError{Code: "discontinuous_track", Message: "overlapping Valhalla trace batch adds no forward geometry"}
	}
	existingGeometry = append(existingGeometry, geometry[firstAppended:]...)
	appendedFromChunk := false
	for index, maneuver := range maneuvers {
		if index == 0 && maneuver.Type == "start" {
			continue
		}
		if !final && index == len(maneuvers)-1 && maneuver.Type == "destination" {
			continue
		}
		if maneuver.EndShapeIndex < firstAppended {
			continue
		}
		maneuver.BeginShapeIndex = stitchedShapeIndex(maneuver.BeginShapeIndex, firstAppended, boundary)
		maneuver.EndShapeIndex = stitchedShapeIndex(maneuver.EndShapeIndex, firstAppended, boundary)
		if len(existingManeuvers) > 0 && (!appendedFromChunk || maneuver.BeginShapeIndex < existingManeuvers[len(existingManeuvers)-1].EndShapeIndex) {
			maneuver.BeginShapeIndex = existingManeuvers[len(existingManeuvers)-1].EndShapeIndex
		}
		if maneuver.EndShapeIndex < maneuver.BeginShapeIndex {
			continue
		}
		existingManeuvers = append(existingManeuvers, maneuver)
		appendedFromChunk = true
	}
	return existingGeometry, existingManeuvers, nil
}

func validateCompleteIntervals(maneuvers []sdk.Maneuver, geometryLength int, source string) error {
	if geometryLength < 2 || len(maneuvers) < 2 {
		return &AdapterError{Code: "provider_error", Message: source + " has incomplete maneuver geometry"}
	}
	lastGeometryIndex := geometryLength - 1
	for index, maneuver := range maneuvers {
		if maneuver.BeginShapeIndex < 0 || maneuver.BeginShapeIndex > maneuver.EndShapeIndex || maneuver.EndShapeIndex > lastGeometryIndex {
			return &AdapterError{Code: "provider_error", Message: source + " contains an invalid maneuver interval"}
		}
		if index > 0 && maneuver.BeginShapeIndex != maneuvers[index-1].EndShapeIndex {
			return &AdapterError{Code: "provider_error", Message: source + " contains maneuver intervals with gaps"}
		}
	}
	first := maneuvers[0]
	last := maneuvers[len(maneuvers)-1]
	if first.Type != "start" || first.BeginShapeIndex != 0 {
		return &AdapterError{Code: "provider_error", Message: source + " has no valid maneuver start"}
	}
	if last.Type != "destination" || last.BeginShapeIndex != last.EndShapeIndex || last.EndShapeIndex != lastGeometryIndex {
		return &AdapterError{Code: "provider_error", Message: source + " has no valid maneuver destination"}
	}
	return nil
}

func setManeuverDistances(maneuvers []sdk.Maneuver, geometry [][2]float64) {
	for index := range maneuvers {
		maneuver := &maneuvers[index]
		if maneuver.BeginShapeIndex == maneuver.EndShapeIndex {
			maneuver.DistanceMeters = 0
			continue
		}
		distance := 0.0
		for pointIndex := maneuver.BeginShapeIndex; pointIndex < maneuver.EndShapeIndex; pointIndex++ {
			distance += coordinateDistanceMeters(geometry[pointIndex], geometry[pointIndex+1])
		}
		maneuver.DistanceMeters = distance
	}
}

func maneuverFromValhalla(native TraceManeuver, limits sdk.ManeuverLimits) sdk.Maneuver {
	typeName, known := valhallaManeuverType(native.Type)
	warnings := []string{}
	if !known {
		warnings = append(warnings, "maneuver_type_unknown")
	}
	providerInstruction := boundedString(native.Instruction, limits.MaxProviderInstructionCharacters)
	streetNames := native.StreetNames
	if limits.MaxStreetNames > 0 && len(streetNames) > limits.MaxStreetNames {
		streetNames = streetNames[:limits.MaxStreetNames]
	}
	streetNames = append([]string(nil), streetNames...)
	for index := range streetNames {
		streetNames[index] = boundedString(streetNames[index], limits.MaxStreetNameCharacters)
	}
	duration := native.Time
	return sdk.Maneuver{
		Type:                typeName,
		ProviderInstruction: providerInstruction,
		DistanceMeters:      0,
		DurationSeconds:     &duration,
		BeginShapeIndex:     native.BeginShapeIndex,
		EndShapeIndex:       native.EndShapeIndex,
		BearingBefore:       normalizedBearing(native.BeginHeading),
		BearingAfter:        normalizedBearing(native.EndHeading),
		RoundaboutExit:      native.RoundaboutExitCount,
		StreetNames:         streetNames,
		Warnings:            warnings,
	}
}

func valhallaManeuverType(value int) (string, bool) {
	switch value {
	case 1, 2, 3:
		return "start", true
	case 4, 5, 6:
		return "destination", true
	case 7, 8, 22:
		return "continue", true
	case 9:
		return "turn_slight_right", true
	case 10:
		return "turn_right", true
	case 11:
		return "turn_sharp_right", true
	case 12:
		return "uturn_right", true
	case 13:
		return "uturn_left", true
	case 14:
		return "turn_sharp_left", true
	case 15:
		return "turn_left", true
	case 16:
		return "turn_slight_left", true
	case 17:
		return "ramp_straight", true
	case 18:
		return "ramp_right", true
	case 19:
		return "ramp_left", true
	case 20:
		return "exit_right", true
	case 21:
		return "exit_left", true
	case 23:
		return "keep_right", true
	case 24:
		return "keep_left", true
	case 25:
		return "merge", true
	case 26:
		return "roundabout_enter", true
	case 27:
		return "roundabout_exit", true
	case 28, 29:
		return "ferry", true
	default:
		return "unknown", false
	}
}

func isValhallaStart(value int) bool {
	return value == 1 || value == 2 || value == 3
}

type traceGeometrySplice struct {
	segment  int
	fraction float64
	distance float64
}

func findTraceGeometrySplice(point [2]float64, geometry [][2]float64) (traceGeometrySplice, bool) {
	if len(geometry) < 2 {
		return traceGeometrySplice{}, false
	}
	best := traceGeometrySplice{distance: math.Inf(1)}
	for index := 0; index+1 < len(geometry); index++ {
		fraction, distance := projectCoordinateToSegment(point, geometry[index], geometry[index+1])
		candidate := traceGeometrySplice{segment: index, fraction: fraction, distance: distance}
		if distance <= traceChunkExactTolerance {
			return candidate, true
		}
		if distance < best.distance {
			best = candidate
		}
	}
	return best, best.distance <= traceChunkSpliceTolerance
}

func projectCoordinateToSegment(point, start, end [2]float64) (float64, float64) {
	const earthRadius = 6371000.0
	latitudeScale := math.Pi / 180 * earthRadius
	longitudeScale := latitudeScale * math.Cos(point[0]*math.Pi/180)
	startX := (start[1] - point[1]) * longitudeScale
	startY := (start[0] - point[0]) * latitudeScale
	endX := (end[1] - point[1]) * longitudeScale
	endY := (end[0] - point[0]) * latitudeScale
	deltaX := endX - startX
	deltaY := endY - startY
	lengthSquared := deltaX*deltaX + deltaY*deltaY
	fraction := 0.0
	if lengthSquared > 0 {
		fraction = -(startX*deltaX + startY*deltaY) / lengthSquared
		if fraction < 0 {
			fraction = 0
		} else if fraction > 1 {
			fraction = 1
		}
	}
	closestX := startX + fraction*deltaX
	closestY := startY + fraction*deltaY
	return fraction, math.Hypot(closestX, closestY)
}

func stitchedShapeIndex(index, firstAppended, boundary int) int {
	if index < firstAppended {
		return boundary
	}
	return boundary + 1 + index - firstAppended
}

func sameCoordinate(left, right [2]float64) bool {
	return math.Abs(left[0]-right[0]) < 0.0000005 && math.Abs(left[1]-right[1]) < 0.0000005
}

func validPoint(point sdk.ManeuverPoint) bool {
	return !math.IsNaN(point.Lat) && !math.IsInf(point.Lat, 0) && !math.IsNaN(point.Lon) && !math.IsInf(point.Lon, 0) && point.Lat >= -90 && point.Lat <= 90 && point.Lon >= -180 && point.Lon <= 180
}

func distanceMeters(left, right sdk.ManeuverPoint) float64 {
	return coordinateDistanceMeters([2]float64{left.Lat, left.Lon}, [2]float64{right.Lat, right.Lon})
}

func coordinateDistanceMeters(left, right [2]float64) float64 {
	const earthRadius = 6371000.0
	lat1 := left[0] * math.Pi / 180
	lat2 := right[0] * math.Pi / 180
	dLat := (right[0] - left[0]) * math.Pi / 180
	dLon := (right[1] - left[1]) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func normalizedBearing(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	normalized := math.Mod(*value, 360)
	if normalized < 0 {
		normalized += 360
	}
	return &normalized
}

func boundedString(value string, maximum int) string {
	if maximum <= 0 || utf8.RuneCountInString(value) <= maximum {
		return value
	}
	runes := []rune(value)
	return string(runes[:maximum])
}

func costingForMode(mode string) string {
	switch mode {
	case "foot":
		return "pedestrian"
	case "bike":
		return "bicycle"
	case "motor":
		return "auto"
	default:
		return ""
	}
}

func valhallaCostingOptions(costing string, preferences map[string]any) map[string]any {
	options := map[string]any{}
	switch costing {
	case "pedestrian":
		setNumberOption(options, "walking_speed", preferences, "speedPreference")
		setNumberOption(options, "use_hills", preferences, "hillPreference")
		setNumberOption(options, "max_hiking_difficulty", preferences, "maxHikingDifficulty")
	case "bicycle":
		setNumberOption(options, "cycling_speed", preferences, "speedPreference")
		setNumberOption(options, "use_hills", preferences, "hillPreference")
		setNumberOption(options, "use_roads", preferences, "roadPreference")
		setNumberOption(options, "avoid_bad_surfaces", preferences, "avoidBadSurfaces")
	case "auto":
		setNumberOption(options, "top_speed", preferences, "speedPreference")
		setNumberOption(options, "width", preferences, "vehicleWidth")
		setNumberOption(options, "height", preferences, "vehicleHeight")
	}
	return options
}

func setNumberOption(options map[string]any, optionKey string, preferences map[string]any, preferenceKey string) {
	value, ok := preferences[preferenceKey]
	if !ok {
		return
	}
	switch value := value.(type) {
	case float64:
		options[optionKey] = value
	case float32:
		options[optionKey] = value
	case int:
		options[optionKey] = value
	case int64:
		options[optionKey] = value
	}
}

func DebugChunks(points []sdk.ManeuverPoint, options ManeuverOptions) ([][]sdk.ManeuverPoint, error) {
	options = normalizedManeuverOptions(options)
	if len(points) < 2 {
		return nil, fmt.Errorf("at least two points are required")
	}
	return splitTracePoints(points, options), nil
}
