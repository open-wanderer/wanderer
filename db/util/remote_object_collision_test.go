package util

import (
	"errors"
	"fmt"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func TestRemoteObjectInsertRequiresIRIUniquenessError(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, test := range []struct {
			name string
			err  error
			want bool
		}{
			{"column-validation", validation.Errors{"iri": validation.NewError("validation_not_unique", "already stored")}, true},
			{"wrapped-validation", fmt.Errorf("record: %w", validation.Errors{"iri": validation.NewError("validation_not_unique", "already stored")}), true},
			{"database-column", errors.New("UNIQUE constraint failed: " + collection + ".iri"), true},
			{"database-driver-code", errors.New("constraint failed: UNIQUE constraint failed: " + collection + ".iri (2067)"), true},
			{"other-column", validation.Errors{"name": validation.NewError("validation_not_unique", "already stored")}, false},
			{"required-column", validation.Errors{"iri": validation.ErrRequired}, false},
			{"different-column", errors.New("UNIQUE constraint failed: " + collection + ".name"), false},
			{"column-prefix", errors.New("UNIQUE constraint failed: " + collection + ".iri_extra"), false},
			{"multiple-columns", errors.New("UNIQUE constraint failed: " + collection + ".iri, " + collection + ".author"), false},
			{"other-error", errors.New("record write unavailable"), false},
			{"no-error", nil, false},
		} {
			t.Run(collection+"/"+test.name, func(t *testing.T) {
				if got := remoteObjectIRITaken(test.err, collection); got != test.want {
					t.Fatalf("retry classification = %v, want %v", got, test.want)
				}
			})
		}
	}
}
