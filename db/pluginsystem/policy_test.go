package pluginsystem

import "testing"

func TestValidateHostRequestSpecAcceptsStaticHostAndAuthReference(t *testing.T) {
	manifest := hammerheadManifestForTest()
	spec := HostRequestSpec{
		Method: "POST",
		URL:    "https://dashboard.hammerhead.io/v1/users/123/routes/import/file",
		Auth:   "provider_session",
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}
	manifest.Permissions.Downloads.ContentTypes = append(manifest.Permissions.Downloads.ContentTypes, "application/json")
	manifest.Permissions.Downloads.MaxBytes = 2048

	if err := ValidateHostRequestSpec(manifest, spec, RequestPolicyContext{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHostRequestSpecRejectsUndeclaredHost(t *testing.T) {
	manifest := hammerheadManifestForTest()
	spec := HostRequestSpec{
		Method: "GET",
		URL:    "https://evil.example/api",
	}

	if err := ValidateHostRequestSpec(manifest, spec, RequestPolicyContext{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateHostRequestSpecAllowsReviewedUserOrigin(t *testing.T) {
	manifest := hammerheadManifestForTest()
	spec := HostRequestSpec{
		Method: "GET",
		URL:    "https://photos.example.test/api/search",
	}

	err := ValidateHostRequestSpec(manifest, spec, RequestPolicyContext{
		UserConfiguredOrigins: []string{"https://photos.example.test"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHostRequestSpecRejectsLimitExpansion(t *testing.T) {
	manifest := hammerheadManifestForTest()
	manifest.Permissions.Downloads.MaxBytes = 100
	spec := HostRequestSpec{
		Method: "GET",
		URL:    "https://dashboard.hammerhead.io/api",
		Expect: ResponseExpect{MaxBytes: 101},
	}

	if err := ValidateHostRequestSpec(manifest, spec, RequestPolicyContext{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestHostnameFromOriginNormalizesURL(t *testing.T) {
	got := HostnameFromOrigin("https://Photos.Example.test:8443/path?q=1")
	if got != "photos.example.test" {
		t.Fatalf("got %q", got)
	}
}
