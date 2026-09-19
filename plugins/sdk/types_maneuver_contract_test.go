package sdk

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestManeuverV1WireContract(t *testing.T) {
	fixture := maneuverContractFixture(t)
	duration := 45.5
	bearingBefore := 90.5
	bearingAfter := 180.5
	roundaboutExit := 2

	request := ManeuverRequest{
		TrackParts:  []ManeuverTrackPart{{Points: []ManeuverPoint{{Lat: 47.1, Lon: 8.2}, {Lat: 47.2, Lon: 8.3}}}},
		Mode:        "foot",
		Category:    "Hiking",
		Subcategory: "Alpine",
		Profile: ManeuverProfile{
			ID: "profile-id", PluginID: "valhalla", Key: "pedestrian", Kind: "builtin", Mode: "foot",
			ContentBase64: "cHJvZmlsZQ==", ContentType: "text/plain",
			Metadata:     map[string]any{"revision": "one"},
			NativeConfig: map[string]any{"pedestrian": map[string]any{"shortest": true}},
			PreparedKey:  "prepared-profile",
		},
		Preferences:         map[string]any{"hillPreference": 0.75},
		RequiredPreferences: []string{"hillPreference"},
		Language:            "de",
		Limits: ManeuverLimits{
			MaxGeometryPoints: 20000, MaxManeuvers: 1000, MaxProviderInstructionCharacters: 500,
			MaxStreetNames: 4, MaxStreetNameCharacters: 120, MaxResponseBytes: 4194304,
		},
	}
	result := ManeuverResult{
		Geometry: ManeuverGeometry{Format: "encoded_polyline", Precision: 6, Coordinates: "encoded"},
		Maneuvers: []Maneuver{{
			Type: "roundabout_exit", ProviderInstruction: "Take the second exit", DistanceMeters: 123.5,
			DurationSeconds: &duration, BeginShapeIndex: 1, EndShapeIndex: 2,
			BearingBefore: &bearingBefore, BearingAfter: &bearingAfter, RoundaboutExit: &roundaboutExit,
			StreetNames: []string{"Example Street"}, Warnings: []string{"maneuver_type_unknown"},
		}},
		Warnings: []string{"input_simplified"},
	}

	assertManeuverContractJSON(t, request, fixture["request"])
	assertManeuverContractJSON(t, result, fixture["result"])
	assertManeuverContractJSON(t, ManeuverRequest{
		TrackParts: []ManeuverTrackPart{},
		Profile:    ManeuverProfile{},
		Limits:     ManeuverLimits{},
	}, fixture["sparseRequest"])
	assertManeuverContractJSON(t, ManeuverResult{
		Geometry:  ManeuverGeometry{},
		Maneuvers: []Maneuver{{Type: "unknown"}},
	}, fixture["sparseResult"])
	assertManeuverContractFields(t, fixture["fields"], map[string]reflect.Type{
		"point":     reflect.TypeOf(ManeuverPoint{}),
		"trackPart": reflect.TypeOf(ManeuverTrackPart{}),
		"profile":   reflect.TypeOf(ManeuverProfile{}),
		"limits":    reflect.TypeOf(ManeuverLimits{}),
		"request":   reflect.TypeOf(ManeuverRequest{}),
		"geometry":  reflect.TypeOf(ManeuverGeometry{}),
		"maneuver":  reflect.TypeOf(Maneuver{}),
		"result":    reflect.TypeOf(ManeuverResult{}),
	})
}

func maneuverContractFixture(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	payload, err := os.ReadFile("testdata/maneuvers_v1.json")
	if err != nil {
		t.Fatalf("read maneuver contract fixture: %v", err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatalf("decode maneuver contract fixture: %v", err)
	}
	return fixture
}

func assertManeuverContractJSON(t *testing.T, actual any, expected json.RawMessage) {
	t.Helper()
	payload, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal maneuver contract: %v", err)
	}
	var actualValue any
	var expectedValue any
	if err := json.Unmarshal(payload, &actualValue); err != nil {
		t.Fatalf("decode actual maneuver contract: %v", err)
	}
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("decode expected maneuver contract: %v", err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("maneuver wire contract mismatch\nactual: %s\nexpected: %s", payload, expected)
	}
}

func assertManeuverContractFields(t *testing.T, expected json.RawMessage, types map[string]reflect.Type) {
	t.Helper()
	var expectedFields map[string][]string
	if err := json.Unmarshal(expected, &expectedFields); err != nil {
		t.Fatalf("decode maneuver field contract: %v", err)
	}
	if len(expectedFields) != len(types) {
		t.Fatalf("maneuver field contract contains %d type entries, want %d", len(expectedFields), len(types))
	}
	for name, contractType := range types {
		assertJSONTagSet(t, name, contractType, expectedFields[name])
	}
}

func assertJSONTagSet(t *testing.T, name string, contractType reflect.Type, expectedFields []string) {
	t.Helper()
	actualFields := make(map[string]struct{}, contractType.NumField())
	for index := 0; index < contractType.NumField(); index++ {
		field := contractType.Field(index).Tag.Get("json")
		if field == "" {
			t.Fatalf("%s field %s has no JSON tag", name, contractType.Field(index).Name)
		}
		if _, duplicate := actualFields[field]; duplicate {
			t.Fatalf("%s field contract contains duplicate JSON tag %q", name, field)
		}
		actualFields[field] = struct{}{}
	}
	expectedFieldSet := make(map[string]struct{}, len(expectedFields))
	for _, field := range expectedFields {
		if _, duplicate := expectedFieldSet[field]; duplicate {
			t.Fatalf("%s fixture contains duplicate JSON tag %q", name, field)
		}
		expectedFieldSet[field] = struct{}{}
	}
	if !reflect.DeepEqual(actualFields, expectedFieldSet) {
		t.Fatalf("%s field contract = %#v, want %#v", name, actualFields, expectedFieldSet)
	}
}
