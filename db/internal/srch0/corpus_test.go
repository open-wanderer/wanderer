package srch0

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"testing"
)

const testGroup = `{
  "family": "projection",
  "dataset_ref": "datasets/reference.json",
  "context": {"principal": "anonymous", "locale": "en", "preferences": {"unit": "metric", "distance": 7}},
  "code": [{"path": "db/util/meilisearch.go", "symbol": "IndexTrails"}],
  "cases": [
    {"case_id": "SRCH0-TEST-001", "input": {"adapter": "go-projection"}, "observed": {"projection": {"id": "first"}}, "stability": "preserve", "evidence": {"coverage": ["positive"]}},
    {"case_id": "SRCH0-TEST-002", "input": {"adapter": "go-projection"}, "observed": {"diagnostics": {"status": "known"}}, "stability": "known_gap", "successor_refs": ["IDX1"], "context": {"principal": "alice", "preferences": {"unit": "imperial"}}, "evidence": {"coverage": ["boundary"], "method": "Eigene Probe", "code": [{"path": "db/main.go", "symbol": "initDataWithBackground"}]}}
  ]
}`

func groupManifest() Manifest {
	return Manifest{
		Baseline: Object{"commit": "baseline"},
		Groups:   []FileDigest{{Path: "cases/projection/trails.json", SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte(testGroup)))}},
		// Absichtlich keine lokal berechneten Falldigests: deren kanonische
		// Berechnung gehört dem Node-Validator, Go übernimmt die Manifestwerte.
		Cases: map[string]string{"SRCH0-TEST-001": "canonical-first", "SRCH0-TEST-002": "canonical-second"},
	}
}

func TestSRCH0GroupedCases(t *testing.T) {
	manifest := groupManifest()
	cases, err := casesGrouped(manifest, func(path string) ([]byte, error) {
		if path != manifest.Groups[0].Path {
			t.Fatalf("Unbekannter Gruppenpfad: %s", path)
		}
		return []byte(testGroup), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 2 {
		t.Fatalf("Gruppenfälle fehlen: %d", len(cases))
	}
	first, second := cases[0], cases[1]
	for _, c := range cases {
		if c.SchemaVersion != "wanderer.srch0/v1" || c.Family != "projection" || c.DatasetRef != "datasets/reference.json" || c.Path != manifest.Groups[0].Path || !reflect.DeepEqual(c.Baseline, manifest.Baseline) || c.Digest != manifest.Cases[c.ID] {
			t.Fatalf("Gemeinsame Vorgaben oder Manifestbindung fehlen: %#v", c)
		}
	}
	wantContext := Object{"principal": "alice", "locale": "en", "preferences": Object{"unit": "imperial"}}
	if !reflect.DeepEqual(second.Context, wantContext) || first.Context["principal"] != "anonymous" {
		t.Fatalf("Kontext muss flach zusammengeführt werden: %#v", second.Context)
	}
	if first.SuccessorRefs == nil || len(first.SuccessorRefs) != 0 || !reflect.DeepEqual(second.SuccessorRefs, []string{"IDX1"}) {
		t.Fatal("Nachfolgereferenzen fehlen oder haben keinen leeren Standardwert")
	}
	if first.Stability != "preserve" || second.Stability != "known_gap" || first.Observed["projection"].(Object)["id"] != "first" || second.Input["adapter"] != "go-projection" {
		t.Fatal("Fallspezifische Werte wurden bei der Expansion verändert")
	}
	if first.Evidence["code"].([]Object)[0]["symbol"] != "IndexTrails" || first.Evidence["coverage"].([]any)[0] != "positive" {
		t.Fatal("Gemeinsame Codebelege oder fallspezifische Abdeckung fehlen")
	}
	if second.Evidence["code"].([]any)[0].(Object)["symbol"] != "initDataWithBackground" || second.Evidence["method"] != "Eigene Probe" {
		t.Fatal("Fallspezifische Codebelege und Methode fehlen")
	}
}

func TestSRCH0GroupedCasesRejectBrokenBindings(t *testing.T) {
	for _, test := range []struct {
		name    string
		prepare func(*Manifest)
		data    string
		message string
	}{
		{"geänderte Gruppenbytes", func(*Manifest) {}, testGroup + "\n", "Gruppendigest"},
		{"Fall fehlt im Manifest", func(m *Manifest) { delete(m.Cases, "SRCH0-TEST-002") }, testGroup, "Falldigest fehlt"},
		{"Fall doppelt vorhanden", func(m *Manifest) { m.Groups = append(m.Groups, m.Groups[0]) }, testGroup, "mehrfach"},
		{"Manifestfall fehlt", func(m *Manifest) { m.Cases["SRCH0-TEST-003"] = "canonical-third" }, testGroup, "Manifestfall fehlt"},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := groupManifest()
			manifest.Cases = maps.Clone(manifest.Cases)
			test.prepare(&manifest)
			_, err := casesGrouped(manifest, func(string) ([]byte, error) { return []byte(test.data), nil })
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Defekte Bindung nicht gezielt abgewiesen: %v", err)
			}
		})
	}
}
