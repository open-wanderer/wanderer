package federation

import (
	"sort"
	"testing"
)

// TestAcceptedPeerActorsOnlyRemoteInstanceActors checks that only remote
// instance actors with an accepted relationship count as peers.
func TestAcceptedPeerActorsOnlyRemoteInstanceActors(t *testing.T) {
	f := newGateFixture(t)

	const remotePersonIRI = "https://person.example.com/api/v1/activitypub/user/mallory"
	remotePerson := createTestActor(t, f.app, remotePersonIRI, "person", false)

	// a local user inserting itself as an accepted inbound follower
	gateSeedFollow(t, f.app, f.user.Id, f.instance.Id, "accepted")
	// a remote person with an accepted inbound row
	gateSeedFollow(t, f.app, remotePerson.Id, f.instance.Id, "accepted")
	// an accepted outbound row towards a local person
	gateSeedFollow(t, f.app, f.instance.Id, f.stranger.Id, "accepted")

	peers, err := acceptedPeerActors(f.app)
	if err != nil {
		t.Fatalf("acceptedPeerActors: %v", err)
	}
	var got []string
	for _, p := range peers {
		got = append(got, p.IRI)
	}
	sort.Strings(got)
	want := []string{gatePeer1, gatePeer2}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("peers = %v, want only the remote instance actors %v", got, want)
	}

	inboxes, err := instanceFollowerInboxes(f.app)
	if err != nil {
		t.Fatalf("instanceFollowerInboxes: %v", err)
	}
	for _, inbox := range inboxes {
		if inbox == f.user.GetString("inbox") || inbox == remotePerson.GetString("inbox") {
			t.Fatalf("fan-out inboxes %v must not contain a person inbox (%s)", inboxes, inbox)
		}
	}
	if len(inboxes) != 2 {
		t.Fatalf("inboxes = %v, want 2", inboxes)
	}

	cases := []struct {
		name string
		iri  string
		want bool
	}{
		{"own origin host", gateOrigin + "/api/v1/activitypub/user/alice", false},
		{"remote person host", remotePersonIRI, false},
		{"user on remote instance peer host", "https://peer1.example.com/api/v1/activitypub/user/bob", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := IsAcceptedPeerHost(f.app, tc.iri)
			if err != nil {
				t.Fatalf("IsAcceptedPeerHost: %v", err)
			}
			if ok != tc.want {
				t.Fatalf("IsAcceptedPeerHost(%q) = %v, want %v", tc.iri, ok, tc.want)
			}
		})
	}
}

// TestProcessInstanceFollowRefusesNonInstanceFollower checks that a Follow of
// the local instance actor from a non-instance actor is refused without writing
// a row.
func TestProcessInstanceFollowRefusesNonInstanceFollower(t *testing.T) {
	f := newGateFixture(t)

	person := createTestActor(t, f.app, "https://person.example.com/api/v1/activitypub/user/mallory", "person", false)
	activity := lifecycleFollow(
		"https://person.example.com/api/v1/activitypub/activity/f1",
		person.GetString("iri"), f.instance.GetString("iri"))

	if err := ProcessFollowActivity(f.app, person, activity); err == nil {
		t.Fatal("Follow of the instance actor from a person must be refused")
	}

	rows, err := f.app.FindRecordsByFilter("follows",
		"follower={:a} && followee={:b}", "", 10, 0,
		map[string]any{"a": person.Id, "b": f.instance.Id})
	if err != nil {
		t.Fatalf("query follows: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("want no follows row, got %d", len(rows))
	}
	acts, err := f.app.FindRecordsByFilter("activitypub_activities", "type='Follow'", "", 10, 0)
	if err != nil {
		t.Fatalf("query activities: %v", err)
	}
	if len(acts) != 0 {
		t.Fatalf("want no Follow activity row, got %d", len(acts))
	}
}
