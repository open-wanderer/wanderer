package routes

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/geocoding"
	"pocketbase/util"
)

const (
	// An upstream 401 is sanitized without any upstream headers and never
	// carries this internal db-to-web marker.
	geocodingFailureHeader    = "X-Wanderer-Geocoding-Failure"
	geocodingProxyAuthFailure = "internal_proxy_auth"

	geocodingPrivacyHeader   = "X-Wanderer-Geocoding-Privacy"
	geocodingPrivateRequired = "private-required"
)

// RegisterGeocoding installs the internal-only web proxy routes. Use one client
// for both paths so requests to public Nominatim share its rate limit.
func RegisterGeocoding(se *core.ServeEvent) {
	client, usesPublicDefault := configuredGeocodingClient()
	if client == nil {
		se.App.Logger().Warn("Geocoding disabled: invalid Nominatim configuration on db",
			"category", "invalid_nominatim_configuration")
	} else if usesPublicDefault {
		se.App.Logger().Info("Geocoding default is public Nominatim because no Nominatim URL is configured on db")
	}

	group := se.Router.Group("/geocoding")
	// The default activity log records request URLs, including search terms and coordinates.
	group.Unbind(apis.DefaultActivityLoggerMiddlewareId)
	group.BindFunc(geocodingProxyAccess(client))

	group.GET("/search", func(e *core.RequestEvent) error {
		params := e.Request.URL.Query()
		q := params.Get("q")
		if q == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Missing query parameter: q"})
		}
		// The public web route validates numbers using JS Number semantics.
		// Pass the accepted spelling through unchanged.
		response, err := client.Search(e.Request.Context(), q, params.Get("limit"))
		return writeGeocodingResponse(e, response, err)
	})
	group.GET("/reverse", func(e *core.RequestEvent) error {
		params := e.Request.URL.Query()
		lat, lon := params.Get("lat"), params.Get("lon")
		if lat == "" || lon == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"message": "Missing query parameter: lat or lon"})
		}
		response, err := client.Reverse(e.Request.Context(), lat, lon)
		return writeGeocodingResponse(e, response, err)
	})
}

func geocodingProxyAccess(client *geocoding.Client) func(*core.RequestEvent) error {
	var warnPrivateRequired sync.Once
	return func(e *core.RequestEvent) error {
		if err := util.RequireInternalProxy(e.Request); err != nil {
			e.Response.Header().Set(geocodingFailureHeader, geocodingProxyAuthFailure)
			return e.UnauthorizedError("Unauthorized", nil)
		}
		if client == nil {
			return geocodingFailure(e, http.StatusBadGateway)
		}
		// Only check the signal from the authenticated web proxy. Unknown or
		// ambiguous values must not permit a request to reach any upstream.
		if values := e.Request.Header.Values(geocodingPrivacyHeader); len(values) != 0 {
			if len(values) != 1 || (values[0] != "" && values[0] != geocodingPrivateRequired) {
				e.App.Logger().Warn("Geocoding blocked: invalid privacy signal from web",
					"category", "invalid_privacy_signal")
				return geocodingFailure(e, http.StatusBadGateway)
			}
			if values[0] == geocodingPrivateRequired && client.UsesPublicService() {
				warnPrivateRequired.Do(func() {
					e.App.Logger().Warn("Geocoding blocked: configure a private NOMINATIM_URL on db or remove the obsolete web-only setting to use public Nominatim",
						"category", "private_nominatim_required")
				})
				return geocodingFailure(e, http.StatusBadGateway)
			}
		}
		return e.Next()
	}
}

func configuredGeocodingClient() (client *geocoding.Client, usesPublicDefault bool) {
	baseURL, configured := os.LookupEnv("NOMINATIM_URL")
	if !configured {
		baseURL, configured = os.LookupEnv("PUBLIC_NOMINATIM_URL")
	}
	// A broken private endpoint setting must not silently fall back to public Nominatim.
	if configured && strings.TrimSpace(baseURL) == "" {
		return nil, false
	}

	// The Go binary has no Wanderer release version; use the client's dev identifier.
	client, err := geocoding.NewClient(baseURL, "")
	if err != nil {
		return nil, false
	}
	return client, !configured
}

func writeGeocodingResponse(e *core.RequestEvent, response geocoding.Response, err error) error {
	status := response.StatusCode
	switch {
	case err != nil:
		logGeocodingFailure(e, err)
		return geocodingFailure(e, http.StatusBadGateway)
	case status >= 400 && status <= 599:
		return geocodingFailure(e, status)
	case status == http.StatusNoContent || status == http.StatusResetContent:
		return e.NoContent(status)
	case status >= 200 && status < 300:
		// Keep the client's JSON intact and avoid PocketBase's fields projection.
		return e.Blob(status, "application/json", response.Body)
	case status >= 300 && status < 400:
		// Upstream redirects are never followed or exposed as redirects to web.
		e.App.Logger().Warn("Nominatim upstream redirect blocked", "category", "upstream_redirect")
		return geocodingFailure(e, http.StatusBadGateway)
	default:
		return geocodingFailure(e, http.StatusBadGateway)
	}
}

func logGeocodingFailure(e *core.RequestEvent, err error) {
	// A caller cancellation or deadline is not an upstream failure.
	if e.Request.Context().Err() != nil {
		return
	}
	var category string
	switch {
	case errors.Is(err, geocoding.ErrRequestTimeout):
		category = "timeout"
	case errors.Is(err, geocoding.ErrResponseTooLarge):
		category = "response_too_large"
	case errors.Is(err, geocoding.ErrRequestFailed):
		category = "transport_read_failure"
	default:
		return
	}
	e.App.Logger().Debug("Nominatim upstream request failed", "category", category)
}

func geocodingFailure(e *core.RequestEvent, status int) error {
	return e.JSON(status, map[string]string{"message": "Nominatim request failed"})
}
