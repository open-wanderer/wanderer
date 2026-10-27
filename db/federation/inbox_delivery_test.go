package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/tools/security"
)

// Inbox URLs come from remote actor documents, so delivering an activity must
// not reach loopback or private addresses.

func TestFollowAcceptIsNotDeliveredToLoopbackInbox(t *testing.T) {
	const encryptionKey = "0123456789abcdef0123456789abcdef"
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", encryptionKey)

	app := newInboxTestApp(t)
	actors := feedDedupSeedActors(t, app)

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}))
	t.Cleanup(server.Close)

	// The local followee signs the Accept.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(key), encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	actors.alice.Set("private_key", encrypted)
	if err := app.Save(actors.alice); err != nil {
		t.Fatal(err)
	}

	// The remote follower advertises an inbox on the loopback server.
	actors.author.Set("inbox", server.URL+"/internal?x=1")
	if err := app.Save(actors.author); err != nil {
		t.Fatal(err)
	}

	follow := pub.FollowNew(pub.IRI(feedDedupAuthorIRI+"/follow/1"), pub.IRI(feedDedupAliceIRI))
	follow.Actor = pub.IRI(feedDedupAuthorIRI)
	// The fixture lacks the records needed after the Accept is sent, so the
	// error is expected and irrelevant here.
	_ = ProcessFollowActivity(app, actors.author, *follow)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("loopback inbox received %d requests, want 0", n)
	}
}
