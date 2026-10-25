package sdk

import (
	"encoding/json"
	"testing"
)

func TestTrailImportDifficultyJSON(t *testing.T) {
	for _, difficulty := range []string{"", "easy", "moderate", "difficult"} {
		t.Run("difficulty="+difficulty, func(t *testing.T) {
			payload, err := json.Marshal(DetailOutput{Item: TrailImport{Difficulty: difficulty}})
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Item map[string]json.RawMessage `json:"item"`
			}
			if err := json.Unmarshal(payload, &wire); err != nil {
				t.Fatal(err)
			}
			value, present := wire.Item["difficulty"]
			if difficulty == "" {
				if present {
					t.Fatalf("unknown difficulty must be omitted, got %s", value)
				}
			} else if string(value) != `"`+difficulty+`"` {
				t.Fatalf("difficulty = %s, want %q", value, difficulty)
			}
			var decoded DetailOutput
			if err := json.Unmarshal(payload, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Item.Difficulty != difficulty {
				t.Fatalf("round-trip difficulty = %q, want %q", decoded.Item.Difficulty, difficulty)
			}
		})
	}
}
