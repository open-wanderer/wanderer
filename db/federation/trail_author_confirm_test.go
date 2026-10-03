package federation

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// The origin author confirmation has its own rate-limit budget, is bounded,
// and remembers refusals.

type confirmCall struct {
	iri         string
	identifier  string
	deadline    time.Time
	hasDeadline bool
}

type confirmFixture struct {
	f      *attachmentFixture
	host   string
	owner  *core.Record
	signer *core.Record
	mu     sync.Mutex
	calls  []confirmCall
}

func (c *confirmFixture) userIRI(name string) string {
	return "https://" + c.host + "/api/v1/activitypub/user/" + name
}

func (c *confirmFixture) trailIRI(name string) string {
	return "https://" + c.host + "/api/v1/trail/" + name
}

func (c *confirmFixture) callsFor(iri string) []confirmCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []confirmCall
	for _, call := range c.calls {
		if call.iri == iri {
			out = append(out, call)
		}
	}
	return out
}

// confirmSetup builds an app and a peer host unique to the run, so repeated
// runs don't share rate-limit buckets. The origin stub charges the rate limit
// under the context identifier, like the real dialer.
func confirmSetup(t *testing.T, origin func(c *confirmFixture, iri string) (*pub.Object, error)) *confirmFixture {
	t.Helper()
	f := attachmentApp(t)
	c := &confirmFixture{f: f, host: fmt.Sprintf("peer-%d.example.com", time.Now().UnixNano())}
	c.owner = createTestActor(t, f.app, c.userIRI("owner"), "person", false)
	c.signer = createTestActor(t, f.app, c.userIRI("signer"), "person", false)

	trailAuthorRefusals.reset()
	orig := fetchTrailObject
	fetchTrailObject = func(ctx context.Context, iri string) (*pub.Object, error) {
		call := confirmCall{iri: iri, identifier: util.RateLimitIdentifier(ctx)}
		call.deadline, call.hasDeadline = ctx.Deadline()
		c.mu.Lock()
		c.calls = append(c.calls, call)
		c.mu.Unlock()
		u, err := url.Parse(iri)
		if err != nil {
			return nil, err
		}
		if err := util.CheckActivityPubRateLimit(ctx, u.Hostname()); err != nil {
			return nil, err
		}
		return origin(c, iri)
	}
	t.Cleanup(func() {
		fetchTrailObject = orig
		trailAuthorRefusals.reset()
	})
	return c
}

// namesOwner answers every trail with the given id attributed to the owner.
func namesOwner(c *confirmFixture, iri string) (*pub.Object, error) {
	o := pub.ObjectNew(pub.NoteType)
	o.ID = pub.IRI(iri)
	o.AttributedTo = pub.IRI(c.owner.GetString("iri"))
	return o, nil
}

func (c *confirmFixture) update(signer *core.Record, trailIRI string) error {
	act := authorGuardAsUpdate(feedDedupTrailCreate(signer.GetString("iri"), trailIRI))
	return ProcessCreateOrUpdateActivity(c.f.app, context.Background(), signer, c.f.owner.alice, act)
}

func (c *confirmFixture) replyComment(commentName, trailIRI string) error {
	act := ownerCommentActivity(pub.CreateType, c.signer.GetString("iri"),
		"https://"+c.host+"/api/v1/comment/"+commentName, trailIRI, "nice")
	return ProcessCreateOrUpdateActivity(c.f.app, context.Background(), c.signer, c.f.owner.alice, act)
}

func TestRefusedNonAuthorUpdatesDoNotStarveTrailFetch(t *testing.T) {
	var uncached string
	c := confirmSetup(t, func(c *confirmFixture, iri string) (*pub.Object, error) {
		if iri != uncached {
			return namesOwner(c, iri)
		}
		trailAct := feedDedupTrailCreate(c.signer.GetString("iri"), iri)
		o := trailAct.Object.(*pub.Object)
		o.AttributedTo = pub.IRI(c.signer.GetString("iri"))
		return o, nil
	})
	uncached = c.trailIRI("uncached")

	for i := 0; i < 31; i++ {
		iri := c.trailIRI(fmt.Sprintf("stored%d", i))
		seedTrailRecord(t, c.f.app, iri, true, c.owner.Id)
		if err := c.update(c.signer, iri); err == nil {
			t.Fatalf("Update %d of a trail owned by someone else was accepted", i)
		}
	}

	if err := c.replyComment("c1", uncached); err != nil {
		t.Fatalf("comment replying to an uncached trail on the peer: %v", err)
	}
	if n := attachmentCount(t, c.f.app, "comments", "https://"+c.host+"/api/v1/comment/c1"); n != 1 {
		t.Errorf("comments rows = %d, want 1", n)
	}
}

func TestRepeatedNonAuthorUpdateUsesCachedRefusal(t *testing.T) {
	c := confirmSetup(t, namesOwner)
	other := createTestActor(t, c.f.app, c.userIRI("other"), "person", false)
	iri := c.trailIRI("stored")
	seedTrailRecord(t, c.f.app, iri, true, c.owner.Id)

	clock := time.Now()
	trailAuthorRefusals.now = func() time.Time { return clock }

	for i := 0; i < 5; i++ {
		if err := c.update(c.signer, iri); err == nil {
			t.Fatalf("Update %d of a trail owned by someone else was accepted", i)
		}
	}
	if n := len(c.callsFor(iri)); n != 1 {
		t.Errorf("origin fetches after 5 identical refused Updates = %d, want 1", n)
	}

	// A definitive refusal outlasts the transient window ...
	clock = clock.Add(trailAuthorTransientRefusalTTL + time.Second)
	_ = c.update(c.signer, iri)
	if n := len(c.callsFor(iri)); n != 1 {
		t.Errorf("origin fetches %v after a definitive refusal = %d, want 1", trailAuthorTransientRefusalTTL+time.Second, n)
	}

	// ... but not trailAuthorRefusalTTL.
	clock = clock.Add(trailAuthorRefusalTTL)
	_ = c.update(c.signer, iri)
	if n := len(c.callsFor(iri)); n != 2 {
		t.Errorf("origin fetches after the refusal TTL = %d, want 2", n)
	}

	// Another signer for the same trail is asked separately.
	_ = c.update(other, iri)
	if n := len(c.callsFor(iri)); n != 3 {
		t.Errorf("origin fetches after a different signer = %d, want 3", n)
	}
}

func TestTransientConfirmFailureRetriedAfterShortTTL(t *testing.T) {
	c := confirmSetup(t, func(c *confirmFixture, iri string) (*pub.Object, error) {
		return nil, errors.New("origin unreachable")
	})
	iri := c.trailIRI("stored")
	seedTrailRecord(t, c.f.app, iri, true, c.owner.Id)

	clock := time.Now()
	trailAuthorRefusals.now = func() time.Time { return clock }

	_ = c.update(c.signer, iri)
	_ = c.update(c.signer, iri)
	if n := len(c.callsFor(iri)); n != 1 {
		t.Errorf("origin fetches after a transient failure and a repeat = %d, want 1", n)
	}

	clock = clock.Add(trailAuthorTransientRefusalTTL - time.Second)
	_ = c.update(c.signer, iri)
	if n := len(c.callsFor(iri)); n != 1 {
		t.Errorf("origin fetches just inside the transient TTL = %d, want 1", n)
	}

	clock = clock.Add(2 * time.Second)
	_ = c.update(c.signer, iri)
	if n := len(c.callsFor(iri)); n != 2 {
		t.Errorf("origin fetches past the transient TTL = %d, want 2", n)
	}
}

func TestTrailAuthorConfirmFetchHasDeadlineAndOwnBudget(t *testing.T) {
	c := confirmSetup(t, namesOwner)
	iri := c.trailIRI("stored")
	seedTrailRecord(t, c.f.app, iri, true, c.owner.Id)

	_ = c.update(c.signer, iri)

	calls := c.callsFor(iri)
	if len(calls) != 1 {
		t.Fatalf("origin fetches = %d, want 1", len(calls))
	}
	call := calls[0]
	if !call.hasDeadline {
		t.Error("confirmation fetch has no deadline")
	} else if left := time.Until(call.deadline); left > trailAuthorConfirmTimeout || left < trailAuthorConfirmTimeout-5*time.Second {
		t.Errorf("confirmation deadline in %v, want about %v", left, trailAuthorConfirmTimeout)
	}
	if call.identifier != trailAuthorConfirmRateLimitID {
		t.Errorf("confirmation rate-limit identifier = %q, want %q", call.identifier, trailAuthorConfirmRateLimitID)
	}
}

func TestFetchTrailKeepsSystemBudget(t *testing.T) {
	var uncached string
	c := confirmSetup(t, func(c *confirmFixture, iri string) (*pub.Object, error) {
		trailAct := feedDedupTrailCreate(c.signer.GetString("iri"), iri)
		o := trailAct.Object.(*pub.Object)
		o.AttributedTo = pub.IRI(c.signer.GetString("iri"))
		return o, nil
	})
	uncached = c.trailIRI("uncached")

	if err := c.replyComment("c1", uncached); err != nil {
		t.Fatalf("comment replying to an uncached trail: %v", err)
	}
	calls := c.callsFor(uncached)
	if len(calls) != 1 {
		t.Fatalf("origin fetches = %d, want 1", len(calls))
	}
	if calls[0].identifier != "system" {
		t.Errorf("fetchTrail rate-limit identifier = %q, want %q", calls[0].identifier, "system")
	}
}
