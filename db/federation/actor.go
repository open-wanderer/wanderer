package federation

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"pocketbase/util"
	"regexp"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

var ErrProfilePrivate = errors.New("profile is private")
var ErrActorNotFound = errors.New("actor not found")

var newActorFetchHTTPClient = util.SafeHTTPClient
var newWebfingerHTTPClient = util.SafeHTTPClient
var ErrInvalidActorResponse = errors.New("invalid or incomplete actor response")
var ErrInvalidCursor = errors.New("invalid collection cursor")

// SetActorFetchClientForTesting replaces the client remote actor documents
// are fetched with and returns a restore function. Tests only.
func SetActorFetchClientForTesting(f func() *http.Client) (restore func()) {
	previous := newActorFetchHTTPClient
	newActorFetchHTTPClient = f
	return func() { newActorFetchHTTPClient = previous }
}

type WebfingerResponse struct {
	Subject string `json:"subject"`
	Links   []struct {
		Rel  string `json:"rel"`
		Href string `json:"href"`
	} `json:"links"`
}

func validateActorResponse(actor *pub.Actor) error {
	if actor == nil {
		return ErrInvalidActorResponse
	}

	if actor.GetID().String() == "" {
		return fmt.Errorf("%w: missing ID", ErrInvalidActorResponse)
	}

	if actor.PreferredUsername.String() == "" && actor.Name.String() == "" {
		return fmt.Errorf("%w: missing username or name", ErrInvalidActorResponse)
	}

	if util.ItemID(actor.Inbox) == "" {
		return fmt.Errorf("%w: missing inbox", ErrInvalidActorResponse)
	}

	if util.ItemID(actor.Outbox) == "" {
		return fmt.Errorf("%w: missing outbox", ErrInvalidActorResponse)
	}

	if actor.PublicKey.PublicKeyPem == "" {
		return fmt.Errorf("%w: missing public key", ErrInvalidActorResponse)
	}

	return nil
}

func GetActorByHandle(app core.App, ctx context.Context, handle string, includeFollows bool) (*core.Record, error) {
	username, domain := util.SplitHandle(handle)
	if domain != "" && isLocalHandleDomain(domain) {
		domain = ""
	}
	if domain == "" {
		username = strings.ToLower(username)
	}

	filter := "preferred_username={:username}&&"
	if domain != "" {
		filter += "domain={:domain}"
	} else {
		filter += "is_local=true"
	}

	var dbActor *core.Record
	dbActor, err := app.FindFirstRecordByFilter("activitypub_actors", filter, dbx.Params{"username": username, "domain": domain})
	if err != nil && err == sql.ErrNoRows {
		if domain == "" {
			return nil, ErrActorNotFound
		}
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return nil, err
		}

		dbActor = core.NewRecord(collection)
		dbActor.Set("is_local", false)
		iri, err := iriFromHandle(ctx, domain, username)
		if err != nil {
			return nil, err
		}
		dbActor.Set("iri", iri)

	} else if err != nil {
		return nil, err
	}

	return assembleActor(app, ctx, dbActor, includeFollows || dbActor.Id == "")
}

func isLocalHandleDomain(domain string) bool {
	origin, err := url.Parse(os.Getenv("ORIGIN"))
	if err != nil || origin.Hostname() == "" {
		return false
	}
	handleDomain, err := url.Parse("//" + domain)
	if err != nil || handleDomain.Host != domain || handleDomain.Hostname() == "" || strings.HasSuffix(handleDomain.Host, ":") {
		return false
	}
	if !strings.EqualFold(handleDomain.Hostname(), origin.Hostname()) &&
		!strings.EqualFold(handleDomain.Hostname(), strings.TrimPrefix(origin.Hostname(), "www.")) {
		return false
	}
	if handleDomain.Port() == "" {
		return true
	}
	port := origin.Port()
	if port == "" {
		switch origin.Scheme {
		case "https":
			port = "443"
		case "http":
			port = "80"
		default:
			return false
		}
	}
	return handleDomain.Port() == port
}

func isLocalActorIRI(iri string) bool {
	origin, err := url.Parse(os.Getenv("ORIGIN"))
	if err != nil || origin.Hostname() == "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return false
	}
	actorIRI, err := url.Parse(iri)
	if err != nil || (actorIRI.Scheme != "http" && actorIRI.Scheme != "https") {
		return false
	}
	if !strings.EqualFold(actorIRI.Hostname(), origin.Hostname()) &&
		!strings.EqualFold(actorIRI.Hostname(), strings.TrimPrefix(origin.Hostname(), "www.")) {
		return false
	}
	port := func(u *url.URL) string {
		if u.Port() != "" {
			return u.Port()
		}
		if u.Scheme == "https" {
			return "443"
		}
		return "80"
	}
	originPort, actorPort := port(origin), port(actorIRI)
	// Unlike handles, an IRI without a port uses its scheme's default port.
	// The default HTTP endpoint can also be an alias of an HTTPS origin.
	return actorPort == originPort || (origin.Scheme == "https" && originPort == "443" && actorIRI.Scheme == "http" && actorPort == "80")
}

type actorIdentityLookupKey struct{}

// WithActorIdentityLookup avoids fetching follower collections when a sync only
// needs to confirm a contributor's identity. Profile lookups keep their defaults.
func WithActorIdentityLookup(ctx context.Context) context.Context {
	return context.WithValue(ctx, actorIdentityLookupKey{}, true)
}

func GetActorByIRI(app core.App, ctx context.Context, iri string, includeFollows bool) (*core.Record, error) {
	var dbActor *core.Record
	dbActor, err := app.FindFirstRecordByFilter("activitypub_actors", "iri={:iri}", dbx.Params{"iri": iri})
	if err != nil && err == sql.ErrNoRows {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return nil, err
		}

		dbActor = core.NewRecord(collection)
		dbActor.Set("is_local", false)
		dbActor.Set("iri", iri)

	} else if err != nil {
		return nil, err
	}

	return assembleActor(app, ctx, dbActor, includeFollows || (dbActor.Id == "" && ctx.Value(actorIdentityLookupKey{}) != true))
}

func iriFromHandle(ctx context.Context, domain string, username string) (string, error) {
	client := newWebfingerHTTPClient()

	u := &url.URL{
		Scheme: "https",
		Host:   domain,
		Path:   "/.well-known/webfinger",
	}
	q := u.Query()
	q.Set("resource", fmt.Sprintf("acct:%s@%s", username, domain))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("webfinger request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return "", ErrActorNotFound
		}
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var wf WebfingerResponse
	if err := util.DecodeFederationJSON(resp.Body, util.WebfingerBodyLimit, &wf); err != nil {
		return "", fmt.Errorf("failed to decode JSON: %w", err)
	}

	for _, link := range wf.Links {
		if link.Rel == "self" {
			if _, err := url.Parse(link.Href); err != nil {
				return "", fmt.Errorf("invalid IRI in response")
			}
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("no iri in response")
}

func assembleActor(app core.App, ctx context.Context, dbActor *core.Record, includeFollows bool) (*core.Record, error) {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return nil, fmt.Errorf("ORIGIN environment variable not set")
	}

	private := false
	if dbActor.GetBool("is_local") && dbActor.GetString("actor_type") == "instance" {
		// The instance actor has no user record.
		return dbActor, nil
	}
	if dbActor.GetBool("is_local") {
		user, err := app.FindRecordById("users", dbActor.GetString("user"))
		if err != nil {
			return nil, err
		}
		settings, err := app.FindFirstRecordByData("settings", "user", user.Id)
		if err != nil {
			return nil, err
		}

		if user.GetString("avatar") != "" {
			dbActor.Set("icon", fmt.Sprintf("%s/api/v1/files/users/%s/%s", origin, user.Id, user.GetString("avatar")))
		}
		dbActor.Set("summary", settings.GetString("bio"))
		followerCount, err := app.CountRecords("follows", dbx.NewExp("followee={:user} AND status='accepted'", dbx.Params{"user": dbActor.Id}))
		if err != nil {
			return nil, err
		}
		dbActor.Set("follower_count", followerCount)
		followingCount, err := app.CountRecords("follows", dbx.NewExp("follower={:user} AND status='accepted'", dbx.Params{"user": dbActor.Id}))
		if err != nil {
			return nil, err
		}
		dbActor.Set("following_count", followingCount)

		dbActor.Set("last_fetched", time.Now())

		// an empty privacy field is the default for users who never touched
		// their privacy settings and is treated as public. A non-empty but
		// corrupt value fails closed (private) so a broken setting can't
		// silently expose a profile.
		privacy := settings.GetString("privacy")
		if privacy != "" {
			result := make(map[string]interface{})
			if err := json.Unmarshal([]byte(privacy), &result); err != nil {
				private = true
			} else {
				// check that it's not our own profile
				actorVal, _ := ctx.Value("actor").(string)
				private = result["account"] == "private" && dbActor.Id != strings.TrimPrefix(actorVal, "actor:")
			}
		}

	} else {
		if isLocalActorIRI(dbActor.GetString("iri")) {
			return nil, ErrActorNotFound
		}

		// check if value is still cached
		twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)
		if CachedActorIdentityMatchesOrigin(dbActor) && dbActor.GetDateTime("last_fetched").Time().After(twoHoursAgo) {
			if includeFollows {
				return completeActorFollowCounts(app, ctx, dbActor)
			}
			return dbActor, nil
		}
		pubActor, followers, following, err := fetchRemoteActor(app, ctx, dbActor.GetString("iri"), includeFollows)
		if err != nil {
			if dbActor.Id != "" {
				return dbActor, err
			}
			return nil, err
		}

		icon := ""
		if pub.IsObject(pubActor.Icon) {
			iconObject, err := pub.ToObject(pubActor.Icon)
			if err == nil && iconObject.URL != nil {
				icon = iconObject.URL.GetID().String()
			}
		}

		parsedUrl, err := url.Parse(dbActor.GetString("iri"))
		if err != nil {
			return nil, err
		}
		domain := parsedUrl.Host

		// this is a race condition that gets triggered when the profile is opened for the first time
		existingActor, _ := app.FindFirstRecordByData("activitypub_actors", "iri", dbActor.GetString("iri"))

		if existingActor != nil {
			if existingActor.GetBool("is_local") {
				return nil, fmt.Errorf("%w: remote response matches a local actor", ErrInvalidActorResponse)
			}
			dbActor = existingActor
		}

		dbActor.Set("domain", domain)
		dbActor.Set("followers", util.ItemID(pubActor.Followers))
		dbActor.Set("inbox", util.ItemID(pubActor.Inbox))
		// Keep the requested identity stable after a same-host HTTPS upgrade.
		// The new document and key were validated; rewriting the stored IRI
		// would split existing follows and later activities into two actors.
		dbActor.Set("username", pubActor.Name.String())
		dbActor.Set("preferred_username", pubActor.PreferredUsername.String())
		dbActor.Set("following", util.ItemID(pubActor.Following))
		dbActor.Set("summary", pubActor.Summary.String())
		dbActor.Set("outbox", util.ItemID(pubActor.Outbox))
		dbActor.Set("icon", icon)
		dbActor.Set("published", pubActor.Published.String())
		dbActor.Set("public_key", pubActor.PublicKey.PublicKeyPem)
		// A key confirmed at the origin ends the pre-upgrade key for good.
		dbActor.Set("legacy_public_key", "")
		dbActor.Set("last_fetched", time.Now())
		if pubActor.Type == pub.ApplicationType {
			dbActor.Set("actor_type", "instance")
		} else {
			dbActor.Set("actor_type", "person")
		}

		// A refreshed document starts a new count window. Keep an older total
		// if its collection cannot be read. Only reads not yet attempted remain
		// pending; failed attempts wait for the next actor document refresh.
		for _, collection := range []struct {
			url, count, pending string
			page                *pub.OrderedCollectionPage
		}{
			{"followers", "follower_count", "follower_count_pending", followers},
			{"following", "following_count", "following_count_pending", following},
		} {
			dbActor.Set(collection.pending, dbActor.GetString(collection.url) != "" && !includeFollows)
			if collection.page != nil {
				dbActor.Set(collection.count, int(collection.page.TotalItems))
			} else if dbActor.GetString(collection.url) == "" {
				dbActor.Set(collection.count, 0)
			}
		}
	}

	util.SanitizeHTMLFieldsWithLimits(dbActor)
	err := app.Save(dbActor)
	if err != nil {
		return nil, err
	}

	if private {
		return dbActor, ErrProfilePrivate
	}

	return dbActor, nil
}

func completeActorFollowCounts(app core.App, ctx context.Context, actor *core.Record) (*core.Record, error) {
	for _, collection := range []struct{ url, count, pending string }{
		{"followers", "follower_count", "follower_count_pending"},
		{"following", "following_count", "following_count_pending"},
	} {
		iri := actor.GetString(collection.url)
		if iri == "" || !actor.GetBool(collection.pending) {
			continue
		}

		// Claim the attempt before fetching, without holding a transaction or
		// saving an actor snapshot over concurrent identity or count updates.
		current := dbx.HashExp{
			"id": actor.Id, "is_local": false,
			"last_fetched": actor.GetString("last_fetched"),
			collection.url: iri, collection.pending: true,
		}
		result, err := app.DB().Update("activitypub_actors", dbx.Params{collection.pending: false}, current).WithContext(ctx).Execute()
		if err != nil {
			return actor, err
		}
		claimed, err := result.RowsAffected()
		if err != nil {
			return actor, err
		}
		if claimed == 0 {
			continue
		}

		if page, err := FetchCollection(app, ctx, iri); err == nil {
			// Discard results superseded by a document refresh or another count
			// update. Persist the completed read even if its caller has cancelled.
			current[collection.pending] = false
			current[collection.count] = actor.GetInt(collection.count)
			if _, err := app.DB().Update("activitypub_actors", dbx.Params{collection.count: int(page.TotalItems)}, current).WithContext(context.WithoutCancel(ctx)).Execute(); err != nil {
				return actor, err
			}
		}
	}
	return app.FindRecordById("activitypub_actors", actor.Id)
}

// Fetches an AP actor and optionally followers/following collections
func fetchRemoteActor(app core.App, ctx context.Context, iri string, includeFollows bool) (*pub.Actor, *pub.OrderedCollectionPage, *pub.OrderedCollectionPage, error) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, nil, nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	// Preserve the caller's client and redirect policy; a fetched identity
	// must continue to be served by its original host, including its port.
	client := *newActorFetchHTTPClient()
	previousRedirect := client.CheckRedirect
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("actor fetch exceeded redirect limit")
		}
		if next.URL.User != nil || (next.URL.Scheme != "http" && next.URL.Scheme != "https") || !strings.EqualFold(next.URL.Host, via[0].URL.Host) {
			return fmt.Errorf("actor fetch redirected outside its origin host")
		}
		if via[len(via)-1].URL.Scheme == "https" && next.URL.Scheme == "http" {
			return fmt.Errorf("actor fetch redirected from HTTPS to HTTP")
		}
		if previousRedirect != nil {
			return previousRedirect(next, via)
		}
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", iri, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	headers := map[string]string{
		"Accept":       "application/ld+json",
		"Content-Type": "application/activity+json",
		"Date":         strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"),
		"Host":         req.Host,
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	actorVal, _ := ctx.Value("actor").(string)
	userActorId := strings.TrimPrefix(actorVal, "actor:")
	userActor, err := app.FindRecordById("activitypub_actors", userActorId)
	if userActor != nil && userActor.GetString("private_key") != "" {
		dbPrivateKey := userActor.GetString("private_key")

		algs := []httpsig.Algorithm{httpsig.RSA_SHA256}
		postHeaders := []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}
		expiresIn := 60

		signer, _, err := httpsig.NewSigner(algs, httpsig.DigestSha256, postHeaders, httpsig.Signature, int64(expiresIn))
		if err != nil {
			return nil, nil, nil, err
		}

		decryptedPrivateKey, err := security.Decrypt(dbPrivateKey, encryptionKey)
		if err != nil {
			return nil, nil, nil, err
		}
		privateKey, err := x509.ParsePKCS1PrivateKey(decryptedPrivateKey)
		if err != nil {
			return nil, nil, nil, err
		}

		pubID := userActor.GetString("iri") + "#main-key"

		if err := signer.SignRequest(privateKey, pubID, req, []byte{}); err != nil {
			return nil, nil, nil, err
		}

	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("actor fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return nil, nil, nil, ErrActorNotFound
		}
		return nil, nil, nil, fmt.Errorf("actor fetch failed: status %v", resp.StatusCode)
	}

	var pubActor pub.Actor
	if err := util.DecodeFederationJSON(resp.Body, util.FederationActorBodyLimit, &pubActor); err != nil {
		return nil, nil, nil, err
	}

	// Validate actor response has required fields
	if err := validateActorResponse(&pubActor); err != nil {
		return nil, nil, nil, fmt.Errorf("actor validation failed for %s: %w", iri, err)
	}
	if err := validateFetchedActorIdentity(iri, &pubActor); err != nil {
		return nil, nil, nil, err
	}

	// A nil result records an unread or failed collection, distinct from zero.
	var followers, following *pub.OrderedCollectionPage

	if includeFollows {
		if iri := util.ItemID(pubActor.Followers); iri != "" {
			followers, _ = FetchCollection(app, ctx, iri)
		}

		if iri := util.ItemID(pubActor.Following); iri != "" {
			following, _ = FetchCollection(app, ctx, iri)
		}
	}

	return &pubActor, followers, following, nil
}

func validateFetchedActorIdentity(requested string, actor *pub.Actor) error {
	documentIRI := actor.GetID().String()
	if !actorIdentityMatchesDocument(requested, documentIRI) {
		return fmt.Errorf("actor document does not match requested identity")
	}
	if actor.PublicKey.Owner.String() != documentIRI {
		return fmt.Errorf("actor public key owner does not match identity")
	}
	if !util.SignatureKeyIsOnActorHost(actor.PublicKey.ID.String(), requested) {
		return fmt.Errorf("actor public key id is not on the actor's host")
	}
	return nil
}

// A scheme upgrade changes only HTTP to HTTPS on the exact same host, port,
// path and query. Downgrades and URL aliases do not establish identity.
func actorIdentityMatchesDocument(requested, document string) bool {
	parsed, err := url.Parse(requested)
	if err != nil || parsed.User != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	return requested == document || (parsed.Scheme == "http" && document == "https://"+strings.TrimPrefix(requested, "http://"))
}

// Older caches stored the response ID alongside the requested host. A
// mismatch requires a fresh lookup before the record can authenticate.
func CachedActorIdentityMatchesOrigin(actor *core.Record) bool {
	if actor == nil || strings.TrimSpace(actor.GetString("public_key")) == "" {
		return false
	}
	parsed, err := url.Parse(actor.GetString("iri"))
	domain := strings.TrimSpace(actor.GetString("domain"))
	return err == nil && parsed.User == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != "" && domain != "" && strings.EqualFold(parsed.Host, domain)
}

// newHTTPClient is replaced in tests.
var newHTTPClient = util.SafeHTTPClient

// FetchCollection fetches a collection or one of its pages.
func FetchCollection(app core.App, ctx context.Context, collectionURL string) (*pub.OrderedCollectionPage, error) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", collectionURL, nil)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"Accept":       "application/ld+json",
		"Content-Type": "application/activity+json",
		"Date":         strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"),
		"Host":         req.Host,
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	actorVal, _ := ctx.Value("actor").(string)
	userActorId := strings.TrimPrefix(actorVal, "actor:")
	userActor, err := app.FindRecordById("activitypub_actors", userActorId)
	if userActor != nil && userActor.GetString("private_key") != "" {
		dbPrivateKey := userActor.GetString("private_key")
		if dbPrivateKey != "" {
			algs := []httpsig.Algorithm{httpsig.RSA_SHA256}
			postHeaders := []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}
			expiresIn := 60

			signer, _, err := httpsig.NewSigner(algs, httpsig.DigestSha256, postHeaders, httpsig.Signature, int64(expiresIn))
			if err != nil {
				return nil, err
			}

			decryptedPrivateKey, err := security.Decrypt(dbPrivateKey, encryptionKey)
			if err != nil {
				return nil, err
			}
			privateKey, err := x509.ParsePKCS1PrivateKey(decryptedPrivateKey)
			if err != nil {
				return nil, err
			}

			pubID := userActor.GetString("iri") + "#main-key"

			if err := signer.SignRequest(privateKey, pubID, req, []byte{}); err != nil {
				return nil, err
			}

		}
	}

	client := newHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("collection fetch failed for %s: %v", collectionURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrProfilePrivate
		}
		return nil, fmt.Errorf("collection fetch %s returned: %v", collectionURL, resp.StatusCode)
	}
	var collection pub.OrderedCollectionPage
	if err := util.DecodeFederationJSON(resp.Body, util.FederationObjectBodyLimit, &collection); err != nil {
		return nil, err
	}

	return &collection, nil
}

// maxCollectionPages bounds the pages FetchCollectionPage walks.
const maxCollectionPages = 50

// FetchCollectionPage returns page n (1-based) of a remote collection. It
// follows the collection's first and next links, as servers page
// differently (Mastodon ?page=N, GoToSocial max_id).
func FetchCollectionPage(app core.App, ctx context.Context, collectionURL string, n int) (*pub.OrderedCollectionPage, error) {
	n = max(n, 1)
	if n > maxCollectionPages {
		return nil, fmt.Errorf("page %d exceeds %d", n, maxCollectionPages)
	}

	page, err := FetchCollection(app, ctx, collectionURL)
	if err != nil {
		return nil, err
	}
	total := page.TotalItems

	// Older wanderer versions answer with the first page itself.
	link, i := page.First, 1
	if page.OrderedItems != nil {
		link, i = page.Next, 2
	}
	for ; i <= n; i++ {
		if link == nil {
			return &pub.OrderedCollectionPage{TotalItems: total}, nil
		}
		if page, err = FetchCollection(app, ctx, collectionLink(collectionURL, link, i)); err != nil {
			return nil, err
		}
		link = page.Next
	}
	page.TotalItems = total
	return page, nil
}

// collectionLink resolves a first or next link. A link off the collection
// falls back to ?page=n; older wanderer versions pointed followers' next at
// the outbox.
func collectionLink(collectionURL string, link pub.Item, n int) string {
	fallback := fmt.Sprintf("%s?page=%d", collectionURL, n)
	base, err := url.Parse(collectionURL)
	if err != nil {
		return fallback
	}
	target, err := base.Parse(link.GetLink().String())
	if err != nil || target.Host != base.Host || target.Path != base.Path {
		return fallback
	}
	return target.String()
}

// CollectionNext returns the cursor for page n+1: the resolved next link of
// page n, or "" on the last page. Only the link's query is kept, so the result
// always passes validateCursor.
func CollectionNext(collectionURL string, page *pub.OrderedCollectionPage, n int) string {
	if page == nil || page.Next == nil || page.Next.GetLink().String() == "" {
		return ""
	}
	base, err := cursorBase(collectionURL)
	if err != nil {
		return ""
	}
	target, err := url.Parse(collectionLink(collectionURL, page.Next, max(n, 1)+1))
	if err != nil || target.RawQuery == "" {
		return base
	}
	return base + "?" + target.RawQuery
}

// FetchCollectionCursor fetches the one page a cursor names, in a single
// request. The cursor is client input sent with the user's signature, so
// unlike collectionLink it gets no fallback: see validateCursor.
func FetchCollectionCursor(app core.App, ctx context.Context, collectionURL, cursor string) (*pub.OrderedCollectionPage, error) {
	if err := validateCursor(collectionURL, cursor); err != nil {
		return nil, err
	}
	return FetchCollection(app, ctx, cursor)
}

// cursorBase returns the collection URL without query or fragment, the
// prefix every cursor must start with.
func cursorBase(collectionURL string) (string, error) {
	base, err := url.Parse(collectionURL)
	if err != nil || !base.IsAbs() || base.Opaque != "" || base.User != nil {
		return "", fmt.Errorf("%w: bad collection url", ErrInvalidCursor)
	}
	base.RawQuery, base.Fragment = "", ""
	return base.String(), nil
}

// validateCursor checks that a cursor stays on the collection's scheme, host
// and path, and only its query may differ.
func validateCursor(collectionURL, cursor string) error {
	base, err := cursorBase(collectionURL)
	if err != nil {
		return err
	}

	// The cursor must be the collection URL itself, optionally followed by a
	// query. Matching the raw string also rules out userinfo, other ports,
	// escaped path tricks and control characters.
	pattern, err := regexp.Compile(`^` + regexp.QuoteMeta(base) + `(\?[^#\s\x00-\x1f\x7f]*)?$`)
	if err != nil {
		return fmt.Errorf("%w: bad collection url", ErrInvalidCursor)
	}
	if !pattern.MatchString(cursor) {
		return fmt.Errorf("%w: leaves the collection", ErrInvalidCursor)
	}
	return nil
}
