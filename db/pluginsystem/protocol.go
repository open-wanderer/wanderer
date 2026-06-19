package pluginsystem

const (
	ManifestVersion = "1.0"
	RuntimeWASM     = "wasm"

	PluginTypeTrails = "trails"

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
	HostRequestBodyTypeForm      = "form"
	HostRequestBodyTypeMultipart = "multipart"
	MultipartSourceTrail         = "trail"
	MultipartSourceTrailGPX      = "trail.gpx"
	MultipartTrailFilename       = "trail.gpx"
)

type Manifest struct {
	ManifestVersion string               `json:"manifestVersion"`
	ID              string               `json:"id"`
	Type            string               `json:"type"`
	Name            string               `json:"name"`
	Description     string               `json:"description,omitempty"`
	Version         string               `json:"version"`
	Runtime         RuntimeManifest      `json:"runtime"`
	Capabilities    []CapabilityManifest `json:"capabilities"`
	Auth            AuthManifest         `json:"auth,omitempty"`
	Permissions     PermissionManifest   `json:"permissions,omitempty"`
	ConfigSchema    []ConfigField        `json:"configSchema,omitempty"`
	HostConfig      map[string]any       `json:"hostConfig,omitempty"`
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
	Key          string              `json:"key"`
	Type         string              `json:"type"`
	Label        string              `json:"label,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	Description  string              `json:"description,omitempty"`
	Descriptions map[string]string   `json:"descriptions,omitempty"`
	Options      []ConfigFieldOption `json:"options,omitempty"`
	Default      any                 `json:"default,omitempty"`
	Required     bool                `json:"required,omitempty"`
	Hidden       bool                `json:"hidden,omitempty"`
}

type ConfigFieldOption struct {
	Value  string            `json:"value"`
	Label  string            `json:"label,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
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
	Connectors []ConnectorTargetPermission `json:"connectors,omitempty"`
	Redirects  RedirectPermissions         `json:"redirects,omitempty"`
}

type ConnectorTargetPermission struct {
	Name                     string   `json:"name"`
	Type                     string   `json:"type"`
	FixedBaseURL             string   `json:"fixedBaseURL,omitempty"`
	ConfigKey                string   `json:"configKey,omitempty"`
	AllowedPathPrefixes      []string `json:"allowedPathPrefixes,omitempty"`
	Auth                     []string `json:"auth,omitempty"`
	SupportsMediaAuth        bool     `json:"supportsMediaAuth,omitempty"`
	SupportsStorageRedirects bool     `json:"supportsStorageRedirects,omitempty"`
	SupportsCustomTLS        bool     `json:"supportsCustomTLS,omitempty"`
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
	Method          string            `json:"method"`
	Target          RequestTarget     `json:"target"`
	Auth            string            `json:"auth,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            *HostRequestBody  `json:"body,omitempty"`
	Expect          ResponseExpect    `json:"expect,omitempty"`
	FollowRedirects *bool             `json:"followRedirects,omitempty"`
}

type RequestTarget struct {
	Type      string       `json:"type"`
	Connector string       `json:"connector,omitempty"`
	Path      string       `json:"path,omitempty"`
	Query     []QueryParam `json:"query,omitempty"`
}

type QueryParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type HostRequestBody struct {
	Type  string          `json:"type"`
	JSON  any             `json:"json,omitempty"`
	Form  []FormField     `json:"form,omitempty"`
	Parts []MultipartPart `json:"parts,omitempty"`
}

type FormField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type MultipartPart struct {
	Name        string `json:"name"`
	Source      string `json:"source,omitempty"`
	Filename    string `json:"filename,omitempty"`
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

type TrailSendPlan struct {
	Request HostRequestSpec `json:"request"`
}

type PluginError struct {
	Code              string `json:"code"`
	Message           string `json:"message,omitempty"`
	RetryAfterSeconds *int   `json:"retryAfterSeconds,omitempty"`
}

type HostLogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}
