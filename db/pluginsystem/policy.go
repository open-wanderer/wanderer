package pluginsystem

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

type RequestPolicyContext struct {
	UserConfiguredOrigins []string
}

// ValidateHostRequestSpec checks the static manifest policy before the host
// performs any plugin-controlled HTTP request. The request may narrow manifest
// limits through expect.*, but it may not add hosts, auth contexts, or larger
// response limits than the manifest permits.
func ValidateHostRequestSpec(manifest Manifest, spec HostRequestSpec, policy RequestPolicyContext) error {
	if strings.TrimSpace(spec.Method) == "" {
		return fmt.Errorf("method is required")
	}
	parsedURL, err := url.Parse(spec.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("url is invalid")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https")
	}
	if !NetworkURLAllowed(parsedURL, manifest.Permissions.Network, policy) {
		return fmt.Errorf("host %q is not allowed by manifest permissions", parsedURL.Hostname())
	}
	if spec.Auth != "" {
		if _, ok := manifest.Auth.Contexts[spec.Auth]; !ok {
			return fmt.Errorf("auth context %q is not declared", spec.Auth)
		}
		if !slices.Contains(manifest.Permissions.Auth, spec.Auth) {
			return fmt.Errorf("auth context %q is not permitted", spec.Auth)
		}
	}
	if err := validateExpectedResponse(spec.Expect, manifest.Permissions.Downloads); err != nil {
		return err
	}
	return nil
}

// NetworkURLAllowed enforces the host allow-list for provider traffic. Static
// hosts come from the manifest; user configured origins are added only by host
// code after reading trusted instance configuration.
func NetworkURLAllowed(parsedURL *url.URL, permissions NetworkPermissions, policy RequestPolicyContext) bool {
	host := strings.ToLower(parsedURL.Hostname())
	for _, allowed := range permissions.StaticHosts {
		if strings.EqualFold(host, allowed) {
			return true
		}
	}
	for _, origin := range policy.UserConfiguredOrigins {
		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			continue
		}
		if strings.EqualFold(host, parsedOrigin.Hostname()) {
			return true
		}
	}
	return false
}

// HostnameFromOrigin normalizes a configured origin into the hostname used for
// policy comparison and storage.
func HostnameFromOrigin(origin string) string {
	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsedOrigin.Hostname())
}

// validateExpectedResponse lets a plugin request stricter response checks for a
// specific call while preventing it from exceeding manifest download limits.
func validateExpectedResponse(expect ResponseExpect, permissions DownloadPermissions) error {
	if expect.MaxBytes < 0 {
		return fmt.Errorf("expect.maxBytes must not be negative")
	}
	if permissions.MaxBytes > 0 && expect.MaxBytes > permissions.MaxBytes {
		return fmt.Errorf("expect.maxBytes exceeds manifest download limit")
	}
	for _, contentType := range expect.ContentTypes {
		if len(permissions.ContentTypes) > 0 && !slices.Contains(permissions.ContentTypes, contentType) {
			return fmt.Errorf("content type %q is not allowed by manifest permissions", contentType)
		}
	}
	return nil
}
