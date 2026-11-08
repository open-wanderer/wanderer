package routes

import (
	"net/url"
	"strings"
	"testing"
)

func TestRemoteSyncQueriesFilterShareValues(t *testing.T) {
	valid := "abcdefghijklmnopqrstuvwxyz012345"
	for _, target := range []struct {
		name  string
		query func(*url.URL) url.Values
	}{
		{"trail", remoteTrailSyncQuery},
		{"list", remoteListSyncQuery},
	} {
		for _, scenario := range []struct {
			name string
			raw  string
			want string
		}{
			{"valid", valid, valid},
			{"invalid-character", strings.Repeat("a", 31) + "-", ""},
			{"oversized", strings.Repeat("a", 33), ""},
			{"uppercase", strings.Repeat("A", 32), ""},
			{"encoded-delimiter", strings.Repeat("a", 31) + "%26", ""},
			{"encoded-space", strings.Repeat("a", 31) + "%20", ""},
			{"encoded-query", url.QueryEscape(valid + "&fields=*"), ""},
			{"encoded-percent", url.QueryEscape("%61" + strings.Repeat("a", 29)), ""},
		} {
			t.Run(target.name+"/"+scenario.name, func(t *testing.T) {
				query := target.query(&url.URL{RawQuery: "share=" + scenario.raw})
				if query.Get("share") != scenario.want {
					t.Fatalf("forwarded share=%q, want %q", query.Get("share"), scenario.want)
				}
				if scenario.want == "" && query.Has("share") {
					t.Fatalf("invalid share parameter reached the origin: %v", query)
				}
			})
		}
	}
}

func TestRemoteSyncQueriesCannotBeInflatedByCallerParameters(t *testing.T) {
	request, err := url.Parse("https://a.example/trail?expand=" + strings.Repeat("caller.", 10_000) + "&padding=" + strings.Repeat("x", 50_000) + "&share=" + strings.Repeat("a", 10_000))
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []url.Values{remoteTrailSyncQuery(request), remoteListSyncQuery(request)} {
		if len(query.Encode()) > 1024 || query.Get("padding") != "" || query.Get("share") != "" || strings.Contains(query.Get("expand"), "caller") {
			t.Fatalf("caller input reached the sync origin: %d encoded bytes", len(query.Encode()))
		}
	}
}
