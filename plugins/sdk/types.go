package sdk

const (
	HostRequestBodyTypeJSON      = "json"
	HostRequestBodyTypeMultipart = "multipart"
	MultipartSourceRoute         = "route"
	MultipartSourceRouteGPX      = "route.gpx"

	AuthHeaderAuthorization = "Authorization"
	AuthSchemeBearer        = "Bearer"
)

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

type HostResponse struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
	BodyBase64 string            `json:"bodyBase64,omitempty"`
	Error      *PluginError      `json:"error,omitempty"`
}

type PluginError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
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
	Instance          InstanceRef    `json:"instance"`
	Auth              map[string]any `json:"auth,omitempty"`
	State             map[string]any `json:"state,omitempty"`
	Options           map[string]any `json:"options,omitempty"`
	Limits            SyncLimits     `json:"limits,omitempty"`
	RecentExternalIDs []string       `json:"recentExternalIds,omitempty"`
}

type ListOutput struct {
	Items   []TrailImport  `json:"items"`
	State   map[string]any `json:"state,omitempty"`
	HasMore bool           `json:"hasMore"`
	Error   *PluginError   `json:"error,omitempty"`
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
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
}

type SendRouteInput struct {
	Instance InstanceRef    `json:"instance"`
	Auth     map[string]any `json:"auth,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Name     string         `json:"name,omitempty"`
	Route    Track          `json:"route"`
}

type UploadPlan struct {
	Request HostRequestSpec `json:"request"`
}
