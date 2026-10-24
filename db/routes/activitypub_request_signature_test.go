package routes

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

type inboxRequestTimes struct {
	date             time.Time
	created, expires *time.Time
}

// Build the signed string explicitly so the tests cover accepted clock offsets
// independently of a signing library's own clock and expiration checks.
func signedInboxRequest(t *testing.T, w *knownKeyWorld, actor *core.Record, key *rsa.PrivateKey, body []byte, times inboxRequestTimes) *http.Request {
	t.Helper()
	path := strings.TrimPrefix(w.recipient.GetString("inbox"), "https://example.com")
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, bytes.NewReader(body))
	request.Header.Set("X-Internal-Secret", "known-key-test-secret")
	request.Header.Set("X-Forwarded-Path", path)
	request.Header.Set("Content-Type", "application/activity+json")
	request.Header.Set("Date", times.date.UTC().Format(http.TimeFormat))
	digest := sha256.Sum256(body)
	request.Header.Set("Digest", "SHA-256="+base64.StdEncoding.EncodeToString(digest[:]))
	headers := []string{"(request-target)", "date", "digest"}
	lines := []string{"(request-target): post " + path, "date: " + request.Header.Get("Date"), "digest: " + request.Header.Get("Digest")}
	var parameters strings.Builder
	if times.created != nil {
		headers = append(headers, "(created)")
		lines = append(lines, fmt.Sprintf("(created): %d", times.created.Unix()))
		fmt.Fprintf(&parameters, ",created=%d", times.created.Unix())
	}
	if times.expires != nil {
		headers = append(headers, "(expires)")
		lines = append(lines, fmt.Sprintf("(expires): %d", times.expires.Unix()))
		fmt.Fprintf(&parameters, ",expires=%d", times.expires.Unix())
	}
	signed := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, signed[:])
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Signature", fmt.Sprintf("keyId=%q,algorithm=\"hs2019\",headers=%q%s,signature=%q", actor.GetString("iri")+"#main-key", strings.Join(headers, " "), parameters.String(), base64.StdEncoding.EncodeToString(signature)))
	// The internal route receives the original body with the signed public path
	// forwarded separately by the web frontend.
	request.URL.Path = "/inbox"
	return request
}

func inboxRecordSnapshot(t *testing.T, app core.App) map[string]string {
	t.Helper()
	snapshot := make(map[string]string)
	for _, collection := range []string{"activitypub_actors", "activitypub_activities", "follows", "notifications", "trails", "lists", "comments", "summit_logs", "waypoints", "trail_like"} {
		records, err := app.FindAllRecords(collection)
		if err != nil {
			t.Fatal(err)
		}
		sort.Slice(records, func(i, j int) bool { return records[i].Id < records[j].Id })
		fields := make([]map[string]any, 0, len(records))
		for _, record := range records {
			fields = append(fields, record.FieldsData())
		}
		encoded, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		snapshot[collection] = string(encoded)
	}
	return snapshot
}

func waitForSignatureTestDeliveries(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := federation.WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestInboxChecksExactBodyAndSignatureDate(t *testing.T) {
	for _, scenario := range []string{"current", "changed-body", "expired-date"} {
		t.Run(scenario, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			actor := w.sender(t, "sender", "remote.example", false)
			actor.Set("public_key", w.publicKey)
			if err := w.app.SaveNoValidate(actor); err != nil {
				t.Fatal(err)
			}
			handler := knownKeyInboxRouter(t, w)
			// Whitespace and field order intentionally differ from json.Marshal.
			body := []byte(fmt.Sprintf("{\n \"object\": %q, \"actor\": %q,\n \"id\": \"https://remote.example/activities/follow\", \"type\": \"Follow\"\n}\n", w.recipient.GetString("iri"), actor.GetString("iri")))
			date := time.Now().UTC()
			if scenario == "expired-date" {
				date = date.Add(-2 * time.Hour)
			}
			request := signedInboxRequest(t, w, actor, w.key, body, inboxRequestTimes{date: date})
			if scenario == "changed-body" {
				changed := append(append([]byte(nil), body...), ' ')
				request.Body = io.NopCloser(bytes.NewReader(changed))
				request.ContentLength = int64(len(changed))
			}
			before := inboxRecordSnapshot(t, w.app)
			status := knownKeyRouterStatus(handler, request)
			waitForSignatureTestDeliveries(t)
			if scenario == "current" {
				if status != http.StatusOK {
					t.Fatalf("valid signed delivery: %d", status)
				}
				follow, err := w.app.FindFirstRecordByData("follows", "follower", actor.Id)
				if err != nil || follow.GetString("followee") != w.recipient.Id || follow.GetString("status") != "accepted" {
					t.Fatalf("valid follow was not applied: follow=%v error=%v", follow, err)
				}
			} else {
				if status != http.StatusUnauthorized {
					t.Fatalf("invalid signed delivery: %d, want 401", status)
				}
				if after := inboxRecordSnapshot(t, w.app); !reflect.DeepEqual(after, before) {
					t.Fatalf("refused delivery changed stored records: before=%v after=%v", before, after)
				}
			}
			if requests := w.origin.count("/api/v1/activitypub/user/sender"); requests != 0 {
				t.Fatalf("cached sender was fetched %d times", requests)
			}
		})
	}
}

func TestInboxAcceptsSignatureParametersWithinClockWindow(t *testing.T) {
	for _, scenario := range []string{"future-created", "past-expires"} {
		t.Run(scenario, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			actor := w.sender(t, "sender", "remote.example", false)
			actor.Set("public_key", w.publicKey)
			if err := w.app.SaveNoValidate(actor); err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC().Truncate(time.Second)
			created := now.Add(30 * time.Minute)
			times := inboxRequestTimes{date: now, created: &created}
			if scenario == "past-expires" {
				created = now.Add(-35 * time.Minute)
				expires := now.Add(-30 * time.Minute)
				times.expires = &expires
			}
			body := []byte(fmt.Sprintf(`{"type":"Follow","id":"https://remote.example/activities/clock","actor":%q,"object":%q}`, actor.GetString("iri"), w.recipient.GetString("iri")))
			status := knownKeyRouterStatus(knownKeyInboxRouter(t, w), signedInboxRequest(t, w, actor, w.key, body, times))
			waitForSignatureTestDeliveries(t)
			if status != http.StatusOK {
				t.Fatalf("signature within accepted clock window: %d", status)
			}
			if count, err := w.app.CountRecords("follows"); err != nil || count != 1 {
				t.Fatalf("accepted delivery did not create its follow: count=%d error=%v", count, err)
			}
		})
	}
}

func TestInboxLegacySelfDeletionChecksBodyAndDateBeforeConfirmation(t *testing.T) {
	for _, scenario := range []string{"changed-body", "expired-date"} {
		t.Run(scenario, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			actor := w.sender(t, "sender", "remote.example", true)
			iri, path := actor.GetString("iri"), "/api/v1/activitypub/user/sender"
			w.origin.unavailable[path] = true
			if status := w.deliver(t, iri, w.previousKey); status != http.StatusUnauthorized {
				t.Fatalf("initial unavailable refresh: %d", status)
			}
			w.save(t, "trails", map[string]any{"name": "Existing trail", "author": actor.Id, "iri": "https://remote.example/api/v1/trail/existing"})
			w.save(t, "lists", map[string]any{"name": "Existing list", "author": actor.Id, "iri": "https://remote.example/api/v1/list/existing"})
			w.origin.mu.Lock()
			w.origin.unavailable[path] = false
			delete(w.origin.documents, path)
			w.origin.mu.Unlock()
			previousClient := newRemoteSyncHTTPClient
			newRemoteSyncHTTPClient = func() *http.Client { return w.client }
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			handler := knownKeyInboxRouter(t, w)
			body := []byte(fmt.Sprintf(`{"type":"Delete","id":"https://remote.example/activities/delete","actor":%q,"object":%q}`, iri, iri))
			date := time.Now().UTC()
			if scenario == "expired-date" {
				date = date.Add(-2 * time.Hour)
			}
			request := signedInboxRequest(t, w, actor, w.previousKey, body, inboxRequestTimes{date: date})
			if scenario == "changed-body" {
				changed := append(append([]byte(nil), body...), '\n')
				request.Body = io.NopCloser(bytes.NewReader(changed))
				request.ContentLength = int64(len(changed))
			}
			before := inboxRecordSnapshot(t, w.app)
			if status := knownKeyRouterStatus(handler, request); status != http.StatusUnauthorized {
				t.Fatalf("invalid retained-key self-deletion: %d, want 401", status)
			}
			if after := inboxRecordSnapshot(t, w.app); !reflect.DeepEqual(after, before) {
				t.Fatalf("refused retained-key deletion changed records: before=%v after=%v", before, after)
			}
			if got := w.origin.count(path); got != 1 {
				t.Fatalf("refused deletion requested confirmation: %d origin requests", got)
			}
			// A valid request still owns the one confirmation in this window.
			valid := signedInboxRequest(t, w, actor, w.previousKey, body, inboxRequestTimes{date: time.Now()})
			if status := knownKeyRouterStatus(handler, valid); status != http.StatusOK {
				t.Fatalf("valid retained-key deletion after refusal: %d", status)
			}
			if got := w.origin.count(path); got != 2 {
				t.Fatalf("valid deletion confirmation requests: %d, want 2 total", got)
			}
			if _, err := w.app.FindRecordById("activitypub_actors", actor.Id); err == nil {
				t.Fatal("confirmed retained-key deletion kept the actor")
			}
			for _, collection := range []string{"trails", "lists"} {
				if count, err := w.app.CountRecords(collection); err != nil || count != 0 {
					t.Fatalf("confirmed deletion kept %s: count=%d error=%v", collection, count, err)
				}
			}
		})
	}
}
