package permissions_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pocketbase/routes"
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestTagExactLookup(t *testing.T) {
	app := newRulesTestApp(t)
	user := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "tag-reader", "password": "test-password", "email": "tag-reader@example.com",
	})
	other := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "other-reader", "password": "test-password", "email": "other-reader@example.com",
	})
	superuser := saveRulesTestRecord(t, app, "_superusers", map[string]any{
		"password": "test-password", "email": "tag-admin@example.com",
	})
	authToken := func(record *core.Record) string {
		t.Helper()
		token, err := record.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	userToken, otherToken, superToken := authToken(user), authToken(other), authToken(superuser)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.POST("/tags/lookup", routes.TagLookup)
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	collection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	newTag := func(id, name, owner string) {
		t.Helper()
		record := core.NewRecord(collection)
		record.Id = id
		record.Set("name", name)
		if owner != "" {
			record.Set("owner", owner)
		}
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := func(t *testing.T) map[string]string {
		t.Helper()
		records, err := app.FindAllRecords("tags")
		if err != nil {
			t.Fatal(err)
		}
		stored := make(map[string]string, len(records))
		for _, record := range records {
			body, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			stored[record.Id] = string(body)
		}
		return stored
	}
	request := func(t *testing.T, token string, body []byte, status int, wantID, wantName string) {
		t.Helper()
		before := snapshot(t)
		req := httptest.NewRequest(http.MethodPost, "/tags/lookup", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != status {
			t.Fatalf("lookup status = %d, want %d: %s", response.Code, status, response.Body.String())
		}
		if status == http.StatusOK {
			var result struct {
				Items []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"items"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Items == nil {
				t.Fatal("lookup must return an items array, including when empty")
			}
			if wantID == "" {
				if len(result.Items) != 0 {
					t.Fatalf("lookup returned an unexpected tag: %v", result.Items)
				}
			} else if len(result.Items) != 1 || result.Items[0].ID != wantID || result.Items[0].Name != wantName {
				t.Fatalf("lookup returned wrong tag: count=%d, want ID=%s", len(result.Items), wantID)
			}
		}
		if !reflect.DeepEqual(before, snapshot(t)) {
			t.Fatal("lookup changed stored tag records")
		}
	}
	nameBody := func(name string) []byte {
		t.Helper()
		body, err := json.Marshal(map[string]string{"name": name})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}

	names := []string{
		"O'Brien", `both ' and " quotes`, `100%_route`, `internal\slash`, `trailing\`, `two\\`,
		`quote\" and trailing\`, `' || id != ''`, "@request.auth.id", "CaseSensitive", "casesensitive",
		"Grüezi e\u0301 👩‍👩‍👧‍👦 \u00a0\u2028", "", "   ", strings.Repeat("🌍", 5000),
	}
	for i, name := range names {
		// Reverse insertion order proves deterministic reuse of the smallest ID.
		id := fmt.Sprintf("a%014d", i)
		newTag(fmt.Sprintf("z%014d", i), name, "")
		newTag(id, name, "")
		t.Run(fmt.Sprintf("literal_%d", i), func(t *testing.T) {
			request(t, userToken, nameBody(name), http.StatusOK, id, name)
		})
	}
	t.Run("absence and case are exact", func(t *testing.T) {
		for _, name := range []string{"missing", "CaseSENSITIVE", "O'", "%"} {
			request(t, userToken, nameBody(name), http.StatusOK, "", "")
		}
	})
	t.Run("invalid or missing names are rejected", func(t *testing.T) {
		for _, body := range [][]byte{nil, []byte(`{}`), []byte(`{"name":null}`), []byte(`{"name":42}`), []byte(`[]`), []byte(`{`)} {
			request(t, userToken, body, http.StatusBadRequest, "", "")
		}
		for _, name := range []string{"raw\ttab", "raw\nnewline", "raw\vVT", "raw\fFF", "raw\rCR", "raw\x00NUL", "raw\x7fDEL", strings.Repeat("🌍", 5001)} {
			request(t, userToken, nameBody(name), http.StatusBadRequest, "", "")
		}
	})
	t.Run("current list rule filters anonymous access", func(t *testing.T) {
		request(t, "", nameBody(names[0]), http.StatusOK, "", "")
	})
	t.Run("nil and empty list rules retain PocketBase access", func(t *testing.T) {
		originalRule := collection.ListRule
		t.Cleanup(func() {
			collection.ListRule = originalRule
			if err := app.Save(collection); err != nil {
				t.Error(err)
			}
		})
		collection.ListRule = nil
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
		request(t, userToken, nameBody(names[0]), http.StatusForbidden, "", "")
		request(t, "", nameBody(names[0]), http.StatusForbidden, "", "")
		request(t, superToken, nameBody(names[0]), http.StatusOK, "a00000000000000", names[0])
		publicRule := ""
		collection.ListRule = &publicRule
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
		request(t, "", nameBody(names[0]), http.StatusOK, "a00000000000000", names[0])
	})
	t.Run("visibility rule applies before ordering and limit", func(t *testing.T) {
		collection.Fields.Add(&core.RelationField{Name: "owner", CollectionId: user.Collection().Id, MaxSelect: 1})
		rule := "owner.username = @request.auth.username"
		collection.ListRule = &rule
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
		const name = "visibility"
		newTag("111111111111111", name, other.Id)
		newTag("333333333333333", name, user.Id)
		newTag("222222222222222", name, user.Id)
		request(t, userToken, nameBody(name), http.StatusOK, "222222222222222", name)
		request(t, otherToken, nameBody(name), http.StatusOK, "111111111111111", name)
		request(t, "", nameBody(name), http.StatusOK, "", "")
		request(t, superToken, nameBody(name), http.StatusOK, "111111111111111", name)
	})
}
