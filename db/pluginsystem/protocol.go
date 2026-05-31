package pluginsystem

const (
	ManifestVersion = "1.0"
	RuntimeWASM     = "wasm"

	AuthTypeOAuth2  = "oauth2"
	AuthTypeAPIKey  = "api_key"
	AuthTypeBearer  = "bearer"
	AuthTypeSession = "session"

	AuthRefreshModeHost   = "host"
	AuthRefreshModePlugin = "plugin"

	AuthPlacementQuery         = "query"
	AuthHeaderAuthorization    = "Authorization"
	AuthSchemeBearer           = "Bearer"
	TokenRequestFormatJSON     = "json"
	TokenAuthClientSecretPost  = "client_secret_post"
	TokenAuthClientSecretBasic = "client_secret_basic"

	HostRequestBodyTypeJSON      = "json"
	HostRequestBodyTypeMultipart = "multipart"
	MultipartSourceRoute         = "route"
	MultipartSourceRouteGPX      = "route.gpx"
	MultipartRouteFilename       = "route.gpx"
)

type Manifest struct {
	ManifestVersion string               `json:"manifestVersion"`
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	Version         string               `json:"version"`
	Runtime         RuntimeManifest      `json:"runtime"`
	Capabilities    []CapabilityManifest `json:"capabilities"`
	Auth            AuthManifest         `json:"auth,omitempty"`
	Permissions     PermissionManifest   `json:"permissions,omitempty"`
	ConfigSchema    []ConfigField        `json:"configSchema,omitempty"`
	Metadata        map[string]any       `json:"metadata,omitempty"`
}

type RuntimeManifest struct {
	Type       string `json:"type"`
	Entrypoint string `json:"entrypoint"`
}

type CapabilityManifest struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	Export            string   `json:"export"`
	RequiredFunctions []string `json:"requiredHostFunctions,omitempty"`
	Job               string   `json:"job,omitempty"`
}

type ConfigField struct {
	Key     string              `json:"key"`
	Type    string              `json:"type"`
	Options []ConfigFieldOption `json:"options,omitempty"`
	Default any                 `json:"default,omitempty"`
}

type ConfigFieldOption struct {
	Value string `json:"value"`
}

type AuthManifest struct {
	Contexts map[string]AuthContext `json:"contexts,omitempty"`
}

type AuthContext struct {
	Type                string            `json:"type"`
	Fields              []string          `json:"fields,omitempty"`
	AuthorizationURL    string            `json:"authorizationUrl,omitempty"`
	TokenURL            string            `json:"tokenUrl,omitempty"`
	Scopes              []string          `json:"scopes,omitempty"`
	ScopeSeparator      string            `json:"scopeSeparator,omitempty"`
	PKCE                bool              `json:"pkce,omitempty"`
	TokenRequestFormat  string            `json:"tokenRequestFormat,omitempty"`
	TokenAuth           string            `json:"tokenAuth,omitempty"`
	AuthorizationParams map[string]string `json:"authorizationParams,omitempty"`
	Refresh             *AuthRefresh      `json:"refresh,omitempty"`
	Placement           string            `json:"placement,omitempty"`
	Name                string            `json:"name,omitempty"`
	SecretField         string            `json:"secretField,omitempty"`
	SecretFields        []string          `json:"secretFields,omitempty"`
}

type AuthRefresh struct {
	Mode      string `json:"mode"`
	GrantType string `json:"grantType,omitempty"`
	Function  string `json:"function,omitempty"`
}

type PermissionManifest struct {
	Network   NetworkPermissions  `json:"network,omitempty"`
	Auth      []string            `json:"auth,omitempty"`
	Downloads DownloadPermissions `json:"downloads,omitempty"`
	Uploads   UploadPermissions   `json:"uploads,omitempty"`
}

type NetworkPermissions struct {
	StaticHosts           []string            `json:"staticHosts,omitempty"`
	UserConfiguredOrigins []string            `json:"userConfiguredOrigins,omitempty"`
	Redirects             RedirectPermissions `json:"redirects,omitempty"`
}

type RedirectPermissions struct {
	Mode  string   `json:"mode,omitempty"`
	Hosts []string `json:"hosts,omitempty"`
}

type DownloadPermissions struct {
	MaxBytes     int64    `json:"maxBytes,omitempty"`
	ContentTypes []string `json:"contentTypes,omitempty"`
}

type UploadPermissions struct {
	MaxBytes     int64    `json:"maxBytes,omitempty"`
	ContentTypes []string `json:"contentTypes,omitempty"`
}

type HostRequestSpec struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Auth    string            `json:"auth,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    *HostRequestBody  `json:"body,omitempty"`
	Expect  ResponseExpect    `json:"expect,omitempty"`
}

type HostRequestBody struct {
	Type  string          `json:"type"`
	JSON  any             `json:"json,omitempty"`
	Parts []MultipartPart `json:"parts,omitempty"`
}

type MultipartPart struct {
	Name        string `json:"name"`
	Source      string `json:"source,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	JSON        any    `json:"json,omitempty"`
}

type ResponseExpect struct {
	ContentTypes []string `json:"contentTypes,omitempty"`
	MaxBytes     int64    `json:"maxBytes,omitempty"`
}

type TrackTransferPlan struct {
	Format   string          `json:"format"`
	Transfer HostRequestSpec `json:"transfer"`
}

type UploadPlan struct {
	Request HostRequestSpec `json:"request"`
}

type PluginError struct {
	Code              string `json:"code"`
	Message           string `json:"message,omitempty"`
	RetryAfterSeconds *int   `json:"retryAfterSeconds,omitempty"`
}
