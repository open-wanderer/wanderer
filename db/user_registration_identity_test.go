package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestUserRegistrationGeneratesIdentityAcrossNativeRequestFormats(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "fde406459dc1f6ca6f348e1f44a9a2af")
	f := newRecordHooksFixture(t)
	f.bindHooks()
	f.app.Settings().Batch.Enabled = true
	f.app.Settings().Batch.MaxRequests = 10
	f.app.Settings().Batch.Timeout = 5
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer"}`))
			return
		}
		w.Write([]byte(`{"sub":"oauth-subject","email":"oauth@example.com","preferred_username":"oauth-user"}`))
	}))
	defer provider.Close()
	users, err := f.app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.OAuth2.Enabled = true
	users.OAuth2.Providers = []core.OAuth2ProviderConfig{{Name: "oidc", ClientId: "test", ClientSecret: "test", AuthURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token", UserInfoURL: provider.URL + "/userinfo", PKCE: types.Pointer(false)}}
	if err := f.app.Save(users); err != nil {
		t.Fatal(err)
	}
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string, payload any, format, token string) *httptest.ResponseRecorder {
		var body bytes.Buffer
		contentType := "application/json"
		if format == "multipart" {
			writer := multipart.NewWriter(&body)
			for key, value := range payload.(map[string]any) {
				writer.WriteField(key, value.(string))
			}
			writer.Close()
			contentType = writer.FormDataContentType()
		} else {
			json.NewEncoder(&body).Encode(payload)
		}
		req := httptest.NewRequest(http.MethodPost, path, &body)
		req.Header.Set("Content-Type", contentType)
		if token != "" {
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	data := func(name, id string) map[string]any {
		return map[string]any{"id": id, "username": name, "email": name + "@example.com", "password": "test-password", "passwordConfirm": "test-password"}
	}
	for _, format := range []string{"json", "multipart", "batch", "oauth"} {
		t.Run(format, func(t *testing.T) {
			chosen := format + strings.Repeat("0", 15-len(format))
			name := format + "-user"
			path := "/api/collections/users/records"
			payload := any(data(name, chosen))
			if format == "batch" {
				path = "/api/batch"
				payload = map[string]any{"requests": []any{map[string]any{"method": "POST", "url": "/api/collections/users/records", "body": payload}}}
			}
			if format == "oauth" {
				path = "/api/collections/users/auth-with-oauth2"
				payload = map[string]any{"provider": "oidc", "code": "code", "redirectURL": "https://example.com/oauth", "createData": map[string]any{"id": chosen, "username": "oauth-user"}}
			}
			response := request(path, payload, format, "")
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "validation_not_allowed") {
				t.Fatalf("chosen ID %s was not rejected by registered hook: HTTP%d %s", format, response.Code, response.Body.String())
			}
			if _, err := f.app.FindRecordById("users", chosen); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("rejected identity persisted: %v", err)
			}
		})
	}
	for _, format := range []string{"json", "multipart"} {
		response := request("/api/collections/users/records", data("valid-"+format, ""), format, "")
		if response.Code != http.StatusOK {
			t.Fatalf("generated registration rejected: %d %s", response.Code, response.Body.String())
		}
	}
	response := request("/api/collections/users/auth-with-oauth2", map[string]any{"provider": "oidc", "code": "code", "redirectURL": "https://example.com/oauth", "createData": map[string]any{"username": "oauth-user"}}, "json", "")
	if response.Code != http.StatusOK {
		t.Fatalf("generated OAuth registration rejected: %d %s", response.Code, response.Body.String())
	}
	response = request("/api/collections/users/auth-with-password", map[string]any{"identity": "valid-json@example.com", "password": "test-password"}, "json", "")
	if response.Code != http.StatusOK {
		t.Fatalf("generated user login rejected: %d %s", response.Code, response.Body.String())
	}
	var login struct{ Token string }
	if err := json.Unmarshal(response.Body.Bytes(), &login); err != nil || login.Token == "" {
		t.Fatalf("user token missing: %v", err)
	}
	for _, format := range []string{"json", "multipart", "batch"} {
		t.Run("authenticated/"+format, func(t *testing.T) {
			chosen := "signed" + format + strings.Repeat("0", 9-len(format))
			path := "/api/collections/users/records"
			payload := any(data("signed-"+format, chosen))
			if format == "batch" {
				path = "/api/batch"
				payload = map[string]any{"requests": []any{map[string]any{"method": "POST", "url": "/api/collections/users/records", "body": payload}}}
			}
			response := request(path, payload, format, login.Token)
			if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "validation_not_allowed") {
				t.Fatalf("authenticated chosen ID accepted: HTTP%d %s", response.Code, response.Body.String())
			}
			if _, err := f.app.FindRecordById("users", chosen); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("rejected authenticated identity persisted: %v", err)
			}
		})
	}
	admin := f.save(core.CollectionNameSuperusers, map[string]any{"email": "admin@example.com", "password": "test-password"})
	token, err := admin.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	response = request("/api/collections/users/records", data("admin-created", "adminchosen0001"), "json", token)
	if response.Code != http.StatusOK {
		t.Fatalf("administrative import ID rejected: %d %s", response.Code, response.Body.String())
	}
}
