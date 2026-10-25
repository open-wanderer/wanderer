package pluginsystem

import (
	"encoding/json"
	"strings"
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

// A non-string difficulty is invalid plugin output, not an unknown rating.
func TestTrailImportDifficultyRejectsNonStringValues(t *testing.T) {
	for _, rawDifficulty := range []string{"1", "true", `["easy"]`, `{"grade":"easy"}`} {
		t.Run(rawDifficulty, func(t *testing.T) {
			payload := `{"name":"Provider trail","difficulty":` + rawDifficulty + `}`
			var item TrailImport
			err := json.Unmarshal([]byte(payload), &item)
			if err == nil {
				t.Fatalf("expected a decoding error for difficulty %s", rawDifficulty)
			}
			if !strings.Contains(err.Error(), "difficulty") {
				t.Fatalf("decoding error does not name the difficulty field: %v", err)
			}
		})
	}
}
