package util

import "testing"

func TestObjectKindFromIRI(t *testing.T) {
	t.Run("Kinds", func(t *testing.T) {
		cases := map[string]ObjectKind{
			"https://wanderer.example/api/v1/trail/abc123":                 ObjectKindTrail,
			"https://wanderer.example/api/v1/comment/abc123":               ObjectKindComment,
			"https://wanderer.example/api/v1/summit-log/abc123":            ObjectKindSummitLog,
			"https://wanderer.example/api/v1/list/abc123":                  ObjectKindList,
			"https://wanderer.example/api/v1/waypoint/abc123":              ObjectKindWaypoint,
			"https://wanderer.example/api/v1/activitypub/user/alice":       ObjectKindActor,
			"https://wanderer.example/api/v1/activitypub/activity/abc123":  ObjectKindActivity,
			"https://wanderer.example/api/v1/activitypub/user/alice/inbox": ObjectKindActor,
		}

		for iri, want := range cases {
			if got := ObjectKindFromIRI(iri); got != want {
				t.Fatalf("ObjectKindFromIRI(%q) = %q, want %q", iri, got, want)
			}
		}
	})

	// The bug this classifier replaced: the host is part of the IRI, so matching
	// a bare keyword anywhere in the string misrouted every object sent by an
	// instance whose domain happened to contain one.
	t.Run("HostIsNotMatched", func(t *testing.T) {
		cases := map[string]ObjectKind{
			"https://mylist.social/api/v1/trail/abc123":       ObjectKindTrail,
			"https://trails.example/api/v1/comment/abc123":    ObjectKindComment,
			"https://comment.example/api/v1/list/abc123":      ObjectKindList,
			"https://summit-log.example/api/v1/trail/abc123":  ObjectKindTrail,
			"https://list.example/api/v1/activitypub/user/bo": ObjectKindActor,
		}

		for iri, want := range cases {
			if got := ObjectKindFromIRI(iri); got != want {
				t.Fatalf("ObjectKindFromIRI(%q) = %q, want %q", iri, got, want)
			}
		}
	})

	// Same trap in the other half of the IRI: a username is attacker-chosen.
	t.Run("UsernameIsNotMatched", func(t *testing.T) {
		for _, iri := range []string{
			"https://wanderer.example/api/v1/activitypub/user/trailrunner",
			"https://wanderer.example/api/v1/activitypub/user/listkeeper",
			"https://wanderer.example/api/v1/activitypub/user/comment",
		} {
			if got := ObjectKindFromIRI(iri); got != ObjectKindActor {
				t.Fatalf("ObjectKindFromIRI(%q) = %q, want %q", iri, got, ObjectKindActor)
			}
		}
	})

	t.Run("PathPrefixedOrigin", func(t *testing.T) {
		if got := ObjectKindFromIRI("https://example.com/wanderer/api/v1/trail/abc"); got != ObjectKindTrail {
			t.Fatalf("path-prefixed origin: got %q, want %q", got, ObjectKindTrail)
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		for _, iri := range []string{
			"",
			"not a url",
			"https://mastodon.example/users/alice/statuses/123",
			"https://wanderer.example/api/v1/trail",       // matched by the old keyword search
			"https://wanderer.example/api/v2/trail/abc",   // wrong version
			"https://wanderer.example/v1/trail/abc",       // no api segment
			"https://wanderer.example/api/v1/activitypub", // truncated
			"https://wanderer.example/api/v1/activitypub/nope",
			"https://wanderer.example/api/v1",
		} {
			if got := ObjectKindFromIRI(iri); got != ObjectKindUnknown {
				t.Fatalf("ObjectKindFromIRI(%q) = %q, want unknown", iri, got)
			}
		}
	})
}
