package routes

import (
	"net/http"
	"net/url"
	"testing"
)

func TestCanonicalContributionsAcceptLegacyIdentityOnlyAtTheirExactEndpoint(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"summit-log", "comment"} {
		for _, state := range []string{"legacy", "modern", "wrong-id", "wrong-object", "wrong-author", "wrong-parent", "query", "encoded-path"} {
			t.Run(kind+"/"+state, func(t *testing.T) {
				f := newRemoteChildFixture(t, "summit_log")
				iri := "https://third.example/api/v1/" + kind + "/entry"
				target, _ := url.Parse(iri)
				data := map[string]any{"id": "entry", "iri": "", "text": "Authoritative legacy text", "expand": map[string]any{
					"author": map[string]any{"iri": "https://third.example/users/carol"}, "trail": map[string]any{"iri": f.trail.GetString("iri")}}}
				switch state {
				case "modern":
					data["iri"] = iri
				case "wrong-id":
					data["id"] = "other"
				case "wrong-object":
					data["iri"] = "https://third.example/api/v1/" + kind + "/other"
				case "wrong-author":
					data["expand"].(map[string]any)["author"] = map[string]any{"iri": "https://another.example/users/carol"}
				case "wrong-parent":
					data["expand"].(map[string]any)["trail"] = map[string]any{"iri": "https://remote.example/api/v1/trail/other"}
				case "query":
					iri += "?caller=provided"
				case "encoded-path":
					iri = "https://third.example/api/v1/" + kind + "/%65ntry"
				}
				client := &http.Client{Transport: richTextSyncTransport{data}}
				accepted, text := false, ""
				if kind == "comment" {
					value, ok := fetchCanonicalRemoteComment(f.app, t.Context(), client, f.trail, canonicalCommentRequest{iri: iri, url: target})
					accepted = ok
					if ok {
						text, _ = value.data["text"].(string)
					}
				} else {
					value, ok := fetchCanonicalRemoteSummitLog(f.app, t.Context(), client, f.trail, canonicalLogRequest{iri: iri, url: target})
					accepted = ok
					if ok {
						text, _ = value.data["text"].(string)
					}
				}
				want := state == "legacy" || state == "modern"
				if accepted != want || (accepted && text != "Authoritative legacy text") {
					t.Fatalf("canonical %s accepted=%v text=%q, want %v", kind, accepted, text, want)
				}
			})
		}
	}
}
