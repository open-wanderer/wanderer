package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRemoteOriginChangesKeepKnownCopiesAvailableWithoutTrustingAnAlias(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, completed := range []bool{false, true} {
			for _, public := range []bool{false, true} {
				name := collection
				if completed {
					name += "/completed"
				} else {
					name += "/placeholder"
				}
				if !public {
					name += "/private"
				}
				t.Run(name, func(t *testing.T) {
					f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
					actor := f.remoteActor(t, "bob", "registered.example")
					copy := f.object(t, collection, actor.Id, "https://changed.example", "remoteid0000001", true, false)
					copy.Set("full_sync_completed", completed)
					copy.Set("public", public)
					if err := f.app.Save(copy); err != nil {
						t.Fatal(err)
					}
					data, err := json.Marshal(map[string]any{
						"id": "remoteid0000001", "iri": copy.GetString("iri"), "name": "Unconfirmed update",
						"expand": map[string]any{"author": map[string]any{"iri": actor.GetString("iri")}},
					})
					if err != nil {
						t.Fatal(err)
					}
					f.transport.body = string(data)
					status, body := f.get(t, collection, copy.Id, "@bob@registered.example")
					want := http.StatusServiceUnavailable
					if completed && public {
						want = http.StatusOK
					} else if completed {
						want = http.StatusForbidden
					}
					if status != want || strings.Contains(body, "Unconfirmed update") {
						t.Fatalf("origin change returned unverified content or a server error: status=%d want=%d body=%s", status, want, body)
					}
					stored, err := f.app.FindRecordById(collection, copy.Id)
					if err != nil || stored.GetString("author") != actor.Id || stored.GetString("name") != "object" || stored.GetString("iri") != copy.GetString("iri") || !stored.GetBool("needs_full_sync") {
						t.Fatalf("failed binding rewrote the stored copy: record=%v error=%v", stored, err)
					}
					requests := f.transport.seen()
					if len(requests) != 1 || !strings.HasPrefix(requests[0], "https://changed.example/api/v1/") {
						t.Fatalf("copy was not fetched from its own canonical origin: %v", requests)
					}
				})
			}
		}
	}
}
