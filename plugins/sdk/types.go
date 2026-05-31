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
