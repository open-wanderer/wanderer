package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/security"
)

// The web hook turns a 401 from /auth/token into a 401 for the API client and
// anything else into a 500, so every rejected token must answer 401 here.

func setupAuthTokenTest(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.SetEmail("alice@example.com")
	user.SetPassword("password123")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}

	tokens := core.NewBaseCollection("api_tokens")
	tokens.Fields.Add(
		&core.TextField{Name: "token"},
		&core.DateField{Name: "expiration"},
		&core.DateField{Name: "last_used"},
		&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1},
	)
	if err := app.Save(tokens); err != nil {
		t.Fatal(err)
	}

	for _, token := range []struct {
		value      string
		expiration time.Time
	}{
		{"wanderer_key_valid", time.Time{}},
		{"wanderer_key_future", time.Now().Add(time.Hour)},
		{"wanderer_key_expired", time.Now().Add(-time.Hour)},
	} {
		record := core.NewRecord(tokens)
		record.Set("token", security.SHA256(token.value))
		record.Set("user", user.Id)
		if !token.expiration.IsZero() {
			record.Set("expiration", token.expiration)
		}
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}

	return app
}

func postAuthToken(t *testing.T, app core.App, body string) int {
	t.Helper()

	e := &core.RequestEvent{App: app}
	e.Request = httptest.NewRequest(http.MethodPost, "/auth/token", strings.NewReader(body))
	e.Request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	e.Response = response
	if err := AuthToken(e); err != nil {
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatal(err)
		}
		return apiErr.Status
	}
	return response.Code
}

func TestAuthTokenStatus(t *testing.T) {
	app := setupAuthTokenTest(t)

	for _, tt := range []struct {
		name, body string
		want       int
	}{
		{"valid token", `{"api_token":"wanderer_key_valid"}`, http.StatusOK},
		{"token expiring later", `{"api_token":"wanderer_key_future"}`, http.StatusOK},
		{"unknown or revoked token", `{"api_token":"wanderer_key_bogus"}`, http.StatusUnauthorized},
		{"expired token", `{"api_token":"wanderer_key_expired"}`, http.StatusUnauthorized},
		{"unreadable body", `{`, http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := postAuthToken(t, app, tt.body); got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}
