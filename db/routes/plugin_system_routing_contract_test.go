package routes

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"pocketbase/pluginsystem"
)

func TestHostManeuverV1WireContract(t *testing.T) {
	fixture := hostManeuverContractFixture(t)

	duration := 45.5
	bearingBefore := 90.5
	bearingAfter := 180.5
	roundaboutExit := 2
	request := pluginRoutingManeuverRequest{
		TrackParts:  []pluginRoutingManeuverTrackPart{{Points: []pluginRoutingManeuverPoint{{Lat: 47.1, Lon: 8.2}, {Lat: 47.2, Lon: 8.3}}}},
		Mode:        "foot",
		Category:    "Hiking",
		Subcategory: "Alpine",
		Profile: pluginRoutingProfile{
			ID: "profile-id", PluginID: "valhalla", Key: "pedestrian", Kind: "builtin", Mode: "foot",
			ContentBase64: "cHJvZmlsZQ==", ContentType: "text/plain",
			Metadata:     map[string]any{"revision": "one"},
			NativeConfig: map[string]any{"pedestrian": map[string]any{"shortest": true}},
			PreparedKey:  "prepared-profile",
		},
		Preferences:         map[string]any{"hillPreference": 0.75},
		RequiredPreferences: []string{"hillPreference"},
		Language:            "de",
		Limits: pluginRoutingManeuverLimits{
			MaxGeometryPoints: 20000, MaxManeuvers: 1000, MaxProviderInstructionCharacters: 500,
			MaxStreetNames: 4, MaxStreetNameCharacters: 120, MaxResponseBytes: 4194304,
		},
	}
	result := pluginRoutingManeuverOutput{
		Geometry: pluginRoutingGeometry{Format: "encoded_polyline", Precision: 6, Coordinates: "encoded"},
		Maneuvers: []pluginRoutingManeuver{{
			Type: "roundabout_exit", ProviderInstruction: "Take the second exit", DistanceMeters: 123.5,
			DurationSeconds: &duration, BeginShapeIndex: 1, EndShapeIndex: 2,
			BearingBefore: &bearingBefore, BearingAfter: &bearingAfter, RoundaboutExit: &roundaboutExit,
			StreetNames: []string{"Example Street"}, Warnings: []string{"maneuver_type_unknown"},
		}},
		Warnings: []string{"input_simplified"},
	}

	assertHostManeuverContractJSON(t, request, fixture["request"])
	assertHostManeuverContractJSON(t, result, fixture["result"])
	assertHostManeuverContractJSON(t, pluginRoutingManeuverRequest{
		TrackParts: []pluginRoutingManeuverTrackPart{},
		Profile:    pluginRoutingProfile{},
		Limits:     pluginRoutingManeuverLimits{},
	}, fixture["sparseRequest"])
	assertHostManeuverContractJSON(t, pluginRoutingManeuverOutput{
		Geometry:  pluginRoutingGeometry{},
		Maneuvers: []pluginRoutingManeuver{{Type: "unknown"}},
	}, fixture["sparseResult"])
	assertHostManeuverContractFields(t, fixture["fields"], map[string]reflect.Type{
		"point":     reflect.TypeOf(pluginRoutingManeuverPoint{}),
		"trackPart": reflect.TypeOf(pluginRoutingManeuverTrackPart{}),
		"profile":   reflect.TypeOf(pluginRoutingProfile{}),
		"limits":    reflect.TypeOf(pluginRoutingManeuverLimits{}),
		"request":   reflect.TypeOf(pluginRoutingManeuverRequest{}),
		"geometry":  reflect.TypeOf(pluginRoutingGeometry{}),
		"maneuver":  reflect.TypeOf(pluginRoutingManeuver{}),
		"result":    reflect.TypeOf(pluginRoutingManeuverOutput{}),
	})
}

func TestHostPluginErrorWireContract(t *testing.T) {
	fixture := hostManeuverContractFixture(t)
	var expectedFields []string
	if err := json.Unmarshal(fixture["pluginError"], &expectedFields); err != nil {
		t.Fatalf("decode plugin error field contract: %v", err)
	}
	assertHostJSONTagSet(t, "PluginError", reflect.TypeOf(pluginsystem.PluginError{}), expectedFields)
}

func TestManeuverV1ContractFixtureCopiesMatch(t *testing.T) {
	hostFixture, err := os.ReadFile("testdata/maneuvers_v1.json")
	if err != nil {
		t.Fatalf("read host maneuver contract fixture: %v", err)
	}
	sdkFixture, err := os.ReadFile("../../plugins/sdk/testdata/maneuvers_v1.json")
	if os.IsNotExist(err) {
		t.Skip("SDK maneuver fixture is unavailable outside a full repository checkout")
	}
	if err != nil {
		t.Fatalf("read SDK maneuver contract fixture: %v", err)
	}
	if !bytes.Equal(hostFixture, sdkFixture) {
		t.Fatal("host and SDK maneuver contract fixture copies differ")
	}
}

func hostManeuverContractFixture(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	payload, err := os.ReadFile("testdata/maneuvers_v1.json")
	if err != nil {
		t.Fatalf("read host maneuver contract fixture: %v", err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fixture); err != nil {
		t.Fatalf("decode host maneuver contract fixture: %v", err)
	}
	return fixture
}

func assertHostManeuverContractJSON(t *testing.T, actual any, expected json.RawMessage) {
	t.Helper()
	payload, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal host maneuver contract: %v", err)
	}
	var actualValue any
	var expectedValue any
	if err := json.Unmarshal(payload, &actualValue); err != nil {
		t.Fatalf("decode actual host maneuver contract: %v", err)
	}
	if err := json.Unmarshal(expected, &expectedValue); err != nil {
		t.Fatalf("decode expected host maneuver contract: %v", err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("host maneuver wire contract mismatch\nactual: %s\nexpected: %s", payload, expected)
	}
}

func assertHostManeuverContractFields(t *testing.T, expected json.RawMessage, types map[string]reflect.Type) {
	t.Helper()
	var expectedFields map[string][]string
	if err := json.Unmarshal(expected, &expectedFields); err != nil {
		t.Fatalf("decode maneuver field contract: %v", err)
	}
	if len(expectedFields) != len(types) {
		t.Fatalf("maneuver field contract contains %d type entries, want %d", len(expectedFields), len(types))
	}
	for name, contractType := range types {
		assertHostJSONTagSet(t, name, contractType, expectedFields[name])
	}
}

func assertHostJSONTagSet(t *testing.T, name string, contractType reflect.Type, expectedFields []string) {
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
