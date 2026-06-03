package pluginsystem

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"pocketbase/util"

	extism "github.com/extism/go-sdk"
)

type hostHTTPResponse struct {
	Status     int               `json:"status"`
	Headers    map[string]string `json:"headers,omitempty"`
	BodyBase64 string            `json:"bodyBase64,omitempty"`
	Error      *PluginError      `json:"error,omitempty"`
}

type HostRequestOptions struct {
	Route []byte
}

type HostResponse struct {
	Status  int
	Headers map[string]string
	Body    []byte
}

var newConnectorHTTPClient = util.ConnectorHTTPClient

// extismHostFunctions exposes the host APIs that WASM plugins may call. Each
// function must delegate to the same policy-controlled host implementation that
// route handlers use.
func extismHostFunctions(manifest Manifest, policy RequestPolicyContext) []extism.HostFunction {
	fn := extism.NewHostFunctionWithStack(
		"http_request",
		func(ctx context.Context, plugin *extism.CurrentPlugin, stack []uint64) {
			requestBytes, err := plugin.ReadBytes(stack[0])
			if err != nil {
				writeHostHTTPResponse(ctx, plugin, stack, hostHTTPResponse{
					Error: &PluginError{Code: "invalid_request", Message: err.Error()},
				})
				return
			}

			var spec HostRequestSpec
			if err := json.Unmarshal(requestBytes, &spec); err != nil {
				writeHostHTTPResponse(ctx, plugin, stack, hostHTTPResponse{
					Error: &PluginError{Code: "invalid_request", Message: "invalid host request: " + err.Error()},
				})
				return
			}

			executed, err := ExecuteHostRequest(ctx, manifest, policy, spec, HostRequestOptions{})
			response := hostHTTPResponse{}
			if err != nil {
				response = hostHTTPResponse{
					Error: &PluginError{Code: "provider_unavailable", Message: err.Error()},
				}
			} else {
				response = hostHTTPResponse{
					Status:     executed.Status,
					Headers:    executed.Headers,
					BodyBase64: base64.StdEncoding.EncodeToString(executed.Body),
				}
			}
			writeHostHTTPResponse(ctx, plugin, stack, response)
		},
		[]extism.ValueType{extism.ValueTypePTR},
		[]extism.ValueType{extism.ValueTypePTR},
	)
	fn.SetNamespace("wanderer")
	return []extism.HostFunction{fn}
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
	if err := ValidateHostRequestSpec(manifest, spec, policy); err != nil {
		return HostResponse{}, err
	}
	if err := InjectHostRequestAuthFromPolicy(manifest, policy.HostAuth, &spec); err != nil {
		return HostResponse{}, err
	}
	resolved, err := ResolveRequestTarget(manifest, spec.Target, policy)
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

	headers := map[string]string{}
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	return HostResponse{
		Status:  resp.StatusCode,
		Headers: headers,
		Body:    bodyBytes,
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
	case HostRequestBodyTypeMultipart:
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for _, part := range spec.Body.Parts {
			if part.Source == MultipartSourceRoute || part.Source == MultipartSourceRouteGPX {
				if len(options.Route) == 0 {
					return nil, "", 0, fmt.Errorf("multipart part %q requires route content", part.Name)
				}
				partWriter, err := writer.CreateFormFile(part.Name, MultipartRouteFilename)
				if err != nil {
					return nil, "", 0, err
				}
				if _, err := partWriter.Write(options.Route); err != nil {
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

func validateHostRequestUpload(manifest Manifest, spec HostRequestSpec, contentType string, bodySize int64) error {
	if spec.Body == nil {
		return nil
	}
	if manifest.Permissions.Uploads.MaxBytes > 0 && bodySize > manifest.Permissions.Uploads.MaxBytes {
		return fmt.Errorf("plugin upload request exceeds manifest upload limit")
	}
	if contentType == "" || len(manifest.Permissions.Uploads.ContentTypes) == 0 {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("upload request has invalid content type")
	}
	for _, allowed := range manifest.Permissions.Uploads.ContentTypes {
		if strings.EqualFold(mediaType, allowed) {
			return nil
		}
	}
	return fmt.Errorf("upload request content type %q is not allowed", mediaType)
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
