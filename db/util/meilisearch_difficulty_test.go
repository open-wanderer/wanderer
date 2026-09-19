package util

import (
	"encoding/json"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestTrailSearchDocumentDifficulty(t *testing.T) {
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(&core.SelectField{
		Name: "difficulty", MaxSelect: 1,
		Values: []string{"easy", "moderate", "difficult"},
	})
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.BoolField{Name: "is_local"})
	author := core.NewRecord(actors)
	author.Set("is_local", true)

	for _, test := range []struct {
		name       string
		difficulty any
		wantJSON   string
	}{
		{"missing", nil, "null"},
		{"empty", "", "null"},
		{"unrecognized", "unknown", "null"},
		{"easy", "easy", "0"},
		{"moderate", "moderate", "1"},
		{"difficult", "difficult", "2"},
	} {
		t.Run(test.name, func(t *testing.T) {
			trail := core.NewRecord(trails)
			if test.difficulty != nil {
				trail.Set("difficulty", test.difficulty)
			}
			document, err := documentFromTrailRecord(trail, author, true)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(document)
			if err != nil {
				t.Fatal(err)
			}
			var decoded map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &decoded); err != nil {
				t.Fatal(err)
			}
			if got := string(decoded["difficulty"]); got != test.wantJSON {
				t.Fatalf("search document difficulty = %q, want %q", got, test.wantJSON)
			}
		})
	}
}
