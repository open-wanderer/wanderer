package pluginsystem

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"pocketbase/util"
)

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

func TestExecuteHostRequestBuildsMultipartRouteUpload(t *testing.T) {
	useUnsafeTestHTTPClient(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mediaType := strings.Split(r.Header.Get("Content-Type"), ";")[0]; mediaType != "multipart/form-data" {
			t.Fatalf("unexpected content type %q", r.Header.Get("Content-Type"))
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected file part: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "<gpx />" {
			t.Fatalf("unexpected route body %q", string(data))
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
				Name:   "file",
				Source: MultipartSourceRoute,
			}},
		},
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}, HostRequestOptions{Route: []byte("<gpx />")})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.Status)
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
