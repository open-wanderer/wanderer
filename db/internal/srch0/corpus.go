// Package srch0 contains test adapters for the shared, synthetic search corpus.
// It is imported by tests only and does not participate in the application.
package srch0

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"

	"github.com/pocketbase/pocketbase/core"
)

type Object = map[string]any

type Case struct {
	SchemaVersion  string   `json:"schema_version"`
	ID             string   `json:"case_id"`
	Family         string   `json:"family"`
	Input          Object   `json:"input"`
	Observed       Object   `json:"observed"`
	Context        Object   `json:"context"`
	Baseline       Object   `json:"baseline"`
	Digest         string   `json:"-"`
	DatasetRef     string   `json:"dataset_ref"`
	ManifestDigest string   `json:"-"`
	ChangesDigest  string   `json:"-"`
	Path           string   `json:"-"`
	Stability      string   `json:"stability"`
	SuccessorRefs  []string `json:"successor_refs"`
	Evidence       Object   `json:"evidence"`
}

type FileDigest struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Baseline      Object            `json:"baseline"`
	ChangesSHA256 string            `json:"changes_sha256"`
	Cases         map[string]string `json:"cases"`
	Groups        []FileDigest      `json:"groups"`
	Datasets      []FileDigest      `json:"datasets"`
}

func Root() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "../../../testdata/trail-search/srch0/v1")
}

func Read(t TestingT, name string, value any) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(Root(), name))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, value); err != nil {
		t.Fatal(err)
	}
	return b
}

func Cases(t TestingT, family, adapter string) []Case {
	t.Helper()
	var manifest Manifest
	mb := Read(t, "manifest.json", &manifest)
	manifestDigest := fmt.Sprintf("%x", sha256.Sum256(mb))
	var changes struct {
		SchemaVersion string   `json:"schema_version"`
		Changes       []Change `json:"changes"`
	}
	cb := Read(t, "changes.json", &changes)
	if changes.SchemaVersion != "wanderer.srch0.changes/v1" {
		t.Fatal("changes.json: unbekanntes Änderungsformat")
	}
	if fmt.Sprintf("%x", sha256.Sum256(cb)) != manifest.ChangesSHA256 {
		t.Fatal("Änderungsdigest unterscheidet sich vom Manifest")
	}
	cases, err := casesGrouped(manifest, func(name string) ([]byte, error) {
		return os.ReadFile(filepath.Join(Root(), name))
	})
	if err != nil {
		t.Fatal(err)
	}
	result := []Case{}
	for _, c := range cases {
		c.ManifestDigest = manifestDigest
		c.ChangesDigest = manifest.ChangesSHA256
		if c.Family != family || c.Input["adapter"] != adapter {
			continue
		}
		if c.DatasetRef != "datasets/reference.json" {
			t.Fatalf("case=%s: Go-Adapter erwartet datasets/reference.json", c.ID)
		}
		resolved, err := ResolveObservation(c, changes.Changes)
		if err != nil {
			t.Fatal(err)
		}
		c.Observed = resolved
		result = append(result, c)
	}
	if len(result) == 0 {
		t.Fatalf("no %s cases for %s", family, adapter)
	}
	return result
}

// casesGrouped prüft die gespeicherten Gruppenbytes und expandiert gemeinsame
// Vorgaben. Den kanonischen Falldigest berechnet ausschliesslich der Node-Validator;
// Go übernimmt ihn aus dem Manifest, dessen Gruppenbindung hier geprüft wird.
func casesGrouped(manifest Manifest, read func(string) ([]byte, error)) ([]Case, error) {
	result := []Case{}
	seen := map[string]bool{}
	for _, entry := range manifest.Groups {
		data, err := read(entry.Path)
		if err != nil {
			return nil, err
		}
		if fmt.Sprintf("%x", sha256.Sum256(data)) != entry.SHA256 {
			return nil, fmt.Errorf("%s: Gruppendigest unterscheidet sich vom Manifest", entry.Path)
		}
		var group struct {
			Family     string   `json:"family"`
			DatasetRef string   `json:"dataset_ref"`
			Context    Object   `json:"context"`
			Code       []Object `json:"code"`
			Cases      []Case   `json:"cases"`
		}
		if err := json.Unmarshal(data, &group); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Path, err)
		}
		for _, c := range group.Cases {
			if seen[c.ID] {
				return nil, fmt.Errorf("%s: mehrfach in Gruppen vorhanden", c.ID)
			}
			c.Digest = manifest.Cases[c.ID]
			if c.Digest == "" {
				return nil, fmt.Errorf("%s: kanonischer Falldigest fehlt im Manifest", c.ID)
			}
			seen[c.ID] = true
			c.SchemaVersion = "wanderer.srch0/v1"
			c.Baseline = manifest.Baseline
			c.Family = group.Family
			c.DatasetRef = group.DatasetRef
			c.Path = entry.Path
			context := Object{}
			maps.Copy(context, group.Context)
			maps.Copy(context, c.Context)
			c.Context = context
			if c.SuccessorRefs == nil {
				c.SuccessorRefs = []string{}
			}
			if c.Evidence == nil {
				c.Evidence = Object{}
			}
			if _, override := c.Evidence["code"]; !override {
				c.Evidence["code"] = group.Code
			}
			result = append(result, c)
		}
	}
	for id := range manifest.Cases {
		if !seen[id] {
			return nil, fmt.Errorf("%s: Manifestfall fehlt in den Gruppen", id)
		}
	}
	return result, nil
}

func Assert(t TestingT, c Case, got, want any) {
	t.Helper()
	gb, _ := json.Marshal(got)
	wb, _ := json.Marshal(want)
	var g, w any
	json.Unmarshal(gb, &g)
	json.Unmarshal(wb, &w)
	g = Readable(g)
	w = Readable(w)
	if diff := firstDiff("$", g, w); diff != "" {
		t.Fatalf("case=%s family=%s basis_sha256=%s manifest_sha256=%s changes_sha256=%s engine_profiles=%v: %s", c.ID, c.Family, c.Digest, c.ManifestDigest, c.ChangesDigest, c.Baseline["engine_profiles"], diff)
	}
	t.Logf("case=%s family=%s basis_sha256=%s manifest_sha256=%s changes_sha256=%s engine_profiles=%v", c.ID, c.Family, c.Digest, c.ManifestDigest, c.ChangesDigest, c.Baseline["engine_profiles"])
}

func firstDiff(path string, got, want any) string {
	if reflect.DeepEqual(got, want) {
		return ""
	}
	gm, gok := got.(map[string]any)
	wm, wok := want.(map[string]any)
	if gok && wok {
		keys := []string{}
		for k := range gm {
			keys = append(keys, k)
		}
		for k := range wm {
			if _, ok := gm[k]; !ok {
				keys = append(keys, k)
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			_, gok := gm[k]
			_, wok := wm[k]
			if gok != wok {
				return fmt.Sprintf("%s.%s: field present=%t; observed present=%t", path, k, gok, wok)
			}
			if d := firstDiff(path+"."+k, gm[k], wm[k]); d != "" {
				return d
			}
		}
	}
	ga, gok := got.([]any)
	wa, wok := want.([]any)
	if gok && wok && len(ga) == len(wa) {
		for i := range ga {
			if d := firstDiff(fmt.Sprintf("%s[%d]", path, i), ga[i], wa[i]); d != "" {
				return d
			}
		}
	}
	return fmt.Sprintf("%s: got %v; observed %v", path, got, want)
}

type Dataset struct {
	Records map[string][]Object `json:"pb_records"`
	Trails  []Object            `json:"trails"`
	Lists   []Object            `json:"lists"`
	Actors  []Object            `json:"actors"`
}

func Data(t TestingT) Dataset {
	t.Helper()
	var d Dataset
	b := Read(t, "datasets/reference.json", &d)
	digest := fmt.Sprintf("%x", sha256.Sum256(b))
	var manifest Manifest
	mb := Read(t, "manifest.json", &manifest)
	bound := false
	for _, entry := range manifest.Datasets {
		if entry.Path == "datasets/reference.json" {
			bound = true
			if digest != entry.SHA256 {
				t.Fatal("shared dataset digest differs from manifest")
			}
		}
	}
	if !bound {
		t.Fatal("shared dataset missing from manifest")
	}
	t.Logf("dataset=datasets/reference.json dataset_sha256=%s manifest_sha256=%x", digest, sha256.Sum256(mb))
	return d
}

func Record(t TestingT, app core.App, collection, id string) *core.Record {
	t.Helper()
	r, err := app.FindRecordById(collection, ID(id))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func Save(t TestingT, app core.App, r *core.Record) {
	t.Helper()
	if err := app.Save(r); err != nil {
		t.Fatal(err)
	}
}
