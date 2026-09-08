package main

import (
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"pocketbase/internal/srch0"
	"pocketbase/util"
	"runtime"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestSRCH0Projection(t *testing.T) {
	d := srch0.Data(t)
	app := srch0.App(t, d)
	for _, c := range srch0.Cases(t, "projection", "go-projection") {
		t.Run(c.ID, func(t *testing.T) {
			kind := c.Input["collection"].(string)
			r := srch0.Record(t, app, kind, c.Input["record_id"].(string)).Fresh()
			original := r.Fresh()
			if values, ok := c.Input["set"].(map[string]any); ok {
				srch0.Assign(t, r, values)
			}
			var photoLabels []string
			if values, ok := c.Input["set"].(map[string]any); ok {
				if photos, ok := values["photos"].([]any); ok {
					var jpegBytes bytes.Buffer
					if err := jpeg.Encode(&jpegBytes, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
						t.Fatal(err)
					}
					files := make([]*filesystem.File, len(photos))
					for i, label := range photos {
						photoLabels = append(photoLabels, label.(string))
						file, err := filesystem.NewFileFromBytes(jpegBytes.Bytes(), label.(string))
						if err != nil {
							t.Fatal(err)
						}
						files[i] = file
					}
					r.Set("photos", files)
				}
			}
			defensive := c.Input["record_boundary"] == "defensive"
			validationStatus := "accepted"
			if defensive {
				if err := app.Validate(r); err != nil {
					validationStatus = "rejected"
				}
			} else {
				// Save validates the actual production schema before projection. Restore
				// the record afterwards so every case begins with the shared source row.
				srch0.Save(t, app, r)
				for i, name := range r.GetStringSlice("photos") {
					if i < len(photoLabels) {
						srch0.AssetAlias(name, photoLabels[i])
					}
				}
				defer srch0.Save(t, app, original)
			}
			include, _ := c.Input["include_shares"].(bool)
			m := srch0.NewMeili(t)
			if c.Input["projection_diagnostic"] == true {
				status := "formed"
				func() {
					defer func() {
						if problem := recover(); problem != nil {
							status = "unexpected-panic"
							if bounds, ok := problem.(runtime.Error); ok && strings.Contains(bounds.Error(), "index out of range [-1]") {
								status = "panic"
							}
						}
					}()
					if err := util.IndexTrails(app, []*core.Record{r}, m.Client); err != nil {
						status = "error"
					}
				}()
				// This is an observed defect, not a positive product guarantee.
				// A reviewed change can replace the diagnostic with "formed"
				// without changing this original input or legitimizing the panic.
				if status == "formed" {
					t.Log("Fachliche Prüfung erfüllt: Der regulär gespeicherte Trail lässt sich projizieren.")
				} else if status == "panic" {
					t.Log("Fachlicher Fehler bestätigt: Der regulär gespeicherte Trail verursacht bei Thumbnail -1 einen negativen Arrayzugriff.")
				}
				srch0.Assert(t, c, map[string]any{"schema_validation": "accepted", "projection_status": status}, c.Observed["diagnostics"])
				return
			}
			var err error
			var calls []srch0.Request
			index := ""
			iri := r.GetString("iri")
			switch kind {
			case "trails":
				index = "trails"
				if include {
					err = util.IndexTrails(app, []*core.Record{r}, m.Client)
				} else {
					err = util.UpdateTrail(app, r, srch0.Record(t, app, "activitypub_actors", r.GetString("author")), m.Client)
				}
			case "lists":
				index = "lists"
				if remote, ok := c.Input["remote"].(map[string]any); ok {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						var body any
						json.NewDecoder(req.Body).Decode(&body)
						calls = append(calls, srch0.Request{Method: req.Method, Path: req.URL.Path, Body: body})
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(int(remote["status"].(float64)))
						json.NewEncoder(w).Encode(remote["body"])
					}))
					defer server.Close()
					r.Set("iri", server.URL+"/lists/"+r.Id)
				}
				if include {
					err = util.IndexLists(app, []*core.Record{r}, m.Client)
				} else {
					err = util.UpdateList(app, r, srch0.Record(t, app, "activitypub_actors", r.GetString("author")), m.Client)
				}
			case "activitypub_actors":
				index = "actors"
				err = util.IndexActors([]*core.Record{r}, m.Client)
			default:
				t.Fatalf("unsupported projector %s", kind)
			}
			if defensive && err != nil {
				srch0.Assert(t, c, map[string]any{"validation": validationStatus, "projection": "error", "difficulty_is_number": false}, c.Observed["diagnostics"])
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			got := m.Snapshot()[index].(map[string]any)[r.Id].(map[string]any)
			if kind == "lists" {
				got["iri"] = iri
			}
			if defensive {
				// Invalid source enums are a defensive boundary, not a commitment to the
				// arbitrary numeric fallback produced for a record that cannot be saved.
				_, numeric := got["difficulty"].(float64)
				srch0.Assert(t, c, map[string]any{"validation": validationStatus, "projection": "formed", "difficulty_is_number": numeric}, c.Observed["diagnostics"])
			} else {
				srch0.Assert(t, c, got, c.Observed["projection"])
			}
			if want, ok := c.Observed["remote_requests"]; ok {
				srch0.Assert(t, c, calls, want)
			}
			if profile, ok := c.Input["profile"].(string); ok {
				var p map[string]any
				srch0.Read(t, profile, &p)
				for _, f := range p["document_fields"].([]any) {
					if _, ok := got[f.(string)]; !ok {
						t.Errorf("profile field %s missing", f)
					}
				}
			}
		})
	}
}
