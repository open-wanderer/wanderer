package permissions_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
)

func TestTagSearchQuotedValues(t *testing.T) {
	app := newRulesTestApp(t)
	owner := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "tagsearch", "password": "test-password", "email": "tagsearch@example.com",
	})
	token, err := owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"Grüezi 🌍", "O'Brien", "x' || name != '' || name~'", `x" || name != "" || name~"`, `tag\name`}
	ids := map[string]string{}
	for _, name := range names {
		ids[name] = saveRulesTestRecord(t, app, "tags", map[string]any{"name": name}).Id
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		// PocketBase's JS parameter builder produces a JSON-quoted string;
		// exercise that representation against the real backend filter parser.
		query := url.Values{"filter": {"name ~ " + strconv.Quote(name)}}
		request := httptest.NewRequest(http.MethodGet, "/api/collections/tags/records?"+query.Encode(), nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("quoted tag query failed: HTTP %d", response.Code)
		}
		var result struct{ Items []struct{ ID string } }
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 1 || result.Items[0].ID != ids[name] {
			t.Error("quoted tag query matched other names or lost its literal value")
		}
	}
}
