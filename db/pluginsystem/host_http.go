package pluginsystem

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"pocketbase/util"

	extism "github.com/extism/go-sdk"
)

type hostHTTPResponse struct {
	Status       int                 `json:"status"`
	HeaderValues map[string][]string `json:"headerValues,omitempty"`
	BodyBase64   string              `json:"bodyBase64,omitempty"`
	Error        *PluginError        `json:"error,omitempty"`
}

type HostRequestOptions struct {
	Trail []byte
}

type HostResponse struct {
	Status       int
	HeaderValues map[string][]string
	Body         []byte
}

var newConnectorHTTPClient = util.ConnectorHTTPClient

const maxHostLogPayloadBytes = 8 * 1024

// extismHostFunctions exposes the host APIs that WASM plugins may call. Each
// function must delegate to the same policy-controlled host implementation that
// backend handlers use.
func extismHostFunctions(manifest Manifest, policy RequestPolicyContext) []extism.HostFunction {
	httpFn := extism.NewHostFunctionWithStack(
		"http_request",
		func(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64) {
			requestBytes, err := plugin.ReadBytes(stack[0])
			if err != nil {
				writeHostHTTPResponse(ctx, plugin, stack, hostHTTPResponse{
					Error: &PluginError{Code: "invalid_request", Message: err.Error()},
				})
				return
			}
			response := executeHostHTTPRequest(ctx, manifest, policy, requestBytes)
			writeHostHTTPResponse(ctx, plugin, stack, response)
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
	httpFn.SetNamespace("wanderer")

	logFn := extism.NewHostFunctionWithStack(
		"log",
		func(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64) {
			message, err := readBoundedHostLogPayload(plugin, stack[0])
			if err != nil {
				plugin.Log(extism.LogLevelError, "read host log message: "+err.Error())
				return
			}
			entry, err := parseHostLogEntry(message)
			if err != nil {
				plugin.Log(extism.LogLevelError, "invalid host log message: "+err.Error())
				return
			}
			log.Printf("plugin log [%s]: %s", entry.Level, entry.Message)
			_ = ctx
		},
		[]extism.ValueType{extism.ValueTypePTR},
		nil,
	)
	logFn.SetNamespace("wanderer")

	return []extism.HostFunction{httpFn, logFn}
}

func readBoundedHostLogPayload(plugin *extism.CurrentPlugin, offset uint64) ([]byte, error) {
	length, err := plugin.Length(offset)
	if err != nil {
		return nil, err
	}
	if length > maxHostLogPayloadBytes {
		return nil, fmt.Errorf("log message exceeds maximum size")
	}
	return plugin.ReadBytes(offset)
}

func parseHostLogEntry(message []byte) (HostLogEntry, error) {
	if len(message) > maxHostLogPayloadBytes {
		return HostLogEntry{}, fmt.Errorf("log message exceeds maximum size")
	}
	var entry HostLogEntry
	if err := json.Unmarshal(message, &entry); err != nil {
		return HostLogEntry{}, fmt.Errorf("decode log entry: %w", err)
	}
	level, err := normalizeHostLogLevel(entry.Level)
	if err != nil {
		return HostLogEntry{}, err
	}
	entry.Level = level
	entry.Message = sanitizeHostLogMessage(entry.Message)
	if entry.Message == "" {
		return HostLogEntry{}, fmt.Errorf("log message is required")
	}
	return entry, nil
}

func sanitizeHostLogMessage(message string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, message))
}

func normalizeHostLogLevel(level string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return "debug", nil
	case "info":
		return "info", nil
	case "warn":
		return "warn", nil
	case "error":
		return "error", nil
	default:
		return "", fmt.Errorf("unsupported log level %q", level)
	}
}

// executeHostHTTPRequest turns a raw plugin http_request payload into the
// hostHTTPResponse that the plugin reads back. It is the single source of truth
// for the request/response contract shared by the in-process runtime
// (extismHostFunctions) and the worker process (handleHostHTTPRequest), so the
// two paths cannot drift on error codes or response shape.
func executeHostHTTPRequest(ctx context.Context, manifest Manifest, policy RequestPolicyContext, requestBytes []byte) hostHTTPResponse {
	var spec HostRequestSpec
	if err := json.Unmarshal(requestBytes, &spec); err != nil {
		return hostHTTPResponse{
			Error: &PluginError{Code: "invalid_request", Message: "invalid host request: " + err.Error()},
		}
	}
	executed, err := ExecuteHostRequest(ctx, manifest, policy, spec, HostRequestOptions{})
	if err != nil {
		return hostHTTPResponse{
			Error: &PluginError{Code: "provider_unavailable", Message: err.Error()},
		}
	}
	return hostHTTPResponse{
		Status:       executed.Status,
		HeaderValues: executed.HeaderValues,
		BodyBase64:   base64.StdEncoding.EncodeToString(executed.Body),
	}
}

func writeHostHTTPResponse(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64, response hostHTTPResponse) {
	responseBytes, err := json.Marshal(response)
	if err != nil {
		responseBytes, _ = json.Marshal(hostHTTPResponse{
			Error: &PluginError{Code: "internal_error", Message: err.Error()},
		})
	}
	offset, err := plugin.WriteBytes(responseBytes)
	if err != nil {
		plugin.Log(extism.LogLevelError, "write host http response: "+err.Error())
		stack[0] = 0
		return
	}
	stack[0] = offset
	_ = ctx
}

// ExecuteHostRequest is the single network chokepoint for plugin-controlled
// HTTP. It validates manifest policy, builds optional request bodies, enforces
// upload/response limits, follows only permitted redirects, and returns the
// bounded provider response.
func ExecuteHostRequest(ctx context.Context, manifest Manifest, policy RequestPolicyContext, spec HostRequestSpec, options HostRequestOptions) (HostResponse, error) {
	if err := InjectHostRequestAuthFromPolicy(manifest, policy.HostAuth, &spec); err != nil {
		return HostResponse{}, err
	}
	resolved, err := ValidateAndResolveHostRequestSpec(manifest, spec, policy)
	if err != nil {
		return HostResponse{}, err
	}

	body, contentType, bodySize, err := hostRequestBody(spec, options)
	if err != nil {
		return HostResponse{}, err
	}
	if err := validateHostRequestUpload(manifest, spec, contentType, bodySize); err != nil {
		return HostResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, spec.Method, resolved.URL.String(), body)
	if err != nil {
		return HostResponse{}, err
	}
	for key, value := range spec.Headers {
		req.Header.Set(key, value)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	client, err := newConnectorHTTPClient(util.ConnectorHTTPPolicy{
		BaseURL:      resolved.Connector.BaseURL,
		AllowPrivate: resolved.Connector.AllowPrivate,
		TLSMode:      resolved.Connector.TLS.Mode,
		TLSCABundle:  resolved.Connector.TLS.CABundle,
	}, func(req *http.Request, via []*http.Request) error {
		if spec.FollowRedirects != nil && !*spec.FollowRedirects {
			return http.ErrUseLastResponse
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		previous := resolved.URL
		if len(via) > 0 {
			previous = via[len(via)-1].URL
		}
		return ValidateConnectorRedirect(resolved.Connector, previous, req.URL)
	})
	if err != nil {
		return HostResponse{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return HostResponse{}, err
	}
	defer resp.Body.Close()

	if err := validateHostHTTPResponse(manifest, spec, resp); err != nil {
		return HostResponse{}, err
	}
	maxBytes := effectiveResponseMaxBytes(manifest, spec)
	limit := maxBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return HostResponse{}, err
	}
	if maxBytes > 0 && int64(len(bodyBytes)) > maxBytes {
		return HostResponse{}, fmt.Errorf("provider response exceeds maximum size")
	}
	if maxBytes <= 0 && int64(len(bodyBytes)) > limit {
		return HostResponse{}, fmt.Errorf("provider response exceeds default maximum size")
	}

	headerValues := map[string][]string{}
	for key, values := range resp.Header {
		if len(values) > 0 {
			headerValues[key] = append([]string{}, values...)
		}
	}
	return HostResponse{
		Status:       resp.StatusCode,
		HeaderValues: headerValues,
		Body:         bodyBytes,
	}, nil
}

func hostRequestBody(spec HostRequestSpec, options HostRequestOptions) (io.Reader, string, int64, error) {
	if spec.Body == nil {
		return nil, "", 0, nil
	}
	switch spec.Body.Type {
	case HostRequestBodyTypeJSON:
		body, err := json.Marshal(spec.Body.JSON)
		if err != nil {
			return nil, "", 0, err
		}
		return bytes.NewReader(body), "application/json", int64(len(body)), nil
	case HostRequestBodyTypeForm:
		body, err := formURLEncodedBody(spec.Body.Form)
		if err != nil {
			return nil, "", 0, err
		}
		return strings.NewReader(body), "application/x-www-form-urlencoded", int64(len(body)), nil
	case HostRequestBodyTypeMultipart:
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for _, part := range spec.Body.Parts {
			if part.Source == MultipartSourceTrail || part.Source == MultipartSourceTrailGPX {
				if len(options.Trail) == 0 {
					return nil, "", 0, fmt.Errorf("multipart part %q requires trail content", part.Name)
				}
				filename := part.Filename
				if filename == "" {
					filename = MultipartTrailFilename
				}
				partWriter, err := writer.CreateFormFile(part.Name, filename)
				if err != nil {
					return nil, "", 0, err
				}
				if _, err := partWriter.Write(options.Trail); err != nil {
					return nil, "", 0, err
				}
				continue
			}
			if part.JSON != nil {
				data, err := json.Marshal(part.JSON)
				if err != nil {
					return nil, "", 0, err
				}
				if err := writer.WriteField(part.Name, string(data)); err != nil {
					return nil, "", 0, err
				}
			}
		}
		if err := writer.Close(); err != nil {
			return nil, "", 0, err
		}
		return &body, writer.FormDataContentType(), int64(body.Len()), nil
	default:
		return nil, "", 0, fmt.Errorf("unsupported host request body type %q", spec.Body.Type)
	}
}

func formURLEncodedBody(fields []FormField) (string, error) {
	encoded := make([]string, 0, len(fields))
	for _, field := range fields {
		if field.Name == "" {
			return "", fmt.Errorf("form field name must not be empty")
		}
		if hasControl(field.Name) || hasControl(field.Value) {
			return "", fmt.Errorf("form fields must not contain control characters")
		}
		encoded = append(encoded, url.QueryEscape(field.Name)+"="+url.QueryEscape(field.Value))
	}
	return strings.Join(encoded, "&"), nil
}

func validateHostRequestUpload(manifest Manifest, spec HostRequestSpec, contentType string, bodySize int64) error {
	if spec.Body == nil {
		return nil
	}
	if manifest.Permissions.Uploads.MaxBytes > 0 && bodySize > manifest.Permissions.Uploads.MaxBytes {
		return fmt.Errorf("host request upload exceeds manifest upload limit")
	}
	if contentType == "" || len(manifest.Permissions.Uploads.ContentTypes) == 0 {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("host request upload has invalid content type")
	}
	for _, allowed := range manifest.Permissions.Uploads.ContentTypes {
		if strings.EqualFold(mediaType, allowed) {
			return nil
		}
	}
	return fmt.Errorf("host request upload content type %q is not allowed", mediaType)
}

func validateHostHTTPResponse(manifest Manifest, spec HostRequestSpec, resp *http.Response) error {
	allowedContentTypes := effectiveResponseContentTypes(manifest, spec)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 && len(allowedContentTypes) > 0 {
		contentType := resp.Header.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || mediaType == "" {
			return fmt.Errorf("provider response has invalid content type")
		}
		allowed := false
		for _, expected := range allowedContentTypes {
			if strings.EqualFold(mediaType, expected) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("provider response content type %q is not allowed", mediaType)
		}
	}
	maxBytes := effectiveResponseMaxBytes(manifest, spec)
	if maxBytes > 0 && resp.ContentLength > maxBytes {
		return fmt.Errorf("provider response exceeds maximum size")
	}
	return nil
}

func effectiveResponseContentTypes(manifest Manifest, spec HostRequestSpec) []string {
	if len(spec.Expect.ContentTypes) > 0 {
		return spec.Expect.ContentTypes
	}
	return manifest.Permissions.Downloads.ContentTypes
}

func effectiveResponseMaxBytes(manifest Manifest, spec HostRequestSpec) int64 {
	if spec.Expect.MaxBytes > 0 {
		return spec.Expect.MaxBytes
	}
	return manifest.Permissions.Downloads.MaxBytes
}
