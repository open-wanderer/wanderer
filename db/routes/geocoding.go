package routes

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func GeocodingReverse(e *core.RequestEvent) error {
	query := e.Request.URL.Query()
	lat := query.Get("lat")
	lon := query.Get("lon")
	if lat == "" || lon == "" {
		return apis.NewBadRequestError("Missing query parameter: lat or lon", nil)
	}
	if _, err := strconv.ParseFloat(lat, 64); err != nil {
		return apis.NewBadRequestError("Invalid query parameter: lat or lon", err)
	}
	if _, err := strconv.ParseFloat(lon, 64); err != nil {
		return apis.NewBadRequestError("Invalid query parameter: lat or lon", err)
	}

	return geocodingNominatim(e, "/reverse", url.Values{
		"lat":            []string{lat},
		"lon":            []string{lon},
		"format":         []string{"geojson"},
		"addressdetails": []string{"1"},
	})
}

func GeocodingSearch(e *core.RequestEvent) error {
	query := e.Request.URL.Query()
	q := query.Get("q")
	if q == "" {
		return apis.NewBadRequestError("Missing query parameter: q", nil)
	}
	params := url.Values{
		"q":              []string{q},
		"format":         []string{"geojson"},
		"addressdetails": []string{"1"},
	}
	if limit := query.Get("limit"); limit != "" {
		if _, err := strconv.Atoi(limit); err != nil {
			return apis.NewBadRequestError("Invalid query parameter: limit", err)
		}
		params.Set("limit", limit)
	}

	return geocodingNominatim(e, "/search", params)
}

func geocodingNominatim(e *core.RequestEvent, path string, params url.Values) error {
	baseURL := geocodingExternalServiceBaseURL("NOMINATIM_URL", waypointNameNominatimDefaultURL)
	requestURL, err := geocodingExternalServiceURL(baseURL, path, params)
	if err != nil {
		return apis.NewBadRequestError("invalid Nominatim URL", err)
	}
	if err := nominatimRateLimit(e.Request.Context(), baseURL); err != nil {
		return err
	}

	var raw json.RawMessage
	if err := geocodingFetchJSON(e.Request.Context(), baseURL, requestURL, &raw); err != nil {
		return apis.NewBadRequestError("Nominatim request failed", err)
	}
	return e.Blob(http.StatusOK, "application/json", raw)
}
