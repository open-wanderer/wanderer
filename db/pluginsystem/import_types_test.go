package pluginsystem

import (
	"encoding/json"
	"testing"
)

func TestTrailImportDifficultyJSONCompatibility(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
		want    string
	}{
		{"legacy omitted", `{}`, ""},
		{"explicit empty", `{"difficulty":""}`, ""},
		{"null", `{"difficulty":null}`, ""},
		{"number", `{"difficulty":1}`, ""},
		{"boolean", `{"difficulty":true}`, ""},
		{"array", `{"difficulty":["easy"]}`, ""},
		{"object", `{"difficulty":{"grade":"easy"}}`, ""},
		{"legacy metadata", `{"metadata":{"difficulty":"easy"}}`, ""},
		{"easy", `{"difficulty":"easy"}`, "easy"},
		{"moderate", `{"difficulty":"moderate"}`, "moderate"},
		{"difficult", `{"difficulty":"difficult"}`, "difficult"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var item TrailImport
			if err := json.Unmarshal([]byte(test.payload), &item); err != nil {
				t.Fatal(err)
			}
			if item.Difficulty != test.want {
				t.Fatalf("difficulty = %q, want %q", item.Difficulty, test.want)
			}
			encoded, err := json.Marshal(item)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			field, exists := fields["difficulty"]
			if test.want == "" {
				if exists {
					t.Fatalf("unknown difficulty must be omitted, got %s", field)
				}
			} else if string(field) != `"`+test.want+`"` {
				t.Fatalf("encoded difficulty = %s, want %q", field, test.want)
			}
		})
	}
}

func TestTrailImportDifficultyIgnoresInvalidTypesWithoutDroppingOtherFields(t *testing.T) {
	for _, rawDifficulty := range []string{"1", "true", `["easy"]`, `{"grade":"easy"}`} {
		t.Run(rawDifficulty, func(t *testing.T) {
			payload := `{"name":"Provider trail","source":{"provider":"test","externalId":"123"},"track":{"format":"gpx","contentBase64":"track"},"difficulty":` + rawDifficulty + `}`
			var item TrailImport
			if err := json.Unmarshal([]byte(payload), &item); err != nil {
				t.Fatal(err)
			}
			if item.Difficulty != "" {
				t.Fatalf("difficulty = %q, want unknown", item.Difficulty)
			}
			if item.Name != "Provider trail" || item.Source.Provider != "test" || item.Source.ExternalID != "123" || item.Track.Format != "gpx" || item.Track.ContentBase64 != "track" {
				t.Fatalf("ignoring difficulty also discarded valid import data: %#v", item)
			}
		})
	}
}

func TestTrailImportDifficultyDecodeClearsPreviousValue(t *testing.T) {
	for _, payload := range []string{
		`{}`, `{"difficulty":""}`, `{"difficulty":null}`,
		`{"difficulty":1}`, `{"difficulty":true}`, `{"difficulty":[]}`, `{"difficulty":{}}`,
	} {
		t.Run(payload, func(t *testing.T) {
			item := TrailImport{Difficulty: "easy"}
			if err := json.Unmarshal([]byte(payload), &item); err != nil {
				t.Fatal(err)
			}
			if item.Difficulty != "" {
				t.Fatalf("decode retained previous difficulty %q for %s", item.Difficulty, payload)
			}
		})
	}
}

func TestTrailImportDifficultyLeniencyDoesNotHideOtherDecodingErrors(t *testing.T) {
	for _, payload := range []string{
		`{"difficulty":`,
		`{"difficulty":{},"name":`,
		`{"difficulty":"easy","name":123}`,
		`{"difficulty":true,"source":[]}`,
		`{"difficulty":[],"source":{"provider":123}}`,
		`{"difficulty":{},"track":{"format":false}}`,
		`{"difficulty":{},"startedAt":"not-a-date"}`,
		`{"difficulty":{},"waypoints":[{"lat":"invalid"}]}`,
		`[]`,
		`"easy"`,
	} {
		t.Run(payload, func(t *testing.T) {
			var item TrailImport
			if err := json.Unmarshal([]byte(payload), &item); err == nil {
				t.Fatalf("expected decoding error for %s", payload)
			}
		})
	}
}
