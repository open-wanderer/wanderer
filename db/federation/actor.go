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
		filter += "isLocal=true"
	}

	var dbActor *core.Record
	dbActor, err := app.FindFirstRecordByFilter("activitypub_actors", filter, dbx.Params{"username": username, "domain": domain})
	if err != nil && err == sql.ErrNoRows {
		collection, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			return nil, err
		}

		dbActor = core.NewRecord(collection)
		dbActor.Set("isLocal", false)
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
		dbActor.Set("isLocal", false)
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
	if dbActor.GetBool("isLocal") {
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

		privacy := settings.GetString("privacy")
		result := make(map[string]interface{})
		json.Unmarshal([]byte(privacy), &result)

		// check that it's not our own profile
		private = result["account"] == "private" && dbActor.Id != strings.TrimPrefix(ctx.Value("actor").(string), "actor:")

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
func fetchRemoteActor(app core.App, ctx context.Context, iri string, includeFollows bool) (*pub.Actor, *pub.OrderedCollection, *pub.OrderedCollection, error) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, nil, nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	client := util.SafeHTTPClient()

	req, err := http.NewRequestWithContext(ctx, "GET", iri, nil)

	headers := map[string]string{
		"Accept":       "application/ld+json",
		"Content-Type": "application/activity+json",
		"Date":         strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"),
		"Host":         req.Host,
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	userActorId := strings.TrimPrefix(ctx.Value("actor").(string), "actor:")
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
	if err := json.NewDecoder(resp.Body).Decode(&pubActor); err != nil {
		return nil, nil, nil, err
	}

	// Validate actor response has required fields
	if err := validateActorResponse(&pubActor); err != nil {
		return nil, nil, nil, fmt.Errorf("actor validation failed for %s: %w", iri, err)
	}

	var followers, following pub.OrderedCollection

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

func FetchCollection(app core.App, ctx context.Context, collectionURL string) (*pub.OrderedCollection, error) {
	encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
	if len(encryptionKey) == 0 {
		return nil, fmt.Errorf("POCKETBASE_ENCRYPTION_KEY not set")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", collectionURL, nil)

	headers := map[string]string{
		"Accept":       "application/ld+json",
		"Content-Type": "application/activity+json",
		"Date":         strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"),
		"Host":         req.Host,
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}
	userActorId := strings.TrimPrefix(ctx.Value("actor").(string), "actor:")
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

	client := util.SafeHTTPClient()
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

	var collection pub.OrderedCollection
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, err
	}

	return &collection, nil
}
