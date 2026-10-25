package util

import (
	"slices"
	"testing"
)

func TestUniqueNonEmptyStringsPreservesOpaqueValues(t *testing.T) {
	values := []string{"", "abc", " abc", "abc ", " ", "  ", "\t", "\n", "abc", " abc", " ", "\t", ""}
	want := []string{"abc", " abc", "abc ", " ", "  ", "\t", "\n"}
	if got := UniqueNonEmptyStrings(values); !slices.Equal(got, want) {
		t.Fatalf("opaque values changed: got %q; want %q", got, want)
	}
}
