package permissions_test

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// Recorded activities are publicly listed, except the hidden ones: snapshots
// of private trails and lists and of their replies, and the stripped rows of
// deleted objects.
func TestHiddenActivitiesAreNotPublic(t *testing.T) {
	app := newRulesTestApp(t)
	user := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "owner", "password": "test-password", "email": "owner@example.com",
	})
	actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
		"username": "owner", "preferred_username": "owner", "domain": "example.com", "user": user.Id, "public_key": "test-key", "is_local": true,
		"iri": "https://example.com/owner", "inbox": "https://example.com/owner/inbox",
	})

	for _, test := range []struct {
		name   string
		hidden bool
		auth   *core.Record
		want   bool
	}{
		{"anonymous visible", false, nil, true},
		{"anonymous hidden", true, nil, false},
		{"authenticated hidden", true, user, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			activity := saveRulesTestRecord(t, app, "activitypub_activities", map[string]any{
				"iri": "https://example.com/api/v1/activitypub/activity/" + strings.ReplaceAll(test.name, " ", "-"), "type": "Create",
				"actor": actor.GetString("iri"), "object": map[string]any{"id": "https://example.com/api/v1/trail/t"},
				"published": "2026-01-01 00:00:00.000Z",
				"hidden":    test.hidden,
			})
			fresh, err := app.FindRecordById("activitypub_activities", activity.Id)
			if err != nil {
				t.Fatal(err)
			}
			info := &core.RequestInfo{Auth: test.auth}
			for _, check := range []struct {
				name string
				rule *string
			}{
				{"list", fresh.Collection().ListRule},
				{"view", fresh.Collection().ViewRule},
			} {
				got, err := app.CanAccessRecord(fresh, info, check.rule)
				if err != nil || got != test.want {
					t.Errorf("%s access = %v, %v; want %v", check.name, got, err, test.want)
				}
			}
		})
	}
}
