package sdk

const (
	HostRequestBodyTypeJSON      = "json"
	HostRequestBodyTypeForm      = "form"
	HostRequestBodyTypeText      = "text"
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
	Text  string          `json:"text,omitempty"`
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
	Code              string `json:"code"`
	Message           string `json:"message,omitempty"`
	RetryAfterSeconds *int   `json:"retryAfterSeconds,omitempty"`
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
	MaxItems              int `json:"maxItems,omitempty"`
	MaxPhotosPerTrail     int `json:"maxPhotosPerTrail,omitempty"`
	MaxPhotosPerWaypoint  int `json:"maxPhotosPerWaypoint,omitempty"`
	MaxPhotosPerSummitLog int `json:"maxPhotosPerSummitLog,omitempty"`
}

// PhotoImportLimits are host-enforced write limits advertised to an asset
// capability. Candidate search budgets belong in AssetSearchLimits instead.
type PhotoImportLimits struct {
	MaxPhotosPerTrail     int `json:"maxPhotosPerTrail,omitempty"`
	MaxPhotosPerWaypoint  int `json:"maxPhotosPerWaypoint,omitempty"`
	MaxPhotosPerSummitLog int `json:"maxPhotosPerSummitLog,omitempty"`
}

// AssetSearchLimits bounds a single asset-candidate search call. MaxItems
// limits returned candidates, MaxScannedItems limits inspected provider items,
// and MaxProviderRequests is the plugin's cooperative outbound-request budget.
type AssetSearchLimits struct {
	MaxItems            int `json:"maxItems,omitempty"`
	MaxScannedItems     int `json:"maxScannedItems,omitempty"`
	MaxProviderRequests int `json:"maxProviderRequests,omitempty"`
}

// ManeuverLimits are provider-independent output limits supplied by the host
// to every maneuvers.v1 invocation. Adapters should use them to avoid doing
// work whose result the host would reject, but the host still validates every
// returned result independently.
type ManeuverLimits struct {
	MaxGeometryPoints                int   `json:"maxGeometryPoints"`
	MaxManeuvers                     int   `json:"maxManeuvers"`
	MaxProviderInstructionCharacters int   `json:"maxProviderInstructionCharacters"`
	MaxStreetNames                   int   `json:"maxStreetNames"`
	MaxStreetNameCharacters          int   `json:"maxStreetNameCharacters"`
	MaxResponseBytes                 int64 `json:"maxResponseBytes"`
}

type ManeuverPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type ManeuverTrackPart struct {
	Points []ManeuverPoint `json:"points"`
}

type ManeuverProfile struct {
	ID            string         `json:"id,omitempty"`
	PluginID      string         `json:"pluginId,omitempty"`
	Key           string         `json:"key"`
	Kind          string         `json:"kind,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	ContentBase64 string         `json:"contentBase64,omitempty"`
	ContentType   string         `json:"contentType,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	NativeConfig  map[string]any `json:"nativeConfig,omitempty"`
	PreparedKey   string         `json:"preparedKey,omitempty"`
}

// ManeuverRequest is the provider-neutral maneuvers.v1 payload. Persisted
// trail identifiers, sharing tokens, raw GPX and provider-specific limits are
// intentionally absent from this plugin boundary.
type ManeuverRequest struct {
	TrackParts          []ManeuverTrackPart `json:"trackParts"`
	Mode                string              `json:"mode,omitempty"`
	Category            string              `json:"category,omitempty"`
	Subcategory         string              `json:"subcategory,omitempty"`
	Profile             ManeuverProfile     `json:"profile"`
	Preferences         map[string]any      `json:"preferences,omitempty"`
	RequiredPreferences []string            `json:"requiredPreferences,omitempty"`
	Language            string              `json:"language,omitempty"`
	Limits              ManeuverLimits      `json:"limits"`
}

type ManeuverGeometry struct {
	Format      string `json:"format"`
	Precision   int    `json:"precision"`
	Coordinates string `json:"coordinates"`
}

type Maneuver struct {
	Type                string   `json:"type"`
	ProviderInstruction string   `json:"providerInstruction,omitempty"`
	DistanceMeters      float64  `json:"distanceMeters"`
	DurationSeconds     *float64 `json:"durationSeconds,omitempty"`
	BeginShapeIndex     int      `json:"beginShapeIndex"`
	EndShapeIndex       int      `json:"endShapeIndex"`
	BearingBefore       *float64 `json:"bearingBefore,omitempty"`
	BearingAfter        *float64 `json:"bearingAfter,omitempty"`
	RoundaboutExit      *int     `json:"roundaboutExit,omitempty"`
	StreetNames         []string `json:"streetNames,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

type ManeuverResult struct {
	Geometry  ManeuverGeometry `json:"geometry"`
	Maneuvers []Maneuver       `json:"maneuvers,omitempty"`
	Warnings  []string         `json:"warnings,omitempty"`
	Error     *PluginError     `json:"error,omitempty"`
}

// OmittedAsset describes an explicitly requested asset that a plugin could not
// convert into an importable photo.
type OmittedAsset struct {
	AssetID string `json:"assetId"`
	Reason  string `json:"reason"`
}

// AssetSearchStats contains plugin-reported observability data. Hosts must not
// use it to enforce limits because it is not independently trustworthy.
type AssetSearchStats struct {
	ScannedItems int `json:"scannedItems,omitempty"`
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
	Limits   SyncLimits     `json:"limits,omitempty"`
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
	TakenAt     string      `json:"takenAt,omitempty"`
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
