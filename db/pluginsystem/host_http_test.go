package pluginsystem

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"pocketbase/util"
)

func TestParseHostLogEntry(t *testing.T) {
	payload, err := json.Marshal(HostLogEntry{Level: "warn", Message: "  slow request  "})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := parseHostLogEntry(payload)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Level != "warn" || entry.Message != "slow request" {
		t.Fatalf("unexpected structured entry: %#v", entry)
	}

	if _, err := parseHostLogEntry([]byte(" plain message ")); err == nil {
		t.Fatal("expected plain log message to fail")
	}

	if _, err := parseHostLogEntry([]byte(`{"level":"verbose","message":"hello"}`)); err == nil {
		t.Fatal("expected unsupported log level to fail")
	}

	if _, err := parseHostLogEntry([]byte(`{"level":"info","message":"  "}`)); err == nil {
		t.Fatal("expected empty log message to fail")
	}
}

func TestParseHostLogEntrySanitizesMessage(t *testing.T) {
	entry, err := parseHostLogEntry([]byte(`{"level":"info","message":"first\nsecond\rthird\tfourth"}`))
	if err != nil {
		t.Fatal(err)
	}
	if entry.Message != "first second third fourth" {
		t.Fatalf("unexpected sanitized message: %q", entry.Message)
	}
}

func TestParseHostLogEntryRejectsOversizedPayload(t *testing.T) {
	payload := []byte(`{"level":"info","message":"` + strings.Repeat("x", maxHostLogPayloadBytes) + `"}`)
	if _, err := parseHostLogEntry(payload); err == nil {
		t.Fatal("expected oversized log payload to fail")
	}
}

func TestExecuteHostRequestRejectsRedirectToUndeclaredHost(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.test/v1/upload", http.StatusFound)
	}))
	defer server.Close()

	_, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/v1",
		},
	}, HostRequestOptions{})
	if err == nil {
		t.Fatal("expected redirect policy error")
	}
}

func TestExecuteHostRequestRejectsRedirectOutsidePathScope(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusFound)
	}))
	defer server.Close()

	_, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1"},
	}, HostRequestOptions{})
	if err == nil {
		t.Fatal("expected redirect policy error")
	}
}

func TestExecuteHostRequestEnforcesResponseLimit(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"too":"large"}`))
	}))
	defer server.Close()

	_, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1"},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     4,
		},
	}, HostRequestOptions{})
	if err == nil {
		t.Fatal("expected maxBytes error")
	}
}

func TestExecuteHostRequestAllowsErrorResponseWithoutContentType(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`missing credentials`))
	}))
	defer server.Close()

	resp, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1"},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}, HostRequestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusUnauthorized || string(resp.Body) != "missing credentials" {
		t.Fatalf("unexpected response: %#v body=%q", resp, string(resp.Body))
	}
}

func TestExecuteHostRequestInjectsAPIKeyQueryBeforeBuildingURL(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("api_key"); got != "host-secret" {
			t.Fatalf("api_key = %q, want host-secret; raw query %q", got, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	manifest := testHostManifest(t, server.URL)
	manifest.Auth = AuthManifest{Contexts: map[string]AuthContext{
		"account": {
			Type:        AuthTypeAPIKey,
			SecretField: "apiKey",
			Placement:   AuthPlacementQuery,
			Name:        "api_key",
		},
	}}
	manifest.Permissions.Auth = []string{"account"}
	manifest.Permissions.Network.Connectors[0].Auth = []string{"account"}
	policy := testHostPolicy(t, server.URL).WithHostAuth(map[string]any{"apiKey": "host-secret"})
	policy.Connectors["api"] = ResolvedConnectorTarget{
		Name:                "api",
		Type:                ConnectorTypePublicAPI,
		BaseURL:             policy.Connectors["api"].BaseURL,
		BasePath:            "/",
		AllowPrivate:        true,
		AllowedPathPrefixes: []string{"/v1"},
		Auth:                []string{"account"},
	}

	resp, err := ExecuteHostRequest(context.Background(), manifest, policy, HostRequestSpec{
		Method: "GET",
		Auth:   "account",
		Target: RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/v1",
			Query:     []QueryParam{{Name: "existing", Value: "1"}},
		},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}, HostRequestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.Status)
	}
}

func TestExecuteHostRequestBuildsMultipartTrailSend(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mediaType := strings.Split(r.Header.Get("Content-Type"), ";")[0]; mediaType != "multipart/form-data" {
			t.Fatalf("unexpected content type %q", r.Header.Get("Content-Type"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected file part: %v", err)
		}
		defer file.Close()
		if header.Filename != "My Route.gpx" {
			t.Fatalf("unexpected filename %q", header.Filename)
		}
		data, _ := io.ReadAll(file)
		if string(data) != "<gpx />" {
			t.Fatalf("unexpected trail body %q", string(data))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	resp, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "POST",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1/upload"},
		Body: &HostRequestBody{
			Type: HostRequestBodyTypeMultipart,
			Parts: []MultipartPart{{
				Name:     "file",
				Source:   MultipartSourceTrail,
				Filename: "My Route.gpx",
			}},
		},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}, HostRequestOptions{Trail: []byte("<gpx />")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.Status)
	}
}

func TestExecuteHostRequestBuildsFormURLEncodedBody(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("unexpected content type %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("person[login_identity]"); got != "user@example.test" {
			t.Fatalf("login_identity = %q", got)
		}
		if got := r.Form.Get("person[password]"); got != "secret" {
			t.Fatalf("password = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	manifest := testHostManifest(t, server.URL)
	manifest.Permissions.Uploads.ContentTypes = append(manifest.Permissions.Uploads.ContentTypes, "application/x-www-form-urlencoded")
	resp, err := ExecuteHostRequest(context.Background(), manifest, testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "POST",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1/login"},
		Body: &HostRequestBody{
			Type: HostRequestBodyTypeForm,
			Form: []FormField{
				{Name: "person[login_identity]", Value: "user@example.test"},
				{Name: "person[password]", Value: "secret"},
			},
		},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}, HostRequestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.Status)
	}
}

func TestExecuteHostRequestCanReturnRedirectResponse(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v1/next", http.StatusFound)
	}))
	defer server.Close()

	followRedirects := false
	resp, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method:          "GET",
		Target:          RequestTarget{Type: "connector", Connector: "api", Path: "/v1/start"},
		FollowRedirects: &followRedirects,
	}, HostRequestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusFound {
		t.Fatalf("unexpected status %d", resp.Status)
	}
	if got := resp.HeaderValues["Location"]; len(got) != 1 || got[0] != "/v1/next" {
		t.Fatalf("Location = %#v", got)
	}
}

func TestExecuteHostRequestReturnsMultiValueHeaders(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "session=abc; Path=/")
		w.Header().Add("Set-Cookie", "device=full; Path=/")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	resp, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), testHostPolicy(t, server.URL), HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1"},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}, HostRequestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := resp.HeaderValues["Set-Cookie"]; len(got) != 2 || got[0] != "session=abc; Path=/" || got[1] != "device=full; Path=/" {
		t.Fatalf("Set-Cookie values = %#v", got)
	}
}

func useUnsafeTestHTTPClient(t *testing.T) {
	t.Helper()
	original := newConnectorHTTPClient
	newConnectorHTTPClient = func(policy util.ConnectorHTTPPolicy, checkRedirect func(req *http.Request, via []*http.Request) error) (*http.Client, error) {
		return &http.Client{
			Timeout:       60 * time.Second,
			CheckRedirect: checkRedirect,
		}, nil
	}
	t.Cleanup(func() {
		newConnectorHTTPClient = original
	})
}

func testHostManifest(t *testing.T, rawURL string) Manifest {
	t.Helper()
	return Manifest{
		ManifestVersion: ManifestVersion,
		ID:              "test",
		Type:            PluginTypeTrails,
		Name:            "Test",
		Version:         "0.1.0",
		Runtime: RuntimeManifest{
			Type:       RuntimeWASM,
			Entrypoint: "plugin.wasm",
		},
		Capabilities: []CapabilityManifest{{
			Name:    "test",
			Version: "v1",
			Export:  "test_v1",
		}},
		Permissions: PermissionManifest{
			Network: NetworkPermissions{
				Connectors: []ConnectorTargetPermission{{
					Name:                "api",
					Type:                ConnectorTypePublicAPI,
					FixedBaseURL:        rawURL,
					AllowedPathPrefixes: []string{"/v1"},
				}},
			},
			Downloads: DownloadPermissions{
				MaxBytes:     1024,
				ContentTypes: []string{"application/json"},
			},
			Uploads: UploadPermissions{
				MaxBytes:     1024,
				ContentTypes: []string{"multipart/form-data"},
			},
		},
	}
}

func testHostPolicy(t *testing.T, rawURL string) RequestPolicyContext {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Path = ""
	return RequestPolicyContext{Connectors: map[string]ResolvedConnectorTarget{
		"api": {
			Name:                "api",
			Type:                ConnectorTypePublicAPI,
			BaseURL:             parsed.String(),
			BasePath:            "/",
			AllowPrivate:        true,
			AllowedPathPrefixes: []string{"/v1"},
		},
	}}
}
