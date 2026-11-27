package routes

import (
	"fmt"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteChildrenRequireRecordIDs(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		for _, id := range []any{nil, "", "   ", 7} {
			t.Run(target+"/"+fmt.Sprint(id), func(t *testing.T) {
				f := newRemoteChildFixture(t, target)
				f.remoteActor.Set("iri", "https://a.example/users/alice")
				if err := f.app.Save(f.remoteActor); err != nil {
					t.Fatal(err)
				}
				raw := map[string]any{"id": id, "name": "Remote child", "text": "Remote child"}
				err := f.app.RunInTransaction(func(app core.App) error {
					if target == "waypoint" {
						return syncWaypoints(app, t.Context(), f.trail, "https://a.example", []any{raw})
					}
					return syncSummitLogs(app, t.Context(), f.trail, "https://a.example", []any{raw})
				})
				if err != nil {
					t.Fatal(err)
				}
				records, err := f.app.FindAllRecords(f.children)
				if err != nil || len(records) != 0 {
					t.Fatalf("child without a record id was stored: count=%d error=%v", len(records), err)
				}
			})
		}
	}
}
