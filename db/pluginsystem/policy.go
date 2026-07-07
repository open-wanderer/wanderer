package pluginsystem

import (
	"fmt"
	"net/url"
	"path"
	"slices"
	"strings"
)

const (
	ConnectorTypePublicAPI  = "public_api"
	ConnectorTypeConfigured = "configured"
	TLSModeSystem           = "system"
	TLSModeCustomCA         = "customCA"
)

type RequestPolicyContext struct {
	Connectors map[string]ResolvedConnectorTarget
	HostAuth   map[string]any
}

func (p RequestPolicyContext) WithHostAuth(auth map[string]any) RequestPolicyContext {
	p.HostAuth = auth
	return p
}

type ResolvedConnectorTarget struct {
	Name                     string
	Type                     string
	BaseURL                  string
	BasePath                 string
	AllowPrivate             bool
	TLS                      ConnectorTLSConfig
	StorageOrigins           map[string]ResolvedConnectorOrigin
	AllowedPathPrefixes      []string
	Auth                     []string
	SupportsMediaAuth        bool
	SupportsStorageRedirects bool
	SupportsCustomTLS        bool
}

type ConnectorTLSConfig struct {
	Mode     string
	CABundle []byte
}

type ResolvedConnectorOrigin struct {
	Name         string
	BaseURL      string
	BasePath     string
	AllowPrivate bool
	TLS          ConnectorTLSConfig
}

type ResolvedRequestTarget struct {
	URL       *url.URL
	Connector ResolvedConnectorTarget
}

// ValidateHostRequestSpec checks the static manifest policy before the host
// performs any plugin-controlled HTTP request. Provider traffic must use a
// connector target; plugins no longer hand the host absolute API URLs.
func ValidateHostRequestSpec(manifest Manifest, spec HostRequestSpec, policy RequestPolicyContext) error {
	_, err := ValidateAndResolveHostRequestSpec(manifest, spec, policy)
	return err
}

func ValidateAndResolveHostRequestSpec(manifest Manifest, spec HostRequestSpec, policy RequestPolicyContext) (*ResolvedRequestTarget, error) {
	if strings.TrimSpace(spec.Method) == "" {
		return nil, fmt.Errorf("method is required")
	}
	resolved, err := ResolveRequestTarget(manifest, spec.Target, policy)
	if err != nil {
		return nil, err
	}
	if spec.Auth != "" {
		if err := ValidateAuthReference(manifest, spec.Auth); err != nil {
			return nil, err
		}
		if len(resolved.Connector.Auth) > 0 && !slices.Contains(resolved.Connector.Auth, spec.Auth) {
			return nil, fmt.Errorf("auth context %q is not permitted for connector %q", spec.Auth, resolved.Connector.Name)
		}
	}
	if err := validateExpectedResponse(spec.Expect, manifest.Permissions.Downloads); err != nil {
		return nil, err
	}
	return resolved, nil
}

func ValidateAuthReference(manifest Manifest, auth string) error {
	if _, ok := manifest.Auth.Contexts[auth]; !ok {
		return fmt.Errorf("auth context %q is not declared", auth)
	}
	if !slices.Contains(manifest.Permissions.Auth, auth) {
		return fmt.Errorf("auth context %q is not permitted", auth)
	}
	return nil
}

func ResolveRequestTarget(manifest Manifest, target RequestTarget, policy RequestPolicyContext) (*ResolvedRequestTarget, error) {
	if target.Type != "connector" {
		return nil, fmt.Errorf("request target type must be connector")
	}
	connector, ok := policy.Connectors[target.Connector]
	if !ok {
		return nil, fmt.Errorf("connector %q is not configured", target.Connector)
	}
	manifestConnector, ok := manifestConnector(manifest, target.Connector)
	if !ok {
		return nil, fmt.Errorf("connector %q is not declared by manifest", target.Connector)
	}
	connector.AllowedPathPrefixes = canonicalConnectorPrefixes(manifestConnector.AllowedPathPrefixes)
	connector.Auth = manifestConnector.Auth
	connector.SupportsMediaAuth = manifestConnector.SupportsMediaAuth
	connector.SupportsStorageRedirects = manifestConnector.SupportsStorageRedirects
	connector.SupportsCustomTLS = manifestConnector.SupportsCustomTLS

	built, err := BuildConnectorURL(connector, target.Path, target.Query)
	if err != nil {
		return nil, err
	}
	if err := ValidateConnectorURL(connector, built); err != nil {
		return nil, err
	}
	return &ResolvedRequestTarget{URL: built, Connector: connector}, nil
}

func manifestConnector(manifest Manifest, name string) (ConnectorTargetPermission, bool) {
	for _, connector := range manifest.Permissions.Network.Connectors {
		if connector.Name == name {
			return connector, true
		}
	}
	return ConnectorTargetPermission{}, false
}

func BuildConnectorURL(connector ResolvedConnectorTarget, relPath string, query []QueryParam) (*url.URL, error) {
	base, err := url.Parse(connector.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("connector %q has invalid baseURL", connector.Name)
	}
	if base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("connector %q baseURL must not include query or fragment", connector.Name)
	}
	if base.Scheme != "http" && base.Scheme != "https" {
		return nil, fmt.Errorf("connector %q scheme must be http or https", connector.Name)
	}
	base.Path = ""
	base.RawPath = ""

	cleanBase, err := CanonicalURLPath(connector.BasePath)
	if err != nil {
		return nil, fmt.Errorf("connector %q basePath: %w", connector.Name, err)
	}
	cleanRel, err := CanonicalRelativeURLPath(relPath)
	if err != nil {
		return nil, err
	}
	fullPath := joinURLPaths(cleanBase, cleanRel)
	if strings.HasSuffix(cleanRel, "/") && fullPath != "/" {
		fullPath += "/"
	}
	base.Path = fullPath

	encodedQuery := make([]string, 0, len(query))
	for _, param := range query {
		if hasControl(param.Name) || hasControl(param.Value) {
			return nil, fmt.Errorf("query parameters must not contain control characters")
		}
		if param.Name == "" {
			return nil, fmt.Errorf("query parameter name must not be empty")
		}
		encodedQuery = append(encodedQuery, url.QueryEscape(param.Name)+"="+url.QueryEscape(param.Value))
	}
	base.RawQuery = strings.Join(encodedQuery, "&")
	return base, nil
}

func ValidateConnectorURL(connector ResolvedConnectorTarget, candidate *url.URL) error {
	base, err := url.Parse(connector.BaseURL)
	if err != nil {
		return err
	}
	if !strings.EqualFold(candidate.Scheme, base.Scheme) {
		return fmt.Errorf("connector request scheme escaped scope")
	}
	if !strings.EqualFold(candidate.Hostname(), base.Hostname()) {
		return fmt.Errorf("connector request host escaped scope")
	}
	if effectivePort(candidate) != effectivePort(base) {
		return fmt.Errorf("connector request port escaped scope")
	}

	candidatePath, err := CanonicalURLPath(candidate.EscapedPath())
	if err != nil {
		return err
	}
	basePath, err := CanonicalURLPath(connector.BasePath)
	if err != nil {
		return err
	}
	if !pathInPrefix(candidatePath, subtreePrefix(basePath)) {
		return fmt.Errorf("connector request escaped base path")
	}
	prefixes := connector.AllowedPathPrefixes
	if len(prefixes) == 0 {
		prefixes = []string{"/"}
	}
	for _, prefix := range canonicalConnectorPrefixes(prefixes) {
		fullPrefix := subtreePrefix(joinURLPaths(basePath, prefix))
		if pathInPrefix(candidatePath, fullPrefix) {
			return nil
		}
	}
	return fmt.Errorf("connector request path is not allowed")
}

func ValidateConnectorRedirect(connector ResolvedConnectorTarget, initial *url.URL, redirected *url.URL) error {
	if initial.Scheme == "https" && redirected.Scheme == "http" {
		return fmt.Errorf("connector redirect downgrades https to http")
	}
	return ValidateConnectorURL(connector, redirected)
}

func ValidateConnectorStorageRedirect(connector ResolvedConnectorTarget, initial *url.URL, redirected *url.URL) error {
	_, err := ConnectorStorageRedirectOrigin(connector, initial, redirected)
	return err
}

func ConnectorStorageRedirectOrigin(connector ResolvedConnectorTarget, initial *url.URL, redirected *url.URL) (ResolvedConnectorOrigin, error) {
	if !connector.SupportsStorageRedirects {
		return ResolvedConnectorOrigin{}, fmt.Errorf("connector storage redirects are not supported")
	}
	if initial.Scheme == "https" && redirected.Scheme == "http" {
		return ResolvedConnectorOrigin{}, fmt.Errorf("connector storage redirect downgrades https to http")
	}
	for _, origin := range connector.StorageOrigins {
		target := ResolvedConnectorTarget{
			Name:                origin.Name,
			BaseURL:             origin.BaseURL,
			BasePath:            origin.BasePath,
			AllowPrivate:        origin.AllowPrivate,
			TLS:                 origin.TLS,
			AllowedPathPrefixes: []string{"/"},
		}
		if err := ValidateConnectorURL(target, redirected); err == nil {
			return origin, nil
		}
	}
	return ResolvedConnectorOrigin{}, fmt.Errorf("connector storage redirect target is not allowed")
}

func NormalizeConnectorBase(rawURL string, extraBasePath string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("connector baseURL is invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", "", fmt.Errorf("connector baseURL scheme must be http or https")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", "", fmt.Errorf("connector baseURL must not include credentials, query, or fragment")
	}
	basePath := parsed.EscapedPath()
	if extraBasePath != "" {
		basePath = joinURLPaths(basePath, extraBasePath)
	}
	cleanPath, err := CanonicalURLPath(basePath)
	if err != nil {
		return "", "", err
	}
	parsed.Path = ""
	parsed.RawPath = ""
	return parsed.String(), cleanPath, nil
}

func CanonicalRelativeURLPath(rawPath string) (string, error) {
	if strings.TrimSpace(rawPath) == "" {
		return "/", nil
	}
	if strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://") || strings.HasPrefix(rawPath, "//") {
		return "", fmt.Errorf("connector path must be relative")
	}
	cleaned, err := CanonicalURLPath("/" + strings.TrimLeft(rawPath, "/"))
	if err != nil {
		return "", err
	}
	if strings.HasSuffix(rawPath, "/") && cleaned != "/" {
		cleaned += "/"
	}
	return cleaned, nil
}

func CanonicalURLPath(rawPath string) (string, error) {
	if rawPath == "" {
		rawPath = "/"
	}
	if hasControl(rawPath) {
		return "", fmt.Errorf("path must not contain control characters")
	}
	lower := strings.ToLower(rawPath)
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
		return "", fmt.Errorf("encoded path separators are not allowed")
	}
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return "", fmt.Errorf("path has invalid escapes")
	}
	if strings.Contains(decoded, "\\") {
		return "", fmt.Errorf("backslash is not allowed in URL paths")
	}
	if hasDangerousSecondEscape(decoded) {
		return "", fmt.Errorf("ambiguous encoded path is not allowed")
	}
	cleaned := path.Clean("/" + strings.TrimLeft(decoded, "/"))
	if cleaned == "." {
		cleaned = "/"
	}
	return cleaned, nil
}

func canonicalConnectorPrefixes(prefixes []string) []string {
	if len(prefixes) == 0 {
		return nil
	}
	canonical := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		cleaned, err := CanonicalURLPath(prefix)
		if err == nil {
			canonical = append(canonical, cleaned)
		}
	}
	return canonical
}

func joinURLPaths(left string, right string) string {
	if left == "" {
		left = "/"
	}
	if right == "" {
		right = "/"
	}
	joined := path.Join(left, right)
	if joined == "." {
		return "/"
	}
	if !strings.HasPrefix(joined, "/") {
		joined = "/" + joined
	}
	return joined
}

func subtreePrefix(prefix string) string {
	if prefix == "/" {
		return "/"
	}
	return strings.TrimRight(prefix, "/") + "/"
}

func pathInPrefix(candidate string, prefix string) bool {
	if prefix == "/" {
		return true
	}
	candidate = subtreePrefix(candidate)
	return strings.HasPrefix(candidate, prefix)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch u.Scheme {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

func hasControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

func hasDangerousSecondEscape(value string) bool {
	lower := strings.ToLower(value)
	for _, marker := range []string{"%2f", "%5c", "%2e"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
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
