package sdk

const (
	HostRequestBodyTypeJSON      = "json"
	HostRequestBodyTypeForm      = "form"
	HostRequestBodyTypeMultipart = "multipart"
	MultipartSourceTrail         = "trail"
	MultipartSourceTrailGPX      = "trail.gpx"

	AuthHeaderAuthorization = "Authorization"
	AuthSchemeBearer        = "Bearer"
)

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

type HostResponse struct {
	Status       int                 `json:"status"`
	HeaderValues map[string][]string `json:"headerValues,omitempty"`
	BodyBase64   string              `json:"bodyBase64,omitempty"`
	Error        *PluginError        `json:"error,omitempty"`
}

type PluginError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

type HostLogEntry struct {
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
}

type InstanceRef struct {
	ID       string `json:"id"`
	PluginID string `json:"pluginId"`
}

type RefreshSessionInput struct {
	Instance InstanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

type RefreshSessionOutput struct {
	Token     string `json:"token"`
	Scheme    string `json:"scheme,omitempty"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

type SyncLimits struct {
	MaxItems int `json:"maxItems,omitempty"`
}

type ListInput struct {
	Instance InstanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	State    map[string]any `json:"state,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
	Limits   SyncLimits     `json:"limits,omitempty"`
}

type ListOutput struct {
	Items   []TrailSummary `json:"items"`
	State   map[string]any `json:"state,omitempty"`
	HasMore bool           `json:"hasMore"`
	Error   *PluginError   `json:"error,omitempty"`
}

type DetailInput struct {
	Instance InstanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
	Summary  TrailSummary   `json:"summary"`
}

type DetailOutput struct {
	Item  TrailImport  `json:"item"`
	Error *PluginError `json:"error,omitempty"`
}

type TrailSummary struct {
	Source TrailImportSource `json:"source"`
	Kind   string            `json:"kind,omitempty"`
}

type TrailImport struct {
	Source       TrailImportSource `json:"source"`
	Kind         string            `json:"kind,omitempty"`
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	StartedAt    string            `json:"startedAt,omitempty"`
	ActivityType string            `json:"activityType,omitempty"`
	Privacy      *string           `json:"privacy,omitempty"`
	Track        Track             `json:"track"`
	Waypoints    []Waypoint        `json:"waypoints,omitempty"`
	Photos       []Photo           `json:"photos,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

type TrailImportSource struct {
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	URL        string `json:"url,omitempty"`
}

type Track struct {
	Format        string `json:"format"`
	ContentBase64 string `json:"contentBase64"`
}

type Waypoint struct {
	ExternalID  string   `json:"externalId,omitempty"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Lat         float64  `json:"lat"`
	Lon         float64  `json:"lon"`
	Ele         *float64 `json:"ele,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	Photos      []Photo  `json:"photos,omitempty"`
}

type Photo struct {
	ExternalID  string      `json:"externalId,omitempty"`
	Filename    string      `json:"filename,omitempty"`
	ContentType string      `json:"contentType,omitempty"`
	Lat         *float64    `json:"lat,omitempty"`
	Lon         *float64    `json:"lon,omitempty"`
	Source      MediaSource `json:"source"`
}

type MediaSource struct {
	Type     string    `json:"type"`
	URL      string    `json:"url,omitempty"`
	MediaRef *MediaRef `json:"mediaRef,omitempty"`
}

type MediaRef struct {
	Connector string       `json:"connector"`
	Auth      string       `json:"auth,omitempty"`
	Path      string       `json:"path,omitempty"`
	Query     []QueryParam `json:"query,omitempty"`
	AssetID   string       `json:"assetId,omitempty"`
}

type TrailSendInput struct {
	Instance InstanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Name     string         `json:"name,omitempty"`
	Trail    Track          `json:"trail"`
}

type TrailSendPlan struct {
	Request HostRequestSpec `json:"request"`
}
