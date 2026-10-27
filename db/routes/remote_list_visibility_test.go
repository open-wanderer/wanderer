package routes

import (
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestPublicListRechecksKnownShareCopyAtItsCanonicalOrigin(t *testing.T) {
	for _, state := range []string{"public", "private", "unavailable", "wrong-object", "wrong-author", "local-owner", "historical-local"} {
		t.Run(state, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
			listAuthor := f.remoteActor(t, "alice", "remote.example")
			trailAuthor := f.remoteActor(t, "bob", "third.example")
			if state == "local-owner" {
				trailAuthor.Set("is_local", true)
				if err := f.app.Save(trailAuthor); err != nil {
					t.Fatal(err)
				}
			}
			trails, _ := f.app.FindCollectionByNameOrId("trails")
			lists, _ := f.app.FindCollectionByNameOrId("lists")
			lists.Fields.Add(&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 100})
			if err := f.app.Save(lists); err != nil {
				t.Fatal(err)
			}
			trail := f.object(t, "trails", trailAuthor.Id, "https://third.example", "trail0000000001", false, false)
			trail.Set("public", false)
			if state == "historical-local" {
				trail.Set("iri", "https://third.example/api/v1/trail/"+trail.Id)
			}
			if err := f.app.Save(trail); err != nil {
				t.Fatal(err)
			}
			list := f.object(t, "lists", listAuthor.Id, "https://remote.example", "list00000000001", true, false)
			var probes, probesInsideTX atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host != "third.example" {
						return richTextSyncTransport{map[string]any{"id": "list00000000001", "iri": list.GetString("iri"), "name": "Public list", "public": true,
							"expand": map[string]any{"trails": []any{map[string]any{"id": "trail0000000001", "iri": trail.GetString("iri"), "public": true}}}}}.RoundTrip(request)
					}
					probes.Add(1)
					if f.app.TxInfo() != nil {
						probesInsideTX.Add(1)
					}
					payload := map[string]any{"id": "trail0000000001", "iri": trail.GetString("iri"), "public": state != "private", "expand": map[string]any{"author": map[string]any{"iri": trailAuthor.GetString("iri")}}}
					if state == "wrong-object" {
						payload["iri"] = "https://third.example/api/v1/trail/other"
					}
					if state == "wrong-author" {
						payload["expand"] = map[string]any{"author": map[string]any{"iri": "https://another.example/users/bob"}}
					}
					response, err := richTextSyncTransport{payload}.RoundTrip(request)
					if state == "unavailable" {
						response.StatusCode = http.StatusServiceUnavailable
					}
					return response, err
				}}}
			}
			status, body := f.get(t, "lists", list.Id, "@alice@remote.example")
			if status != http.StatusOK {
				t.Fatalf("list failed: %d %s", status, body)
			}
			storedList, _ := f.app.FindRecordById("lists", list.Id)
			storedTrail, _ := f.app.FindRecordById("trails", trail.Id)
			want := state == "public"
			if storedTrail.GetBool("public") != want || (len(storedList.GetStringSlice("trails")) == 1) != want {
				t.Fatalf("share-copy visibility/list relation: public=%v linked=%v, want %v", storedTrail.GetBool("public"), storedList.GetStringSlice("trails"), want)
			}
			wantProbes := int32(1)
			if state == "local-owner" || state == "historical-local" {
				wantProbes = 0
			}
			if probes.Load() != wantProbes || probesInsideTX.Load() != 0 {
				t.Fatalf("canonical visibility probes=%d insideTX=%d", probes.Load(), probesInsideTX.Load())
			}
		})
	}
}
