package routes

import (
	"testing"
	"time"
)

func TestRemoteSyncMatchesAuthorsToTheirOrigin(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, target := range []string{"summit_log", "trail"} {
		for _, expanded := range []bool{false, true} {
			t.Run(target+"/"+map[bool]string{false: "fallback", true: "expanded"}[expanded], func(t *testing.T) {
				f := newRemoteSyncFixture(t)
				third := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
				parentAuthor := third.Id
				raw := map[string]any{"id": "remote-record", "name": "Remote record", "text": "Remote record"}
				if expanded {
					parentAuthor = f.remoteActor.Id
					raw["expand"] = map[string]any{"author": map[string]any{"iri": third.GetString("iri")}}
				}
				collection := "trails"
				if target == "summit_log" {
					collection = "summit_logs"
					trail := f.save(f.trails, map[string]any{"author": parentAuthor})
					if err := syncSummitLogs(f.app, t.Context(), trail, "https://remote.example", []any{raw}); err != nil {
						t.Fatal(err)
					}
				} else {
					list := f.save(f.lists, map[string]any{"author": parentAuthor})
					if err := syncTrails(f.app, t.Context(), list, "https://remote.example", []any{raw}); err != nil {
						t.Fatal(err)
					}
				}
				count, err := f.app.CountRecords(collection)
				if err != nil || count != 0 {
					t.Fatalf("third-party author accepted: count=%d error=%v", count, err)
				}
			})
		}
	}
}
