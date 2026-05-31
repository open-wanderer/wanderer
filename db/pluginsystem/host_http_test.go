package pluginsystem

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExecuteHostRequestRejectsRedirectToUndeclaredHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example.test/upload", http.StatusFound)
	}))
	defer server.Close()

	_, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), RequestPolicyContext{}, HostRequestSpec{
		Method: "GET",
		URL:    server.URL,
	}, HostRequestOptions{})
	if err == nil {
		t.Fatal("expected redirect policy error")
	}
}

func TestExecuteHostRequestEnforcesResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"too":"large"}`))
	}))
	defer server.Close()

	_, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), RequestPolicyContext{}, HostRequestSpec{
		Method: "GET",
		URL:    server.URL,
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     4,
		},
	}, HostRequestOptions{})
	if err == nil {
		t.Fatal("expected maxBytes error")
	}
}

func TestExecuteHostRequestBuildsMultipartRouteUpload(t *testing.T) {
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

	resp, err := ExecuteHostRequest(context.Background(), testHostManifest(t, server.URL), RequestPolicyContext{}, HostRequestSpec{
		Method: "POST",
		URL:    server.URL,
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

func testHostManifest(t *testing.T, rawURL string) Manifest {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
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
				StaticHosts: []string{parsed.Hostname()},
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
