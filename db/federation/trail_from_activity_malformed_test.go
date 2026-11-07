package federation

import (
	"encoding/json"
	"strings"
	"testing"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// Remote objects are attacker-controlled. Every row decodes a hand-written
// document through the real inbox decoder and imports it under a recover, so a
// regression shows up as a named failing row instead of a crashed test binary.

const malformedSenderIRI = "https://remote.example/users/alice"

// setupMalformedObjectFixture adds the measure and location fields the importer
// writes, so "the other measures are still stored" is not vacuous: PocketBase
// drops values for fields the collection does not have.
func setupMalformedObjectFixture(t *testing.T) (*addressingFixture, *core.Record) {
	t.Helper()
	f := setupAddressingTestApp(t)

	trails, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trails.Fields.Add(
		&core.TextField{Name: "location"},
		&core.TextField{Name: "difficulty"},
		&core.NumberField{Name: "distance"},
		&core.NumberField{Name: "duration"},
		&core.NumberField{Name: "elevation_gain"},
		&core.NumberField{Name: "elevation_loss"},
		&core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"},
		&core.BoolField{Name: "needs_full_sync"},
	)
	if err := f.app.Save(trails); err != nil {
		t.Fatal(err)
	}

	lists, err := f.app.FindCollectionByNameOrId("lists")
	if err != nil {
		t.Fatal(err)
	}
	lists.Fields.Add(&core.BoolField{Name: "needs_full_sync"})
	if err := f.app.Save(lists); err != nil {
		t.Fatal(err)
	}

	sender := f.record(t, "activitypub_actors", map[string]any{"iri": malformedSenderIRI})
	return f, sender
}

// malformedActivity decodes the baseline Create of a valid Wanderer trail with
// the given object id after mutate patched it. mutate may change object keys or
// set activity["object"] to nil or delete it.
func malformedActivity(t *testing.T, objectID string, mutate func(activity, object map[string]any)) pub.Activity {
	t.Helper()

	object := map[string]any{
		"id":           objectID,
		"type":         "Note",
		"attributedTo": malformedSenderIRI,
		"name":         "Probe",
		"content":      "<p>x</p>",
		"startTime":    "2026-01-01T00:00:00Z",
		"location":     map[string]any{"type": "Place", "name": "Somewhere", "latitude": 1.5, "longitude": 2.5},
		"tag": []any{
			map[string]any{"type": "Note", "name": "distance", "content": "100.000000m"},
			map[string]any{"type": "Note", "name": "duration", "content": "5.000000m"},
			map[string]any{"type": "Note", "name": "elevation_gain", "content": "10.000000m"},
			map[string]any{"type": "Note", "name": "elevation_loss", "content": "10.000000m"},
		},
		"attachment": []any{},
	}
	activity := map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       "https://remote.example/activities/probe",
		"type":     "Create",
		"actor":    malformedSenderIRI,
		"object":   object,
	}
	if mutate != nil {
		mutate(activity, object)
	}

	body, err := json.Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalInboxActivity(body)
	if err != nil {
		t.Fatalf("baseline document did not decode: %v", err)
	}
	return decoded
}

func importTrailWithoutPanic(t *testing.T, f *addressingFixture, sender *core.Record, activity pub.Activity) (stored *core.Record, err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("TrailFromActivity panicked: %v", recovered)
		}
	}()
	return util.TrailFromActivity(activity, f.app, sender)
}

func importListWithoutPanic(t *testing.T, f *addressingFixture, sender *core.Record, activity pub.Activity) (stored *core.Record, err error) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("ListFromActivity panicked: %v", recovered)
		}
	}()
	return util.ListFromActivity(activity, f.app, sender)
}

// measureTags builds the baseline measure tags, patching the named measures so
// a single row can break a single tag. A nil patch value drops the content key.
func measureTags(patch map[string]any) []any {
	tags := []any{}
	for _, m := range []struct{ name, content string }{
		{"distance", "100.000000m"},
		{"duration", "5.000000m"},
		{"elevation_gain", "10.000000m"},
		{"elevation_loss", "10.000000m"},
	} {
		tag := map[string]any{"type": "Note", "name": m.name, "content": m.content}
		if override, ok := patch[m.name]; ok {
			if override == nil {
				delete(tag, "content")
			} else {
				tag["content"] = override
			}
		}
		tags = append(tags, tag)
	}
	return tags
}

type measures struct{ distance, duration, gain, loss float64 }

func requireStoredTrail(t *testing.T, stored *core.Record, want measures) {
	t.Helper()
	if stored == nil {
		t.Fatal("trail was not returned")
	}
	if got := stored.GetString("location"); got != "Somewhere" {
		t.Errorf("location = %q, want Somewhere", got)
	}
	if stored.GetFloat("lat") != 1.5 || stored.GetFloat("lon") != 2.5 {
		t.Errorf("lat/lon = %v/%v, want 1.5/2.5", stored.GetFloat("lat"), stored.GetFloat("lon"))
	}
	got := measures{stored.GetFloat("distance"), stored.GetFloat("duration"), stored.GetFloat("elevation_gain"), stored.GetFloat("elevation_loss")}
	if got != want {
		t.Errorf("measures = %+v, want %+v", got, want)
	}
}

var baselineMeasures = measures{distance: 100, duration: 5, gain: 10, loss: 10}

// malformedTrailCase is one row: wantErr empty means the trail is stored and
// check inspects it.
type malformedTrailCase struct {
	name    string
	mutate  func(activity, object map[string]any)
	wantErr string
	check   func(t *testing.T, stored *core.Record)
}

// storedMeasures is a row whose trail is stored without error, with the given
// measures and the baseline location.
func storedMeasures(name string, want measures, mutate func(activity, object map[string]any)) malformedTrailCase {
	return malformedTrailCase{
		name:   name,
		mutate: mutate,
		check: func(t *testing.T, stored *core.Record) {
			requireStoredTrail(t, stored, want)
		},
	}
}

func TestTrailFromActivityMalformedObjects(t *testing.T) {
	f, sender := setupMalformedObjectFixture(t)

	for _, tc := range []malformedTrailCase{
		{
			name: "baseline",
			check: func(t *testing.T, stored *core.Record) {
				requireStoredTrail(t, stored, baselineMeasures)
			},
		},
		{name: "location-absent", mutate: func(_, o map[string]any) { delete(o, "location") }, wantErr: "has no Place location"},
		{name: "location-null", mutate: func(_, o map[string]any) { o["location"] = nil }, wantErr: "has no Place location"},
		{name: "location-iri", mutate: func(_, o map[string]any) { o["location"] = "https://remote.example/places/1" }, wantErr: "has no Place location"},
		{
			name:    "location-note",
			mutate:  func(_, o map[string]any) { o["location"] = map[string]any{"type": "Note", "name": "Somewhere"} },
			wantErr: "has no Place location",
		},
		{
			name:    "location-array",
			mutate:  func(_, o map[string]any) { o["location"] = []any{map[string]any{"type": "Place", "name": "Somewhere"}} },
			wantErr: "has no Place location",
		},
		{name: "object-null", mutate: func(a, _ map[string]any) { a["object"] = nil }, wantErr: "activity has no trail object"},
		{name: "object-absent", mutate: func(a, _ map[string]any) { delete(a, "object") }, wantErr: "activity has no trail object"},

		// A bad measure tag is skipped: the trail is stored, that measure keeps
		// its earlier value (0 if none) and the other measures are unaffected.
		storedMeasures("distance-empty-content", measures{0, 5, 10, 10}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"distance": ""}) }),
		storedMeasures("duration-content-missing", measures{100, 0, 10, 10}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"duration": nil}) }),
		// A one-character content decodes to the empty string.
		storedMeasures("elevation_gain-one-char", measures{100, 5, 0, 10}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"elevation_gain": "5"}) }),
		storedMeasures("elevation_loss-empty", measures{100, 5, 10, 0}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"elevation_loss": ""}) }),
		storedMeasures("distance-nan", measures{0, 5, 10, 10}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"distance": "NaNm"}) }),
		storedMeasures("distance-inf", measures{0, 5, 10, 10}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"distance": "Infm"}) }),
		storedMeasures("distance-overflow", measures{0, 5, 10, 10}, func(_, o map[string]any) { o["tag"] = measureTags(map[string]any{"distance": "1e999m"}) }),
		storedMeasures("distance-valid-then-empty", baselineMeasures, func(_, o map[string]any) {
			o["tag"] = append(measureTags(nil), map[string]any{"type": "Note", "name": "distance", "content": ""})
		}),

		// An attachment without a URL is skipped; there is nothing to download.
		storedMeasures("gpx-without-url", baselineMeasures, func(_, o map[string]any) {
			o["attachment"] = []any{map[string]any{"type": "Document", "mediaType": "application/xml+gpx"}}
		}),
		storedMeasures("image-without-url", baselineMeasures, func(_, o map[string]any) {
			o["attachment"] = map[string]any{"type": "Image"}
		}),
		storedMeasures("attachment-url-empty", baselineMeasures, func(_, o map[string]any) {
			o["attachment"] = []any{map[string]any{"type": "Image", "url": ""}}
		}),

		// Shapes that never panicked stay accepted.
		storedMeasures("tag-null", measures{}, func(_, o map[string]any) { o["tag"] = nil }),
		storedMeasures("tag-absent", measures{}, func(_, o map[string]any) { delete(o, "tag") }),
		storedMeasures("tag-iri", measures{}, func(_, o map[string]any) { o["tag"] = "https://remote.example/tags/1" }),
		storedMeasures("tag-array-with-null", baselineMeasures, func(_, o map[string]any) { o["tag"] = append(measureTags(nil), nil) }),
		storedMeasures("mention-tag", baselineMeasures, func(_, o map[string]any) {
			o["tag"] = append(measureTags(nil), map[string]any{"type": "Mention", "href": "https://remote.example/users/bob", "name": "@bob"})
		}),
		storedMeasures("attachment-null-element", baselineMeasures, func(_, o map[string]any) { o["attachment"] = []any{nil} }),
		storedMeasures("attachment-iri", baselineMeasures, func(_, o map[string]any) { o["attachment"] = "https://remote.example/files/1" }),
		{
			name: "place-without-name",
			mutate: func(_, o map[string]any) {
				o["location"] = map[string]any{"type": "Place", "latitude": 1.5, "longitude": 2.5}
			},
			check: func(t *testing.T, stored *core.Record) {
				if got := stored.GetString("location"); got != "" {
					t.Errorf("location = %q, want empty", got)
				}
				if stored.GetFloat("lat") != 1.5 || stored.GetFloat("lon") != 2.5 {
					t.Errorf("lat/lon = %v/%v, want 1.5/2.5", stored.GetFloat("lat"), stored.GetFloat("lon"))
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			objectID := "https://remote.example/api/v1/trail/" + tc.name
			activity := malformedActivity(t, objectID, tc.mutate)

			stored, err := importTrailWithoutPanic(t, f, sender, activity)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tc.wantErr)
				}
				if strings.Contains(tc.wantErr, "Place location") && !strings.Contains(err.Error(), objectID) {
					t.Errorf("error %q does not name the trail IRI %s", err, objectID)
				}
				if _, findErr := f.app.FindFirstRecordByData("trails", "iri", objectID); findErr == nil {
					t.Fatal("a rejected trail was stored")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, findErr := f.app.FindFirstRecordByData("trails", "iri", objectID); findErr != nil {
				t.Fatalf("trail was not stored: %v", findErr)
			}
			tc.check(t, stored)
		})
	}

	t.Run("place-by-value", func(t *testing.T) {
		// ObjectFromTrail builds the Place by value, not as the decoded pointer.
		object := pub.Object{
			ID:           pub.IRI("https://remote.example/api/v1/trail/place-by-value"),
			Type:         pub.NoteType,
			AttributedTo: pub.IRI(malformedSenderIRI),
			Name:         pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "By value")),
			Tag:          pub.ItemCollection{},
			Location: pub.Place{
				Type:      pub.PlaceType,
				Name:      pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Somewhere")),
				Latitude:  1.5,
				Longitude: 2.5,
			},
		}
		activity := pub.Activity{Type: pub.CreateType, Object: &object, Actor: pub.IRI(malformedSenderIRI)}

		stored, err := importTrailWithoutPanic(t, f, sender, activity)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		requireStoredTrail(t, stored, measures{})
	})

	t.Run("stored-update-without-location", func(t *testing.T) {
		objectID := "https://remote.example/api/v1/trail/stored-update-without-location"
		f.record(t, "trails", map[string]any{"iri": objectID, "author": sender.Id, "name": "Original", "public": false})

		activity := malformedActivity(t, objectID, func(a, o map[string]any) {
			a["type"] = "Update"
			delete(o, "location")
		})
		if _, err := importTrailWithoutPanic(t, f, sender, activity); err != nil {
			t.Fatalf("update of a stored trail failed: %v", err)
		}

		stored, err := f.app.FindFirstRecordByData("trails", "iri", objectID)
		if err != nil {
			t.Fatal(err)
		}
		if !stored.GetBool("needs_full_sync") {
			t.Error("needs_full_sync was not set")
		}
		if stored.GetString("name") != "Original" {
			t.Errorf("name = %q, want Original", stored.GetString("name"))
		}
	})
}

// malformedListActivity decodes a baseline list Create after mutate patched it.
func malformedListActivity(t *testing.T, objectID string, mutate func(activity, object map[string]any)) pub.Activity {
	t.Helper()

	object := map[string]any{
		"id":           objectID,
		"type":         "Note",
		"attributedTo": malformedSenderIRI,
		"name":         "Probe list",
		"content":      "<p>x</p>",
		"attachment":   []any{},
	}
	activity := map[string]any{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       "https://remote.example/activities/probe-list",
		"type":     "Create",
		"actor":    malformedSenderIRI,
		"object":   object,
	}
	if mutate != nil {
		mutate(activity, object)
	}

	body, err := json.Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := UnmarshalInboxActivity(body)
	if err != nil {
		t.Fatalf("baseline document did not decode: %v", err)
	}
	return decoded
}

func TestListFromActivityMalformedObjects(t *testing.T) {
	f, sender := setupMalformedObjectFixture(t)

	for _, tc := range []struct {
		name    string
		mutate  func(activity, object map[string]any)
		wantErr string
	}{
		{name: "baseline"},
		{name: "object-null", mutate: func(a, _ map[string]any) { a["object"] = nil }, wantErr: "activity has no list object"},
		{name: "object-absent", mutate: func(a, _ map[string]any) { delete(a, "object") }, wantErr: "activity has no list object"},
		{name: "image-without-url-array", mutate: func(_, o map[string]any) {
			o["attachment"] = []any{map[string]any{"type": "Image"}}
		}},
		{name: "image-without-url-object", mutate: func(_, o map[string]any) {
			o["attachment"] = map[string]any{"type": "Image"}
		}},
		{name: "image-url-empty", mutate: func(_, o map[string]any) {
			o["attachment"] = []any{map[string]any{"type": "Image", "url": ""}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			objectID := "https://remote.example/api/v1/list/" + tc.name
			activity := malformedListActivity(t, objectID, tc.mutate)

			stored, err := importListWithoutPanic(t, f, sender, activity)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if stored == nil || stored.GetString("name") != "Probe list" {
				t.Fatalf("list was not stored: %v", stored)
			}
			if _, findErr := f.app.FindFirstRecordByData("lists", "iri", objectID); findErr != nil {
				t.Fatalf("list was not stored: %v", findErr)
			}
		})
	}
}

func TestInboxDispatchRejectsMissingObject(t *testing.T) {
	f, sender := setupMalformedObjectFixture(t)

	for _, tc := range []struct {
		name         string
		activityType string
		process      func(activity pub.Activity) error
	}{
		{"create", "Create", func(a pub.Activity) error { return ProcessCreateOrUpdateActivity(f.app, sender, nil, a) }},
		{"update", "Update", func(a pub.Activity) error { return ProcessCreateOrUpdateActivity(f.app, sender, nil, a) }},
		{"announce", "Announce", func(a pub.Activity) error { return ProcessAnnounceActivity(f.app, sender, a) }},
		{"delete", "Delete", func(a pub.Activity) error { return ProcessDeleteActivity(f.app, sender, a) }},
	} {
		for _, shape := range []string{"null", "absent"} {
			t.Run(tc.name+"-object-"+shape, func(t *testing.T) {
				activity := malformedActivity(t, "https://remote.example/api/v1/trail/dispatch", func(a, _ map[string]any) {
					a["type"] = tc.activityType
					if shape == "null" {
						a["object"] = nil
					} else {
						delete(a, "object")
					}
				})

				var err error
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							t.Fatalf("dispatch panicked: %v", recovered)
						}
					}()
					err = tc.process(activity)
				}()
				if err == nil || !strings.Contains(err.Error(), "activity has no object") {
					t.Fatalf("error = %v, want one containing %q", err, "activity has no object")
				}
			})
		}
	}
}
