package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteListSyncRequestsReadableTrailsWithoutCallerProjections(t *testing.T) {
	for _, projected := range []bool{false, true} {
		t.Run(map[bool]string{false: "default", true: "projection"}[projected], func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://a.example", http.StatusOK)
			author := f.remoteActor(t, "alice", "b.example")
			trails, _ := f.app.FindCollectionByNameOrId("trails")
			lists, _ := f.app.FindCollectionByNameOrId("lists")
			lists.Fields.Add(&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 100})
			if err := f.app.Save(lists); err != nil {
				t.Fatal(err)
			}
			list := f.object(t, "lists", author.Id, "https://b.example", "list00000000001", true, false)
			var requested url.Values
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					requested = request.URL.Query()
					payload := map[string]any{"id": "list00000000001", "iri": list.GetString("iri"), "name": "Public list", "public": true,
						"expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}}}
					if slices.Contains(strings.Split(requested.Get("expand"), ","), "trails") && requested.Get("fields") == "" {
						payload["expand"].(map[string]any)["trails"] = []any{map[string]any{"id": "trail0000000001", "name": "Readable trail", "public": true,
							"iri": "https://b.example/api/v1/trail/trail0000000001", "expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}}}}
					}
					wire, err := json.Marshal(payload)
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(wire))), Request: request}, err
				}}}
			}
			var requestURL *url.URL
			if projected {
				requestURL = &url.URL{RawQuery: "fields=id,name&expand=list_share_via_list.actor&foo=bar&handle=alice&share=abcdefghijklmnopqrstuvwxyz012345"}
			}
			if _, err := performFullListSync(f.app, t.Context(), requestURL, list); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(lists, list.Id)
			if err != nil || len(stored.GetStringSlice("trails")) != 1 {
				t.Fatalf("readable trail missing from synchronized list: %v %v", stored, err)
			}
			if requested.Get("fields") != "" || requested.Get("foo") != "" || requested.Get("handle") != "" || strings.Contains(requested.Get("expand"), "list_share") {
				t.Fatalf("caller projection reached the list origin: %v", requested)
			}
			for _, required := range []string{"author", "trails", "trails.author", "trails.category", "trails.subcategory", "trails.tags"} {
				if !slices.Contains(strings.Split(requested.Get("expand"), ","), required) {
					t.Fatalf("origin request omitted %s: %v", required, requested)
				}
			}
			if projected && requested.Get("share") != "abcdefghijklmnopqrstuvwxyz012345" {
				t.Fatalf("origin request lost its share: %v", requested)
			}
		})
	}
}
