package federation

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// Who is told when a local account is deleted. Followers alone are not enough:
// comments, summit logs, likes, shares and outgoing follows are delivered
// straight to one remote inbox, so an actor who never followed the departing
// account still holds a record tied to it and has to hear that it is gone.
// Mentioned actors leave no relation row, but the inboxes every Create/Update
// went out to are recorded on the activity, so they are read back from there.

type recipientsFixture struct {
	app    *pbtests.TestApp
	actors *core.Collection
	trails *core.Collection
	lists  *core.Collection
}

func setupRecipientsTestApp(t *testing.T) *recipientsFixture {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "inbox"},
		&core.BoolField{Name: "is_local"},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}

	follows := core.NewBaseCollection("follows")
	follows.Fields.Add(
		&core.RelationField{Name: "follower", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "followee", CollectionId: actors.Id, MaxSelect: 1},
		&core.TextField{Name: "status"},
	)
	if err := app.Save(follows); err != nil {
		t.Fatal(err)
	}

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"comments", "summit_logs"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
			&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
	}

	for _, name := range []string{"trail_like", "trail_share"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
			&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
	}

	listShare := core.NewBaseCollection("list_share")
	listShare.Fields.Add(
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "list", CollectionId: lists.Id, MaxSelect: 1},
	)
	if err := app.Save(listShare); err != nil {
		t.Fatal(err)
	}

	activities := core.NewBaseCollection("activitypub_activities")
	activities.Fields.Add(
		&core.TextField{Name: "actor"},
		&core.TextField{Name: "type"},
		&core.JSONField{Name: "cc"},
	)
	if err := app.Save(activities); err != nil {
		t.Fatal(err)
	}

	return &recipientsFixture{app: app, actors: actors, trails: trails, lists: lists}
}

// activity records an outgoing activity of actor the way create.go does: the
// actor's IRI, the activity type and the inboxes it was cc'd to.
func (f *recipientsFixture) activity(t *testing.T, actor *core.Record, typ string, cc ...string) {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set("actor", actor.GetString("iri"))
	r.Set("type", typ)
	r.Set("cc", cc)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *recipientsFixture) actor(t *testing.T, name string, local bool) *core.Record {
	t.Helper()

	host := "remote.example"
	if local {
		host = "local.example"
	}
	iri := "https://" + host + "/api/v1/activitypub/user/" + name

	r := core.NewRecord(f.actors)
	r.Set("iri", iri)
	r.Set("inbox", iri+"/inbox")
	r.Set("is_local", local)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *recipientsFixture) follow(t *testing.T, follower, followee *core.Record, status string) {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set("follower", follower.Id)
	r.Set("followee", followee.Id)
	r.Set("status", status)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func (f *recipientsFixture) trail(t *testing.T, author *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(f.trails)
	r.Set("author", author.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *recipientsFixture) list(t *testing.T, author *core.Record) *core.Record {
	t.Helper()

	r := core.NewRecord(f.lists)
	r.Set("author", author.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *recipientsFixture) contentOn(t *testing.T, collection string, author, trail *core.Record) {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set("author", author.Id)
	r.Set("trail", trail.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

// actorOn writes a trail_like / trail_share / list_share row, which key the
// acting party as "actor" rather than "author".
func (f *recipientsFixture) actorOn(t *testing.T, collection, subjectField string, actor, subject *core.Record) {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Set("actor", actor.Id)
	r.Set(subjectField, subject.Id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
}

func assertRecipients(t *testing.T, got []string, want ...*core.Record) {
	t.Helper()

	wantInboxes := make([]string, 0, len(want))
	for _, r := range want {
		wantInboxes = append(wantInboxes, r.GetString("inbox"))
	}
	slices.Sort(got)
	slices.Sort(wantInboxes)

	if !slices.Equal(got, wantInboxes) {
		t.Fatalf("recipients = %v, want %v", got, wantInboxes)
	}
}

func TestActorDeleteRecipients(t *testing.T) {
	// The reviewer's repro: A comments on B's trail, B does not follow A, A
	// deletes her account. B holds the comment and must be told.
	t.Run("TrailAuthorOfCommentedTrailWithoutFollowing", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		trailAuthor := f.actor(t, "bob", false)
		f.contentOn(t, "comments", departing, f.trail(t, trailAuthor))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, trailAuthor)
	})

	t.Run("TrailAuthorOfSummitLoggedTrailWithoutFollowing", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		trailAuthor := f.actor(t, "bob", false)
		f.contentOn(t, "summit_logs", departing, f.trail(t, trailAuthor))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, trailAuthor)
	})

	t.Run("FollowersStillIncluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		follower := f.actor(t, "carol", false)
		f.follow(t, follower, departing, "accepted")

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, follower)
	})

	// The remote side stores the follow row the moment our Follow arrives,
	// before it is accepted, so a pending outgoing follow already counts.
	t.Run("FolloweesIncludedRegardlessOfStatus", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		accepted := f.actor(t, "bob", false)
		pending := f.actor(t, "carol", false)
		f.follow(t, departing, accepted, "accepted")
		f.follow(t, departing, pending, "pending")

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, accepted, pending)
	})

	t.Run("TrailAuthorOfLikedTrail", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		trailAuthor := f.actor(t, "bob", false)
		f.actorOn(t, "trail_like", "trail", departing, f.trail(t, trailAuthor))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, trailAuthor)
	})

	// The reverse direction: someone who interacted with one of the departing
	// actor's trails is holding a copy of that trail, because viewing a remote
	// trail stores it locally. They need the notice whether or not they follow.
	t.Run("RemoteCommenterOnOwnTrail", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		commenter := f.actor(t, "bob", false)
		f.contentOn(t, "comments", commenter, f.trail(t, departing))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, commenter)
	})

	t.Run("RemoteSummitLoggerOnOwnTrail", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		logger := f.actor(t, "bob", false)
		f.contentOn(t, "summit_logs", logger, f.trail(t, departing))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, logger)
	})

	t.Run("RemoteLikerOfOwnTrail", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		liker := f.actor(t, "bob", false)
		f.actorOn(t, "trail_like", "trail", liker, f.trail(t, departing))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, liker)
	})

	// A local interaction with the departing actor's trail is removed by the
	// cascade on this instance; there is nobody remote to tell.
	t.Run("LocalInteractionsWithOwnTrailExcluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		local := f.actor(t, "dave", true)
		trail := f.trail(t, departing)
		f.contentOn(t, "comments", local, trail)
		f.contentOn(t, "summit_logs", local, trail)
		f.actorOn(t, "trail_like", "trail", local, trail)

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	// A share links two actors directly; whichever side departs, the other
	// holds the share row.
	t.Run("TrailShareBothDirections", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		sharedWith := f.actor(t, "bob", false)
		sharedBy := f.actor(t, "carol", false)
		f.actorOn(t, "trail_share", "trail", sharedWith, f.trail(t, departing))
		f.actorOn(t, "trail_share", "trail", departing, f.trail(t, sharedBy))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, sharedWith, sharedBy)
	})

	t.Run("ListShareBothDirections", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		sharedWith := f.actor(t, "bob", false)
		sharedBy := f.actor(t, "carol", false)
		f.actorOn(t, "list_share", "list", sharedWith, f.list(t, departing))
		f.actorOn(t, "list_share", "list", departing, f.list(t, sharedBy))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, sharedWith, sharedBy)
	})

	// Shares between two other actors, and remote likes on someone else's
	// trail, have nothing to do with the departing account.
	t.Run("OtherPeoplesSharesAndLikesExcluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		localOther := f.actor(t, "erin", true)
		remoteA := f.actor(t, "bob", false)
		remoteB := f.actor(t, "carol", false)
		f.actorOn(t, "trail_share", "trail", remoteA, f.trail(t, localOther))
		f.actorOn(t, "list_share", "list", remoteA, f.list(t, localOther))
		f.actorOn(t, "trail_like", "trail", localOther, f.trail(t, remoteB))
		f.follow(t, localOther, remoteB, "accepted")

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	// The reviewer's repro: Alice comments on Bob's local trail and mentions
	// Carol on another instance. Carol got the comment through the mention
	// alone, so she has to hear that Alice is gone.
	t.Run("MentionedActorFromRecordedAudience", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		localAuthor := f.actor(t, "bob", true)
		mentioned := f.actor(t, "carol", false)
		f.contentOn(t, "comments", departing, f.trail(t, localAuthor))
		f.activity(t, departing, "Create", mentioned.GetString("inbox"), localAuthor.GetString("inbox"))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, mentioned)
	})

	// An edit can mention someone the original did not; both audiences hold
	// a copy.
	t.Run("RecordedUpdateAudienceIncluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		first := f.actor(t, "carol", false)
		later := f.actor(t, "erin", false)
		f.activity(t, departing, "Create", first.GetString("inbox"))
		f.activity(t, departing, "Update", later.GetString("inbox"))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, first, later)
	})

	// cc also carries the actor's own followers collection and, for local
	// mentions, local inboxes; neither is a remote actor to notify. Values
	// that match no known actor are not posted to blindly.
	t.Run("RecordedAudienceOnlyKnownRemoteInboxes", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		local := f.actor(t, "bob", true)
		f.activity(t, departing, "Create",
			departing.GetString("iri")+"/followers",
			local.GetString("inbox"),
			"https://remote.example/api/v1/activitypub/user/unknown/inbox",
		)

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	// Other actors' audiences and non-content activities say nothing about
	// who holds the departing account's content.
	t.Run("OtherActivitiesRecordedAudienceExcluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		other := f.actor(t, "bob", true)
		mentioned := f.actor(t, "carol", false)
		f.activity(t, other, "Create", mentioned.GetString("inbox"))
		f.activity(t, departing, "Delete", mentioned.GetString("inbox"))
		f.activity(t, departing, "Announce", mentioned.GetString("inbox"))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	t.Run("UnionIsDeduplicated", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		both := f.actor(t, "bob", false)
		f.follow(t, both, departing, "accepted")
		f.follow(t, departing, both, "accepted")
		trail := f.trail(t, both)
		f.contentOn(t, "comments", departing, trail)
		f.contentOn(t, "summit_logs", departing, trail)
		f.actorOn(t, "trail_like", "trail", departing, trail)
		f.actorOn(t, "trail_share", "trail", both, f.trail(t, departing))
		f.actorOn(t, "list_share", "list", both, f.list(t, departing))
		f.activity(t, departing, "Create", both.GetString("inbox"))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got, both)
	})

	// A local trail author's copy lives on this instance and is removed by
	// the cascade; there is nothing to send them.
	t.Run("LocalTrailAuthorExcluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		localAuthor := f.actor(t, "dave", true)
		f.contentOn(t, "comments", departing, f.trail(t, localAuthor))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	// Someone else's comment on a remote trail says nothing about the
	// departing account; only its own interactions count.
	t.Run("OtherPeoplesInteractionsExcluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		someoneElse := f.actor(t, "erin", true)
		trailAuthor := f.actor(t, "bob", false)
		f.contentOn(t, "comments", someoneElse, f.trail(t, trailAuthor))

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	t.Run("PendingFollowExcluded", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		departing := f.actor(t, "alice", true)
		pending := f.actor(t, "carol", false)
		f.follow(t, pending, departing, "pending")

		got, err := ActorDeleteRecipients(f.app, departing)
		if err != nil {
			t.Fatal(err)
		}
		assertRecipients(t, got)
	})

	t.Run("RemoteActorHasNoRecipients", func(t *testing.T) {
		f := setupRecipientsTestApp(t)

		remote := f.actor(t, "bob", false)
		f.follow(t, f.actor(t, "carol", false), remote, "accepted")

		got, err := ActorDeleteRecipients(f.app, remote)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatalf("a remote actor's deletion is not ours to announce, got %v", got)
		}
	})
}
