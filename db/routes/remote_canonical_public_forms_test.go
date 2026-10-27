package routes

import (
	"net/http"
	"testing"
)

func TestCanonicalNotesAcceptPublicAddressFormsAndRejectPrivateReplies(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, audience := range []string{"https://www.w3.org/ns/activitystreams#Public", "as:Public", "Public", "https://third.example/users/follower", "https://another.example/Public", ""} {
		for _, field := range []string{"to", "cc"} {
			t.Run(audience+"/"+field, func(t *testing.T) {
				f := newRemoteChildFixture(t, "summit_log")
				iri := "https://third.example/notes/entry"
				endpoint, err := canonicalAPNoteURL(iri)
				if err != nil {
					t.Fatal(err)
				}
				note := map[string]any{"id": iri, "type": "Note", "content": "Public reply", "attributedTo": "https://third.example/users/carol", "inReplyTo": f.trail.GetString("iri"), field: []any{map[string]any{"id": audience}}}
				client := &http.Client{Transport: richTextSyncTransport{note}}
				value, accepted := fetchCanonicalRemoteComment(f.app, t.Context(), client, f.trail, canonicalCommentRequest{iri: iri, url: endpoint, activityPubNote: true})
				want := audience == "https://www.w3.org/ns/activitystreams#Public" || audience == "as:Public" || audience == "Public"
				if accepted != want || (accepted && value.data["text"] != "Public reply") {
					t.Fatalf("reply audience %q: accepted=%v, want%v", audience, accepted, want)
				}
			})
		}
	}
}
