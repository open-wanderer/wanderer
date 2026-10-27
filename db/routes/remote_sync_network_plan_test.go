package routes

import (
	"net/http"
	"net/url"
	"testing"
)

func TestRemoteSyncNetworkPlanPreservesTrailVisibility(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, tc := range []struct {
		name, kind string
		public     any
		want       bool
	}{
		{"public-trail", "trail", true, true},
		{"private-trail", "trail", false, true},
		{"string-trail", "trail", "true", false},
		{"omitted-trail", "trail", nil, false},
		{"public-list", "list", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			col := f.trails
			if tc.kind == "list" {
				col = f.lists
			}
			iri := "https://remote.example/api/v1/" + tc.kind + "/copy"
			record := f.save(col, map[string]any{"iri": iri, "author": f.remoteActor.Id})
			data := map[string]any{"id": "copy", "iri": iri, "expand": map[string]any{}}
			if tc.public != nil {
				data["public"] = tc.public
			}
			remoteURL, err := url.Parse(iri)
			if err != nil {
				t.Fatal(err)
			}
			document, err := fetchRemoteSyncDocument(t.Context(), &http.Client{Transport: richTextSyncTransport{data}}, remoteURL, tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			defer document.close()
			plan, err := document.networkPlan(f.app, t.Context(), record, "https://remote.example", tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			public, present := plan.data["public"]
			if present != tc.want || present && public != tc.public {
				t.Fatalf("projected visibility = %v, present=%t; want %v, present=%t", public, present, tc.public, tc.want)
			}
		})
	}
}
