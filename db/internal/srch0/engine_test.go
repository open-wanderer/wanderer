package srch0

import (
	"math"
	"testing"
)

func TestSRCH0MaterializationAllowsOnlyNumericRoundtrip(t *testing.T) {
	document := func(distance float64) Object {
		return Object{"lists": Object{"list-local": Object{"distance": distance, "trails": float64(2), "shares": []any{"alice"}}}}
	}
	want := document(20.490000000000002)
	got := document(20.49)
	if diff := materializationDiff("$", got, want); diff != "" {
		t.Fatalf("machine precision roundtrip rejected: %s", diff)
	}
	if firstDiff("$", got, want) == "" {
		t.Fatal("the ordinary golden comparator must remain exact")
	}
	for _, scenario := range []struct {
		name string
		edit func(Object)
	}{
		{"changed-distance", func(d Object) { d["distance"] = 20.5 }},
		{"changed-integer", func(d Object) { d["trails"] = math.Nextafter(2, 3) }},
		{"changed-type", func(d Object) { d["distance"] = "20.49" }},
		{"missing-field", func(d Object) { delete(d, "distance") }},
		{"extra-field", func(d Object) { d["fabricated"] = true }},
		{"changed-share", func(d Object) { d["shares"] = []any{"bob"} }},
		{"extra-share", func(d Object) { d["shares"] = []any{"alice", "bob"} }},
		{"infinite", func(d Object) { d["distance"] = math.Inf(1) }},
		{"nan", func(d Object) { d["distance"] = math.NaN() }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			got := document(20.49)
			scenario.edit(got["lists"].(Object)["list-local"].(Object))
			if materializationDiff("$", got, want) == "" {
				t.Fatal("a real field change passed the materialization comparison")
			}
		})
	}
}
