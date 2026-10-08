package federation

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
var ErrInvalidActorResponse = errors.New("invalid or incomplete actor response")
var ErrInvalidCursor = errors.New("invalid collection cursor")

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

	filter := "preferred_username={:username}&&"
	if domain != "" {
		filter += "domain={:domain}"
	} else {
		filter += "is_local=true"
	}

	var dbActor *core.Record
	dbActor, err := app.FindFirstRecordByFilter("activitypub_actors", filter, dbx.Params{"username": username, "domain": domain})
	if err != nil && err == sql.ErrNoRows {
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

	return assembleActor(app, ctx, dbActor, includeFollows || dbActor.Id == "")
}

func iriFromHandle(ctx context.Context, domain string, username string) (string, error) {
	client := util.SafeHTTPClient()

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
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	limitedReader := io.LimitReader(resp.Body, 102400)

	var wf WebfingerResponse
	if err := json.NewDecoder(limitedReader).Decode(&wf); err != nil {
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

		// check if value is still cached
		twoHoursAgo := time.Now().UTC().Add(-2 * time.Hour)
		if dbActor.GetDateTime("last_fetched").Time().After(twoHoursAgo) {
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
		domain := strings.TrimPrefix(parsedUrl.Hostname(), "www.")

		// this is a race condition that gets triggered when the profile is opened for the first time
		existingActor, _ := app.FindFirstRecordByData("activitypub_actors", "iri", dbActor.GetString("iri"))

		if existingActor != nil {
			dbActor = existingActor
		}

		dbActor.Set("domain", domain)
		dbActor.Set("followers", util.ItemID(pubActor.Followers))
		dbActor.Set("inbox", util.ItemID(pubActor.Inbox))
		dbActor.Set("iri", pubActor.GetID().String())
		dbActor.Set("username", pubActor.Name.String())
		dbActor.Set("preferred_username", pubActor.PreferredUsername.String())
		dbActor.Set("following", util.ItemID(pubActor.Following))
		dbActor.Set("summary", pubActor.Summary.String())
		dbActor.Set("outbox", util.ItemID(pubActor.Outbox))
		dbActor.Set("icon", icon)
		dbActor.Set("published", pubActor.Published.String())
		dbActor.Set("public_key", pubActor.PublicKey.PublicKeyPem)
		dbActor.Set("last_fetched", time.Now())

		if includeFollows {
			dbActor.Set("follower_count", int(followers.TotalItems))
			dbActor.Set("following_count", int(following.TotalItems))
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

// Fetches an AP actor and optionally followers/following collections
func fetchRemoteActor(app core.App, ctx context.Context, iri string, includeFollows bool) (*pub.Actor, *pub.OrderedCollectionPage, *pub.OrderedCollectionPage, error) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, nil, nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	client := util.SafeHTTPClient()

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
		return nil, nil, nil, fmt.Errorf("actor fetch failed: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		return nil, nil, nil, fmt.Errorf("actor fetch failed: status %v", resp.StatusCode)
	}

	defer resp.Body.Close()

	var pubActor pub.Actor
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&pubActor); err != nil {
		return nil, nil, nil, err
	}

	// Validate actor response has required fields
	if err := validateActorResponse(&pubActor); err != nil {
		return nil, nil, nil, fmt.Errorf("actor validation failed for %s: %w", iri, err)
	}

	var followers, following pub.OrderedCollectionPage

	if includeFollows {
		// Fetch followers
		if data, err := FetchCollection(app, ctx, util.ItemID(pubActor.Followers)); err == nil {
			followers = *data
		}

		// Fetch following
		if data, err := FetchCollection(app, ctx, util.ItemID(pubActor.Following)); err == nil {
			following = *data
		}
	}

	return &pubActor, &followers, &following, nil
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
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrProfilePrivate
		}
		return nil, fmt.Errorf("collection fetch %s returned: %v", collectionURL, resp.StatusCode)
	}
	defer resp.Body.Close()

	var collection pub.OrderedCollectionPage
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
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
