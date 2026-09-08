package srch0

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestSRCH0ResolveObservation(t *testing.T) {
	base := Case{ID: "SRCH0-TEST-001", Digest: "basis", Observed: Object{"result": Object{"total": float64(1), "ids": []any{"a", "b"}}, "removed": true}}
	want := Object{"result": Object{"total": float64(2), "ids": []any{"b", "c"}}, "added": nil}
	change := Change{ID: "approved-change", CaseID: base.ID, BasisDigest: base.Digest, Observed: want}
	got, err := ResolveObservation(base, []Change{change})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Vollständiger Ersatz fehlt: got %#v", got)
	}
	got["result"].(Object)["ids"].([]any)[0] = "changed"
	got["result"].(Object)["total"] = float64(99)
	if want["result"].(Object)["ids"].([]any)[0] != "b" || want["result"].(Object)["total"] != float64(2) {
		t.Fatal("Änderungsbeobachtung mutiert")
	}
	if base.Observed["result"].(Object)["total"] != float64(1) || base.Observed["removed"] != true || base.Digest != "basis" {
		t.Fatal("Basisbeobachtung oder Basisdigest mutiert")
	}

	// Änderungen anderer Fälle dürfen die Basis weder ersetzen noch teilen.
	change.CaseID = "SRCH0-TEST-002"
	got, err = ResolveObservation(base, []Change{change})
	if err != nil || !reflect.DeepEqual(got, base.Observed) {
		t.Fatalf("Unveränderte Basis fehlt: got %#v, error %v", got, err)
	}
	got["result"].(Object)["ids"].([]any)[0] = "changed"
	if base.Observed["result"].(Object)["ids"].([]any)[0] != "a" {
		t.Fatal("Basisbeobachtung wird ohne aktive Änderung geteilt")
	}
}

func TestSRCH0ResolveObservationRejectsInvalidChanges(t *testing.T) {
	base := Case{ID: "case", Digest: "basis", Observed: Object{"value": nil}}
	change := Change{CaseID: base.ID, BasisDigest: base.Digest, Observed: Object{"value": true}}
	for _, test := range []struct {
		name    string
		changes []Change
		message string
	}{
		{"anderer Basisdigest", []Change{{CaseID: base.ID, BasisDigest: "stale", Observed: change.Observed}}, "Basisdigest"},
		{"mehrere Änderungen", []Change{change, change}, "mehrere aktive"},
		{"fehlende Beobachtung", []Change{{CaseID: base.ID, BasisDigest: base.Digest}}, "keine Beobachtung"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ResolveObservation(base, test.changes); err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Ungültige Änderung nicht gezielt abgewiesen: %v", err)
			}
		})
	}
	for _, raw := range []string{`{"observed":[]}`, `{"observed":1}`, `{"observed":"value"}`} {
		var invalid Change
		if err := json.Unmarshal([]byte(raw), &invalid); err == nil {
			t.Fatalf("Beobachtung ohne Objekt akzeptiert: %s", raw)
		}
	}
}
