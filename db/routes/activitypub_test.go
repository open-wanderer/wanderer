package routes

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// newProcessEvent builds a minimal RequestEvent for exercising the early
// authorization gate in ActivitypubActivityProcess. The gate returns before any
// App/DB access, so a nil App is fine for the reject paths under test.
func newProcessEvent(secretHeader, forwardedPath, body string) *core.RequestEvent {
	req := httptest.NewRequest("POST", "/activitypub/activity/process", strings.NewReader(body))
	if secretHeader != "" {
		req.Header.Set("X-Internal-Secret", secretHeader)
	}
	if forwardedPath != "" {
		req.Header.Set("X-Forwarded-Path", forwardedPath)
	}
	e := &core.RequestEvent{}
	e.Request = req
	e.Response = httptest.NewRecorder()
	return e
}

func assertUnauthorized(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *router.ApiError, got %T: %v", err, err)
	}
	if apiErr.Status != 401 {
		t.Fatalf("expected status 401, got %d", apiErr.Status)
	}
}

func TestActivitypubActivityProcessRejectsWithoutSecret(t *testing.T) {
	t.Setenv("ORIGIN", "http://localhost:3000")

	t.Run("secret not configured", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "")
		e := newProcessEvent("anything", "/api/v1/activitypub/user/alice/inbox", "{}")
		assertUnauthorized(t, ActivitypubActivityProcess(e))
	})

	t.Run("missing secret header", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newProcessEvent("", "/api/v1/activitypub/user/alice/inbox", "{}")
		assertUnauthorized(t, ActivitypubActivityProcess(e))
	})

	t.Run("wrong secret header", func(t *testing.T) {
		t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")
		e := newProcessEvent("nope", "/api/v1/activitypub/user/alice/inbox", "{}")
		assertUnauthorized(t, ActivitypubActivityProcess(e))
	})
}

func TestActivitypubActivityProcessRejectsForgedForwardedPath(t *testing.T) {
	t.Setenv("ORIGIN", "http://localhost:3000")
	t.Setenv("POCKETBASE_PROXY_SECRET", "s3cret")

	for _, path := range []string{
		"",
		"/etc/passwd",
		"/api/v1/activitypub/user/alice/outbox",
		"/api/v1/activitypub/user/alice/inbox/../../admin",
		"../api/v1/activitypub/user/alice/inbox",
	} {
		t.Run(path, func(t *testing.T) {
			e := newProcessEvent("s3cret", path, "{}")
			assertUnauthorized(t, ActivitypubActivityProcess(e))
		})
	}
}
