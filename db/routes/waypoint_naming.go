package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"pocketbase/util"
)

const (
	waypointNameOverpassDefaultURL  = "https://overpass-api.de"
	waypointNameNominatimDefaultURL = "https://nominatim.openstreetmap.org"
	waypointNameNominatimMinDelay   = time.Second
	waypointNameHTTPResponseLimit   = 1 << 20
)

var (
	waypointNameDefaultHTTPClient = &http.Client{
		Timeout:       8 * time.Second,
		CheckRedirect: geocodingTrustedServiceRedirectPolicy,
	}
	waypointNameHTTPClient          = waypointNameDefaultHTTPClient
	waypointNameNominatimMu         sync.Mutex
	waypointNameLastNominatimCall   time.Time
	waypointNameNominatimZoomLevels = []int{18, 16, 14, 12, 10}
)

type waypointNameOverpassResponse struct {
	Elements []waypointNameOverpassElement `json:"elements"`
}

type waypointNameOverpassElement struct {
	Type   string             `json:"type"`
	ID     int64              `json:"id"`
	Lat    float64            `json:"lat"`
	Lon    float64            `json:"lon"`
	Center *waypointNameCoord `json:"center"`
	Tags   map[string]string  `json:"tags"`
}

type waypointNameCoord struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type waypointNameNominatimReverseResponse struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Address     map[string]string `json:"address"`
}

func resolveWaypointName(ctx context.Context, logger *slog.Logger, lat float64, lon float64, radius float64) (string, error) {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return "", nil
	}

	poiName, poiErr := waypointNameFromOverpassPOI(ctx, lat, lon, radius)
	if poiName != "" {
		logResolvedWaypointName(logger, "overpass", poiName, lat, lon, radius, nil)
		return poiName, nil
	}

	reverseName, reverseErr := waypointNameFromNominatim(ctx, lat, lon)
	if reverseName != "" {
		logResolvedWaypointName(logger, "nominatim", reverseName, lat, lon, radius, nil)
		return reverseName, nil
	}

	fallbackName := coordinateWaypointName(lat, lon)
	if reverseErr != nil {
		logResolvedWaypointName(logger, "coordinate", fallbackName, lat, lon, radius, reverseErr)
		return fallbackName, nil
	}
	if poiErr != nil {
		logResolvedWaypointName(logger, "coordinate", fallbackName, lat, lon, radius, poiErr)
		return fallbackName, nil
	}
	logResolvedWaypointName(logger, "coordinate", fallbackName, lat, lon, radius, nil)
	return fallbackName, nil
}

func waypointNameFromOverpassPOI(ctx context.Context, lat float64, lon float64, radius float64) (string, error) {
	radiusMeters := int(math.Ceil(radius))
	if radiusMeters <= 0 {
		return "", nil
	}

	baseURL := geocodingExternalServiceBaseURL("OVERPASS_API_URL", waypointNameOverpassDefaultURL)
	requestURL, err := geocodingExternalServiceURL(
		baseURL,
		"/api/interpreter",
		url.Values{"data": []string{waypointNameOverpassPOIQuery(lat, lon, radiusMeters)}},
	)
	if err != nil {
		return "", err
	}

	var result waypointNameOverpassResponse
	if err := geocodingFetchJSON(ctx, baseURL, requestURL, &result); err != nil {
		return "", err
	}
	return bestWaypointNameFromOverpass(result.Elements, lat, lon, radius), nil
}

func waypointNameOverpassPOIQuery(lat float64, lon float64, radiusMeters int) string {
	return fmt.Sprintf(`[out:json][timeout:5];
(
  nwr(around:%d,%.7f,%.7f)["name"]["tourism"~"^(attraction|viewpoint|museum|gallery|artwork|information|picnic_site|alpine_hut|wilderness_hut)$"];
  nwr(around:%d,%.7f,%.7f)["name"]["historic"];
  nwr(around:%d,%.7f,%.7f)["name"]["natural"~"^(peak|saddle|waterfall|spring|cave_entrance|rock|stone|tree)$"];
  nwr(around:%d,%.7f,%.7f)["name"]["mountain_pass"="yes"];
  nwr(around:%d,%.7f,%.7f)["name"]["man_made"~"^(tower|lighthouse|obelisk|cross|watermill|windmill)$"];
  nwr(around:%d,%.7f,%.7f)["name"]["amenity"~"^(place_of_worship|fountain)$"];
  nwr(around:%d,%.7f,%.7f)["name"]["leisure"~"^(park|garden)$"];
);
out center 20;`,
		radiusMeters, lat, lon,
		radiusMeters, lat, lon,
		radiusMeters, lat, lon,
		radiusMeters, lat, lon,
		radiusMeters, lat, lon,
		radiusMeters, lat, lon,
		radiusMeters, lat, lon,
	)
}

func bestWaypointNameFromOverpass(elements []waypointNameOverpassElement, lat float64, lon float64, radius float64) string {
	bestName := ""
	bestScore := 0
	bestDistance := math.MaxFloat64
	for _, element := range elements {
		name := cleanWaypointName(element.Tags["name"])
		if name == "" {
			continue
		}
		poiLat, poiLon, ok := waypointNameOverpassElementCoord(element)
		if !ok {
			continue
		}
		distance := util.HaversineDistanceMeters(lat, lon, poiLat, poiLon)
		if radius > 0 && distance > radius {
			continue
		}
		score := waypointNamePOIScore(element.Tags)
		if score == 0 {
			continue
		}
		if score > bestScore || (score == bestScore && distance < bestDistance) {
			bestName = name
			bestScore = score
			bestDistance = distance
		}
	}
	return bestName
}

func waypointNameOverpassElementCoord(element waypointNameOverpassElement) (float64, float64, bool) {
	if element.Type == "node" {
		return element.Lat, element.Lon, element.Lat >= -90 && element.Lat <= 90 && element.Lon >= -180 && element.Lon <= 180
	}
	if element.Center != nil {
		return element.Center.Lat, element.Center.Lon, element.Center.Lat >= -90 && element.Center.Lat <= 90 && element.Center.Lon >= -180 && element.Center.Lon <= 180
	}
	return 0, 0, false
}

func waypointNamePOIScore(tags map[string]string) int {
	switch tags["tourism"] {
	case "attraction", "viewpoint", "museum", "gallery", "artwork":
		return 100
	case "information", "picnic_site", "alpine_hut", "wilderness_hut":
		return 80
	}
	if _, ok := tags["historic"]; ok {
		return 95
	}
	switch tags["natural"] {
	case "peak", "saddle", "waterfall", "spring", "cave_entrance", "rock", "stone", "tree":
		return 90
	}
	if tags["mountain_pass"] == "yes" {
		return 90
	}
	switch tags["man_made"] {
	case "tower", "lighthouse", "obelisk", "cross", "watermill", "windmill":
		return 75
	}
	switch tags["amenity"] {
	case "place_of_worship", "fountain":
		return 70
	}
	switch tags["leisure"] {
	case "park", "garden":
		return 65
	}
	return 0
}

func waypointNameFromNominatim(ctx context.Context, lat float64, lon float64) (string, error) {
	baseURL := geocodingExternalServiceBaseURL("NOMINATIM_URL", waypointNameNominatimDefaultURL)
	var lastErr error
	for _, zoom := range waypointNameNominatimZoomLevels {
		name, err := waypointNameFromNominatimAtZoom(ctx, baseURL, lat, lon, zoom)
		if name != "" {
			return name, nil
		}
		if err != nil {
			lastErr = err
			break
		}
	}
	return "", lastErr
}

func waypointNameFromNominatimAtZoom(ctx context.Context, baseURL string, lat float64, lon float64, zoom int) (string, error) {
	requestURL, err := geocodingExternalServiceURL(
		baseURL,
		"/reverse",
		url.Values{
			"lat":            []string{fmt.Sprintf("%.7f", lat)},
			"lon":            []string{fmt.Sprintf("%.7f", lon)},
			"format":         []string{"jsonv2"},
			"addressdetails": []string{"1"},
			"zoom":           []string{fmt.Sprintf("%d", zoom)},
		},
	)
	if err != nil {
		return "", err
	}

	if err := nominatimRateLimit(ctx, baseURL); err != nil {
		return "", err
	}

	var result waypointNameNominatimReverseResponse
	if err := geocodingFetchJSON(ctx, baseURL, requestURL, &result); err != nil {
		return "", err
	}
	return waypointNameFromNominatimResponse(result), nil
}

func waypointNameFromNominatimResponse(result waypointNameNominatimReverseResponse) string {
	if name := cleanWaypointName(result.Name); name != "" {
		return name
	}

	addressKeys := []string{
		"attraction",
		"tourism",
		"historic",
		"amenity",
		"natural",
		"leisure",
		"building",
		"road",
		"pedestrian",
		"footway",
		"path",
		"cycleway",
		"neighbourhood",
		"suburb",
		"hamlet",
		"village",
		"town",
		"city",
		"municipality",
		"county",
		"state",
	}
	for _, key := range addressKeys {
		if name := cleanWaypointName(result.Address[key]); name != "" {
			return name
		}
	}

	for _, part := range strings.Split(result.DisplayName, ",") {
		name := cleanWaypointName(part)
		if name != "" && !looksLikeHouseNumber(name) {
			return name
		}
	}
	return ""
}

func nominatimRateLimit(ctx context.Context, baseURL string) error {
	if !strings.Contains(baseURL, "nominatim.openstreetmap.org") {
		return nil
	}

	waypointNameNominatimMu.Lock()
	defer waypointNameNominatimMu.Unlock()

	wait := waypointNameNominatimMinDelay - time.Since(waypointNameLastNominatimCall)
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	waypointNameLastNominatimCall = time.Now()
	return nil
}

func geocodingFetchJSON(ctx context.Context, baseURL string, requestURL string, out any) error {
	headers := geocodingFetchHeaders()
	if waypointNameHTTPClient != waypointNameDefaultHTTPClient || geocodingUsesTrustedServiceClient(baseURL) {
		return geocodingFetchJSONWithClient(ctx, requestURL, out, headers)
	}

	fetched, err := util.FetchPublicURLWithHeaders(ctx, requestURL, waypointNameHTTPResponseLimit, headers)
	if err != nil {
		return err
	}
	return json.Unmarshal(fetched.Body, out)
}

func geocodingUsesTrustedServiceClient(baseURL string) bool {
	normalized := strings.TrimRight(normalizeGeocodingBaseURL(baseURL), "/")
	return normalized != strings.TrimRight(waypointNameOverpassDefaultURL, "/") &&
		normalized != strings.TrimRight(waypointNameNominatimDefaultURL, "/")
}

func geocodingTrustedServiceRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	if len(via) == 0 {
		return nil
	}
	origin := via[0].URL
	if req.URL.Scheme != origin.Scheme || req.URL.Host != origin.Host {
		return fmt.Errorf("external service redirect changed origin")
	}
	return nil
}

func geocodingFetchJSONWithClient(ctx context.Context, requestURL string, out any, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := waypointNameHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("GET %s failed with status %d: %s", requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(io.LimitReader(resp.Body, waypointNameHTTPResponseLimit)).Decode(out)
}

func geocodingFetchHeaders() map[string]string {
	headers := map[string]string{
		"Accept":     "application/json",
		"User-Agent": "wanderer",
	}
	if origin := strings.TrimRight(os.Getenv("ORIGIN"), "/"); origin != "" {
		headers["Referer"] = origin
	}
	return headers
}

func geocodingExternalServiceBaseURL(key string, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("PUBLIC_" + key))
	}
	if raw == "" {
		raw = fallback
	}
	return normalizeGeocodingBaseURL(raw)
}

func normalizeGeocodingBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(raw), "http://") && !strings.HasPrefix(strings.ToLower(raw), "https://") {
		return "https://" + raw
	}
	return raw
}

func geocodingExternalServiceURL(baseURL string, path string, params url.Values) (string, error) {
	baseURL = normalizeGeocodingBaseURL(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("base URL is empty")
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid base URL %q", baseURL)
	}
	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	requestURL := base.ResolveReference(&url.URL{Path: strings.TrimLeft(path, "/")})
	requestURL.RawQuery = params.Encode()
	return requestURL.String(), nil
}

func cleanWaypointName(name string) string {
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if len([]rune(name)) <= 120 {
		return name
	}
	runes := []rune(name)
	return strings.TrimSpace(string(runes[:120]))
}

func coordinateWaypointName(lat float64, lon float64) string {
	return fmt.Sprintf("%.5f, %.5f", lat, lon)
}

func looksLikeHouseNumber(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	hasDigit := false
	for i, r := range name {
		if r >= '0' && r <= '9' {
			hasDigit = true
			continue
		}
		if i == 0 {
			return false
		}
		if r == ' ' || r == '-' || r == '/' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			continue
		}
		return false
	}
	return hasDigit
}

func logResolvedWaypointName(logger *slog.Logger, source string, name string, lat float64, lon float64, radius float64, err error) {
	if logger == nil {
		return
	}
	args := []any{
		"source", source,
		"name", name,
		"lat", lat,
		"lon", lon,
		"radius", radius,
	}
	if err != nil {
		args = append(args, "fallback_error", err)
	}
	logger.Info("waypoint name resolved", args...)
}
