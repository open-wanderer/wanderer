package routes

import (
	"net/url"
	"strings"
	"testing"
)

func TestRemoteTrailSyncQueryAddsRequiredExpands(t *testing.T) {
	reqURL, err := url.Parse("https://local.example/api/v1/trail/abc?handle=@alice@example.test&expand=tags,waypoints_via_trail&foo=bar")
	if err != nil {
		t.Fatal(err)
	}

	query := remoteTrailSyncQuery(reqURL)

	if got := query.Get("handle"); got != "" {
		t.Fatalf("handle query should not be forwarded, got %q", got)
	}
	if got := query.Get("foo"); got != "bar" {
		t.Fatalf("expected unrelated query param to be preserved, got %q", got)
	}

	expands := expandSet(query.Get("expand"))
	for _, required := range remoteTrailSyncExpandPaths {
		if !expands[required] {
			t.Fatalf("expected required expand %q in %q", required, query.Get("expand"))
		}
	}
	if countExpand(query.Get("expand"), "waypoints_via_trail") != 1 {
		t.Fatalf("expected waypoints_via_trail to be deduplicated, got %q", query.Get("expand"))
	}
}

func TestRemoteTrailSyncQueryWithNilURL(t *testing.T) {
	query := remoteTrailSyncQuery(nil)

	expands := expandSet(query.Get("expand"))
	for _, required := range remoteTrailSyncExpandPaths {
		if !expands[required] {
			t.Fatalf("expected required expand %q in %q", required, query.Get("expand"))
		}
	}
}

func expandSet(value string) map[string]bool {
	result := map[string]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result[part] = true
		}
	}
	return result
}

func countExpand(value string, needle string) int {
	count := 0
	for _, part := range strings.Split(value, ",") {
		if strings.TrimSpace(part) == needle {
			count++
		}
	}
	return count
}
