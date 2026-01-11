package federation

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

		body, err := jsonld.WithContext(
			jsonld.IRI(pub.ActivityBaseURI),
			jsonld.IRI(pub.SecurityContextURI),
		).Marshal(activity)
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

		client := &http.Client{}
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

				resp, err := client.Do(req)
				if err != nil {
					app.Logger().Error(fmt.Sprintf("Error sending to inbox %s: %s", inbox, err))
					return
				}
				defer func() {
					if closeErr := resp.Body.Close(); closeErr != nil {
						app.Logger().Warn(fmt.Sprintf("Closing inbox response failed: %s", closeErr))
					}
				}()

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

func ProcessActivity(e *core.RequestEvent) error {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return fmt.Errorf("ORIGIN not set")
	}

	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}
	var activity pub.Activity
	if err := activity.UnmarshalJSON(body); err != nil {
		return err
	}

	inbox := fmt.Sprintf("%s%s", origin, e.Request.Header.Get("X-Forwarded-Path"))

	recipient, err := e.App.FindFirstRecordByData("activitypub_actors", "inbox", inbox)
	if err != nil {
		return err
	}

	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "iri", activity.Actor.GetID().String())
	if err != nil {
		if err == sql.ErrNoRows {
			actor, err = GetActorByIRI(e.App, recipient, activity.Actor.GetID().String(), false)
			if err != nil {
				return err
			}
		} else {
			return err

		}
	}

	verified, err := verifySignature(e.App, e.Request, actor.GetString("public_key"))
	if err != nil || !verified {
		e.App.Logger().Error(err.Error())
		return e.UnauthorizedError("Invalid http signature", err)
	}

	switch activity.Type {
	case pub.FollowType:
		err = ProcessFollowActivity(e.App, actor, activity)
	case pub.AcceptType:
		err = ProcessAcceptActivity(e.App, actor, activity)
	case pub.UndoType:
		err = ProcessUndoActivity(e.App, actor, activity)
	case pub.UpdateType:
		fallthrough
	case pub.CreateType:
		err = ProcessCreateOrUpdateActivity(e.App, actor, recipient, activity)
	case pub.DeleteType:
		err = ProcessDeleteActivity(e.App, actor, activity)
	case pub.AnnounceType:
		err = ProcessAnnounceActivity(e.App, actor, activity)
	case pub.LikeType:
		err = ProcessLikeActivity(e.App, actor, activity)
	}
	return e.JSON(http.StatusOK, err)
}

func verifySignature(app core.App, req *http.Request, publicKeyPem string) (bool, error) {
	origin := os.Getenv("ORIGIN")
	if origin == "" {
		return false, fmt.Errorf("ORIGIN not set")
	}
	block, _ := pem.Decode([]byte(publicKeyPem))
	if block == nil || block.Type != "PUBLIC KEY" {
		return false, fmt.Errorf("could not decode publicKeyPem to PUBLIC KEY pem block type")
	}

	req.URL = &url.URL{
		Path: req.Header.Get("X-Forwarded-Path"),
	}

	url, err := url.Parse(origin)
	if err != nil {
		return false, err
	}

	req.Header.Set("Host", url.Host)
	req.Host = url.Host

	app.Logger().Info(req.Header.Get("signature"))

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}

	v, err := httpsig.NewVerifier(req)
	if err != nil {
		return false, err
	}

	err = v.Verify(publicKey, httpsig.RSA_SHA256)
	if err != nil {
		return false, err
	}

	return true, nil
}
