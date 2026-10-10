package federation

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	pub "github.com/go-ap/activitypub"

	"sync"

	"github.com/go-ap/jsonld"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"golang.org/x/sync/semaphore"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// jsonldMu serializes jsonld.WithContext(...).Marshal, which writes the
// package-level jsonld.Ctxt.
var jsonldMu sync.Mutex

// instanceFollowerInboxes returns inbox URLs for all accepted peers of the local
// instance actor — both instances that follow us and instances we follow.
// Returns (nil, nil) if the instance actor has not yet been seeded.
func instanceFollowerInboxes(app core.App) ([]string, error) {
	peers, err := acceptedPeerActors(app)
	if err != nil {
		return nil, err
	}
	if peers == nil {
		return nil, nil
	}
	var inboxes []string
	for _, p := range peers {
		if p.Inbox != "" {
			inboxes = append(inboxes, p.Inbox)
		}
	}
	return inboxes, nil
}

func PostActivity(app core.App, actor *core.Record, activity *pub.Activity, recipients []string) error {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				app.Logger().Error(fmt.Sprintf("Recovered from panic in PostActivity: %v", r))
			}
		}()

		encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
		if len(encryptionKey) == 0 {
			app.Logger().Error("POCKETBASE_ENCRYPTION_KEY not set")
			return
		}
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			app.Logger().Error("ORIGIN not set")
			return
		}

		algs := []httpsig.Algorithm{httpsig.RSA_SHA256}
		postHeaders := []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}
		expiresIn := 60

		// jsonld.WithContext writes a package-level variable.
		jsonldMu.Lock()
		body, err := jsonld.WithContext(
			jsonld.IRI(pub.ActivityBaseURI),
			jsonld.IRI(pub.SecurityContextURI),
		).Marshal(activity)
		jsonldMu.Unlock()
		if err != nil {
			app.Logger().Error(fmt.Sprintf("Failed to marshal activity: %s", err))
			return
		}

		decryptedPrivateKey, err := security.Decrypt(actor.GetString("private_key"), encryptionKey)
		if err != nil {
			app.Logger().Error(fmt.Sprintf("Failed to decrypt key: %s", err))
			return
		}
		privateKey, err := x509.ParsePKCS1PrivateKey(decryptedPrivateKey)
		if err != nil {
			app.Logger().Error(fmt.Sprintf("Failed to parse private key: %s", err))
			return
		}
		pubID := actor.GetString("iri") + "#main-key"

		sem := semaphore.NewWeighted(5)

		slices.Sort(recipients)
		uniqueRecipients := slices.Compact(recipients)

		var wg sync.WaitGroup
		for _, v := range uniqueRecipients {
			wg.Add(1)
			go func(inbox string) {
				defer wg.Done()

				signer, _, err := httpsig.NewSigner(algs, httpsig.DigestSha256, postHeaders, httpsig.Signature, int64(expiresIn))
				if err != nil {
					app.Logger().Error(fmt.Sprintf("Signer creation failed: %s", err))
					return
				}

				if err := sem.Acquire(context.Background(), 1); err != nil {
					app.Logger().Error(fmt.Sprintf("Semaphore acquire failed: %s", err))
					return
				}
				defer sem.Release(1)

				req, err := http.NewRequest(http.MethodPost, inbox, bytes.NewBuffer(body))
				if err != nil {
					app.Logger().Error(fmt.Sprintf("Request creation failed: %s", err))
					return
				}
				req.Header.Add("Content-Type", "application/activity+json")
				req.Header.Add("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
				req.Header.Add("Host", req.Host)
				if err := signer.SignRequest(privateKey, pubID, req, body); err != nil {
					app.Logger().Error(fmt.Sprintf("Signing request failed: %s", err))
					return
				}

				resp, err := httpClient.Do(req)
				if err != nil {
					app.Logger().Error(fmt.Sprintf("Error sending to inbox %s: %s", inbox, err))
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
					respBody, _ := io.ReadAll(resp.Body)
					app.Logger().Error(fmt.Sprintf("Inbox %s responded with %d: %s", inbox, resp.StatusCode, respBody))
				} else {
					app.Logger().Info(fmt.Sprintf("Sent %s to %s", activity.Type, inbox), "activity", activity)
				}
			}(v)
		}
		wg.Wait()
	}()
	return nil
}
