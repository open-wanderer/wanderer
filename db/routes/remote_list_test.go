package routes

import (
	"net/url"
	"testing"
)

func TestRemoteListSyncQueryAddsCoreExpands(t *testing.T) {
	reqURL, err := url.Parse("https://local.example/api/v1/list/abc?handle=@alice@example.test&expand=list_share_via_list.actor&foo=bar")
	if err != nil {
		t.Fatal(err)
	}

	query := remoteListSyncQuery(reqURL, true)

	if got := query.Get("handle"); got != "" {
		t.Fatalf("handle query should not be forwarded, got %q", got)
	}
	if got := query.Get("foo"); got != "bar" {
		t.Fatalf("expected unrelated query param to be preserved, got %q", got)
	}

	expands := expandSet(query.Get("expand"))
	for _, required := range remoteListSyncCoreExpandPaths {
		if !expands[required] {
			t.Fatalf("expected core expand %q in %q", required, query.Get("expand"))
		}
	}
	if !expands["list_share_via_list.actor"] {
		t.Fatalf("expected caller expand to be preserved, got %q", query.Get("expand"))
	}
}

func TestRemoteListSyncQueryWithoutAssetExpandsKeepsTrails(t *testing.T) {
	reqURL, err := url.Parse("https://local.example/api/v1/list/abc?expand=trails.trail_assets_via_trail.asset,trails.waypoints_via_trail.waypoint_assets_via_waypoint.asset")
	if err != nil {
		t.Fatal(err)
	}

	query := remoteListSyncQuery(reqURL, false)

	expands := expandSet(query.Get("expand"))
	for _, removed := range remoteListSyncAssetExpandPaths {
		if expands[removed] {
			t.Fatalf("did not expect asset expand %q in %q", removed, query.Get("expand"))
		}
	}
	for _, required := range remoteListSyncCoreExpandPaths {
		if !expands[required] {
			t.Fatalf("expected core expand %q in %q", required, query.Get("expand"))
		}
	}
}
