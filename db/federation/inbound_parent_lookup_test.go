package federation

import (
	"fmt"
	"testing"

	pub "github.com/go-ap/activitypub"
)

func TestIncomingRepliesValidateAuthorBeforeFetchingParent(t *testing.T) {
	for _, kind := range []string{"comment", "summit-log"} {
		for _, scenario := range []string{"local-sender", "other-host", "other-author"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := setupAddressingTestApp(t)
				sender := f.record(t, "activitypub_actors", map[string]any{"iri": "https://remote.example/users/sender", "is_local": scenario == "local-sender"})
				objectIRI := "https://remote.example/api/v1/" + kind + "/entry"
				attribution := sender.GetString("iri")
				if scenario == "other-host" {
					objectIRI = "https://other.example/api/v1/" + kind + "/entry"
				}
				if scenario == "other-author" {
					attribution = "https://remote.example/users/other"
				}
				previous := fetchTrailObject
				t.Cleanup(func() { fetchTrailObject = previous })
				fetches := 0
				fetchTrailObject = func(string) (*pub.Object, error) {
					fetches++
					return nil, fmt.Errorf("parent is unavailable")
				}
				object := &pub.Object{ID: pub.IRI(objectIRI), Type: pub.NoteType, AttributedTo: pub.IRI(attribution),
					To: pub.ItemCollection{pub.PublicNS}, InReplyTo: pub.IRI("https://remote.example/api/v1/trail/missing")}
				activity := pub.Activity{Type: pub.CreateType, Object: object, Actor: pub.IRI(sender.GetString("iri"))}
				var err error
				if kind == "comment" {
					err = processCreateOrUpdateCommentActivity(activity, f.app, sender)
				} else {
					err = processCreateOrUpdateSummitLogActivity(activity, f.app, sender)
				}
				if scenario == "local-sender" && err != nil {
					t.Errorf("local sender was not ignored: %v", err)
				} else if scenario != "local-sender" && err == nil {
					t.Error("different object author was accepted")
				}
				if fetches != 0 {
					t.Errorf("invalid reply fetched its parent %d times", fetches)
				}
			})
		}
	}
}
