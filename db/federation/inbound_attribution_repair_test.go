package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestInboundCanonicalSummitAuthorRepairsOnlyLegacyTrailAuthorFallback(t *testing.T) {
	for _, scenario := range []string{"legacy-fallback", "same-host-parent", "local-owner", "another-owner", "different-parent"} {
		t.Run(scenario, func(t *testing.T) {
			f := setupAddressingTestApp(t)
			trailAuthor := f.record(t, "activitypub_actors", map[string]any{"iri": "https://trail.example/users/alice"})
			if scenario == "same-host-parent" {
				trailAuthor.Set("iri", "https://third.example/users/alice")
				if err := f.app.Save(trailAuthor); err != nil {
					t.Fatal(err)
				}
			}
			contributor := f.record(t, "activitypub_actors", map[string]any{"iri": "https://third.example/users/carol"})
			other := f.record(t, "activitypub_actors", map[string]any{"iri": "https://third.example/users/dave", "is_local": scenario == "local-owner"})
			parent := f.record(t, "trails", map[string]any{"iri": "https://trail.example/api/v1/trail/parent", "author": trailAuthor.Id, "public": true})
			storedAuthor := trailAuthor.Id
			if scenario == "local-owner" || scenario == "another-owner" {
				storedAuthor = other.Id
			}
			storedTrail := parent.Id
			if scenario == "different-parent" {
				storedTrail = f.record(t, "trails", map[string]any{"iri": "https://trail.example/api/v1/trail/other", "author": trailAuthor.Id, "public": true}).Id
			}
			log := f.record(t, "summit_logs", map[string]any{"iri": "https://third.example/api/v1/summit-log/original", "author": storedAuthor, "trail": storedTrail, "text": "Original"})
			object := pub.Object{ID: pub.IRI(log.GetString("iri")), Type: pub.NoteType, InReplyTo: pub.IRI(parent.GetString("iri")), AttributedTo: pub.IRI(contributor.GetString("iri")), Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Updated")), Tag: pub.ItemCollection{}}
			activity := pub.Activity{Type: pub.UpdateType, Actor: pub.IRI(contributor.GetString("iri")), Object: &object}
			err := processCreateOrUpdateSummitLogActivity(activity, f.app, contributor)
			stored, lookupErr := f.app.FindRecordById("summit_logs", log.Id)
			if lookupErr != nil {
				t.Fatal(lookupErr)
			}
			if scenario == "legacy-fallback" {
				if err != nil || stored.GetString("author") != contributor.Id || stored.GetString("text") != "Updated" {
					t.Fatalf("canonical repair failed: %v %v", stored, err)
				}
			} else if err == nil || stored.GetString("author") != storedAuthor || stored.GetString("text") != "Original" {
				t.Fatalf("non-legacy ownership changed: %v %v", stored, err)
			}
		})
	}
}
