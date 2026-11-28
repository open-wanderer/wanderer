package routes

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/geocoding"
	"pocketbase/util"
)

// RegisterGeocoding installs the internal-only web proxy routes. Use one client
// for both paths so requests to public Nominatim share its rate limit.
func RegisterGeocoding(se *core.ServeEvent) {
	client, usesPublicDefault := configuredGeocodingClient()
	if client == nil {
		se.App.Logger().Warn("Geocoding disabled: invalid Nominatim configuration")
	} else if usesPublicDefault {
		se.App.Logger().Info("Geocoding is using public Nominatim by default because no Nominatim URL is configured on db")
	}

	group := se.Router.Group("/geocoding")
	// The default activity log records request URLs, including search terms and coordinates.
	group.Unbind(apis.DefaultActivityLoggerMiddlewareId)
	group.BindFunc(func(e *core.RequestEvent) error {
		if err := util.RequireInternalProxy(e.Request); err != nil {
			return e.UnauthorizedError("Unauthorized", nil)
		}
		if client == nil {
			return geocodingFailure(e, http.StatusBadGateway)
		}
		return e.Next()
	})

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
	default:
		// Upstream redirects are never followed or exposed as redirects to web.
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
