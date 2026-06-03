package pluginsystem

import (
	"net/url"
	"testing"
)

func TestValidateHostRequestSpecAcceptsConnectorAndAuthReference(t *testing.T) {
	manifest := hammerheadManifestForTest()
	spec := HostRequestSpec{
		Method: "POST",
		Target: RequestTarget{
			Type:      "connector",
			Connector: "api",
			Path:      "/v1/users/123/routes/import/file",
		},
		Auth: "provider_session",
		Expect: ResponseExpect{
			ContentTypes: []string{"application/json"},
			MaxBytes:     1024,
		},
	}
	manifest.Permissions.Downloads.ContentTypes = append(manifest.Permissions.Downloads.ContentTypes, "application/json")
	manifest.Permissions.Downloads.MaxBytes = 2048

	if err := ValidateHostRequestSpec(manifest, spec, testPolicy()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHostRequestSpecRejectsUnknownConnector(t *testing.T) {
	manifest := hammerheadManifestForTest()
	spec := HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "evil", Path: "/v1"},
	}

	if err := ValidateHostRequestSpec(manifest, spec, testPolicy()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateHostRequestSpecRejectsPathScopeEscape(t *testing.T) {
	manifest := hammerheadManifestForTest()
	spec := HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1-evil"},
	}

	if err := ValidateHostRequestSpec(manifest, spec, testPolicy()); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateHostRequestSpecRejectsLimitExpansion(t *testing.T) {
	manifest := hammerheadManifestForTest()
	manifest.Permissions.Downloads.MaxBytes = 100
	spec := HostRequestSpec{
		Method: "GET",
		Target: RequestTarget{Type: "connector", Connector: "api", Path: "/v1/users"},
		Expect: ResponseExpect{MaxBytes: 101},
	}

	if err := ValidateHostRequestSpec(manifest, spec, testPolicy()); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildConnectorURLPreservesBasePathAndQueryOrder(t *testing.T) {
	target := ResolvedConnectorTarget{
		Name:                "immich",
		BaseURL:             "https://photos.example.test:8443",
		BasePath:            "/immich",
		AllowedPathPrefixes: []string{"/api"},
	}
	u, err := BuildConnectorURL(target, "/api/assets/1/original", []QueryParam{
		{Name: "z", Value: "last"},
		{Name: "key", Value: "a"},
		{Name: "key", Value: "b"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.String() != "https://photos.example.test:8443/immich/api/assets/1/original?z=last&key=a&key=b" {
		t.Fatalf("unexpected url: %s", u.String())
	}
	if err := ValidateConnectorURL(target, u); err != nil {
		t.Fatalf("unexpected scope error: %v", err)
	}
}

func TestBuildConnectorURLPreservesTrailingSlash(t *testing.T) {
	target := ResolvedConnectorTarget{
		Name:                "komoot",
		BaseURL:             "https://api.komoot.de",
		BasePath:            "/",
		AllowedPathPrefixes: []string{"/v006"},
	}
	u, err := BuildConnectorURL(target, "/v006/account/email/user%40example.test/", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.String() != "https://api.komoot.de/v006/account/email/user@example.test/" {
		t.Fatalf("unexpected url: %s", u.String())
	}
}

func TestConnectorPathNormalizationRejectsAmbiguousEscapes(t *testing.T) {
	for _, candidate := range []string{"/api%2fadmin", "/api/%252e%252e/admin", "/api/../admin"} {
		t.Run(candidate, func(t *testing.T) {
			target := ResolvedConnectorTarget{
				Name:                "api",
				BaseURL:             "https://example.test",
				BasePath:            "/",
				AllowedPathPrefixes: []string{"/api"},
			}
			u, err := BuildConnectorURL(target, candidate, nil)
			if err == nil {
				err = ValidateConnectorURL(target, u)
			}
			if err == nil {
				t.Fatal("expected scope error")
			}
		})
	}
}

func TestConnectorStorageRedirectOriginReturnsMatchedOriginPolicy(t *testing.T) {
	connector := ResolvedConnectorTarget{
		Name:                     "immich",
		BaseURL:                  "https://photos.example.test",
		BasePath:                 "/immich",
		SupportsStorageRedirects: true,
		StorageOrigins: map[string]ResolvedConnectorOrigin{
			"minio": {
				Name:         "minio",
				BaseURL:      "https://storage.example.test:9443",
				BasePath:     "/assets",
				AllowPrivate: true,
				TLS:          ConnectorTLSConfig{Mode: TLSModeCustomCA, CABundle: []byte("ca")},
			},
		},
	}
	initial, _ := BuildConnectorURL(connector, "/api/assets/1/original", nil)
	redirected, _ := url.Parse("https://storage.example.test:9443/assets/bucket/photo.jpg")

	origin, err := ConnectorStorageRedirectOrigin(connector, initial, redirected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if origin.Name != "minio" || !origin.AllowPrivate || origin.TLS.Mode != TLSModeCustomCA {
		t.Fatalf("unexpected origin policy: %#v", origin)
	}
}

func testPolicy() RequestPolicyContext {
	return RequestPolicyContext{
		Connectors: map[string]ResolvedConnectorTarget{
			"api": {
				Name:                "api",
				Type:                ConnectorTypePublicAPI,
				BaseURL:             "https://dashboard.hammerhead.io",
				BasePath:            "/",
				AllowedPathPrefixes: []string{"/v1"},
				Auth:                []string{"provider_session"},
			},
		},
	}
}
