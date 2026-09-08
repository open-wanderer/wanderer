package routes

import (
	"encoding/json"
	"net/http/httptest"
	"pocketbase/internal/srch0"
	_ "pocketbase/migrations"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pocketbase/pocketbase/core"
)

func TestSRCH0SearchToken(t *testing.T) {
	d := srch0.Data(t)
	app := srch0.App(t, d)
	for _, c := range srch0.Cases(t, "mutation", "go-search-token") {
		t.Run(c.ID, func(t *testing.T) {
			m := srch0.NewMeili(t)
			e := &core.RequestEvent{App: app}
			e.Request = httptest.NewRequest("GET", "/search/token", nil)
			response := httptest.NewRecorder()
			e.Response = response
			if id, ok := c.Input["user_id"].(string); ok {
				if id == "missing-actor" {
					collection, _ := app.FindCollectionByNameOrId("users")
					e.Auth = core.NewRecord(collection)
					e.Auth.Id = srch0.ID(id)
					e.Auth.Set("username", id)
					e.Auth.SetPassword("srch0-synthetic-password")
					srch0.Save(t, app, e.Auth)
				} else {
					e.Auth = srch0.Record(t, app, "users", id)
				}
			}
			err := SearchToken(m.Client)(e)
			if c.Input["user_id"] == "missing-actor" {
				if err == nil {
					t.Fatal("missing actor unexpectedly issued token")
				}
				srch0.Assert(t, c, map[string]any{"category": "actor-not-found", "token_issued": false}, c.Observed["diagnostics"])
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			token, err := jwt.Parse(body["token"], func(token *jwt.Token) (any, error) { return []byte("synthetic-srch0-search-key"), nil }, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !token.Valid {
				t.Fatalf("invalid issued signature: %v", err)
			}
			claims := token.Claims.(jwt.MapClaims)
			exp, err := claims.GetExpirationTime()
			if err != nil || exp == nil || time.Until(exp.Time) < 23*time.Hour || time.Until(exp.Time) > 25*time.Hour {
				t.Fatalf("invalid expiry: %v", exp)
			}
			srch0.Assert(t, c, map[string]any{"status": response.Code, "search_rules": claims["searchRules"]}, c.Observed["diagnostics"])
		})
	}
}
