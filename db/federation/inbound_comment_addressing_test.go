package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestInboundCommentsRequirePublicAddressingAndSupportLegacyActivities(t *testing.T) {
	private := pub.ItemCollection{pub.IRI("https://remote.example/users/author")}
	followers := pub.ItemCollection{pub.IRI("https://remote.example/users/author/followers")}
	for _, typ := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
		for _, tc := range []struct {
			name                           string
			to, cc, activityTo, activityCC pub.ItemCollection
			public                         bool
		}{
			{"public-note", pub.ItemCollection{pub.PublicNS}, nil, nil, nil, true},
			{"compact-cc", private, pub.ItemCollection{pub.IRI("as:Public")}, nil, nil, true},
			{"compact-public", pub.ItemCollection{pub.IRI("Public")}, nil, nil, nil, true},
			{"legacy-public-activity", nil, nil, pub.ItemCollection{pub.PublicNS}, nil, true},
			{"per-field-legacy-cc", private, nil, private, pub.ItemCollection{pub.PublicNS}, true},
			{"direct", private, pub.ItemCollection{}, private, nil, false},
			{"followers-only", followers, nil, followers, nil, false},
			{"private-note-on-public-activity", private, pub.ItemCollection{}, pub.ItemCollection{pub.PublicNS}, pub.ItemCollection{pub.PublicNS}, false},
			{"explicit-empty-note", pub.ItemCollection{}, pub.ItemCollection{}, pub.ItemCollection{pub.PublicNS}, nil, false},
		} {
			t.Run(string(typ)+"/"+tc.name, func(t *testing.T) {
				f := setupAddressingTestApp(t)
				author := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/author"})
				trail := f.record(t, "trails", map[string]any{"iri": "https://remote.example/api/v1/trail/parent", "author": author.Id, "public": true})
				iri := "https://remote.example/api/v1/comment/reply"
				if typ == pub.UpdateType {
					f.record(t, "comments", map[string]any{"iri": iri, "author": author.Id, "trail": trail.Id, "text": "Original public reply"})
				}
				object := pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, AttributedTo: pub.IRI(author.GetString("iri")), InReplyTo: pub.IRI(trail.GetString("iri")), To: tc.to, CC: tc.cc, Content: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Updated reply")), Tag: pub.ItemCollection{}}
				activity := pub.ActivityNew(pub.IRI("https://remote.example/activity/reply"), typ, &object)
				activity.Actor = pub.IRI(author.GetString("iri"))
				activity.To = tc.activityTo
				activity.CC = tc.activityCC
				err := ProcessCreateOrUpdateActivity(f.app, author, nil, *activity)
				record, findErr := f.app.FindFirstRecordByData("comments", "iri", iri)
				if tc.public {
					if err != nil || findErr != nil || record.GetString("text") != "Updated reply" {
						t.Fatalf("public/legacy reply rejected: err=%v find=%v record=%v", err, findErr, record)
					}
				} else {
					if err == nil {
						t.Fatal("non-public comment accepted")
					}
					if typ == pub.CreateType && record != nil {
						t.Fatal("non-public comment persisted")
					}
					if typ == pub.UpdateType && (findErr != nil || record.GetString("text") != "Original public reply") {
						t.Fatal("non-public update changed existing public reply")
					}
				}
			})
		}
	}
}
