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
	assetPluginOverpassDefaultURL  = "https://overpass-api.de"
	assetPluginNominatimDefaultURL = "https://nominatim.openstreetmap.org"
	assetPluginNominatimMinDelay   = time.Second
	assetPluginHTTPResponseLimit   = 1 << 20
)

var (
	assetPluginWaypointNameDefaultHTTPClient = &http.Client{Timeout: 8 * time.Second}
	assetPluginWaypointNameHTTPClient        = assetPluginWaypointNameDefaultHTTPClient
	assetPluginNominatimMu                   sync.Mutex
	assetPluginLastNominatimCall             time.Time
	assetPluginNominatimZoomLevels           = []int{18, 16, 14, 12, 10}
)

type assetPluginOverpassResponse struct {
	Elements []assetPluginOverpassElement `json:"elements"`
}

type assetPluginOverpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *assetPluginCoord `json:"center"`
	Tags   map[string]string `json:"tags"`
}

type assetPluginCoord struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type assetPluginNominatimReverseResponse struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Address     map[string]string `json:"address"`
}

func assetPluginWaypointName(ctx context.Context, logger *slog.Logger, lat float64, lon float64, radius float64) (string, error) {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return "", nil
	}

	poiName, poiErr := assetPluginOverpassPOIName(ctx, lat, lon, radius)
	if poiName != "" {
		assetPluginLogResolved(logger, "overpass", poiName, lat, lon, radius, nil)
		return poiName, nil
	}

	reverseName, reverseErr := assetPluginNominatimReverseName(ctx, lat, lon)
	if reverseName != "" {
		assetPluginLogResolved(logger, "nominatim", reverseName, lat, lon, radius, nil)
		return reverseName, nil
	}

	fallbackName := assetPluginCoordinateWaypointName(lat, lon)
	if reverseErr != nil {
		assetPluginLogResolved(logger, "coordinate", fallbackName, lat, lon, radius, reverseErr)
		return fallbackName, nil
	}
	if poiErr != nil {
		assetPluginLogResolved(logger, "coordinate", fallbackName, lat, lon, radius, poiErr)
		return fallbackName, nil
	}
	assetPluginLogResolved(logger, "coordinate", fallbackName, lat, lon, radius, nil)
	return fallbackName, nil
}

func assetPluginOverpassPOIName(ctx context.Context, lat float64, lon float64, radius float64) (string, error) {
	radiusMeters := int(math.Ceil(radius))
	if radiusMeters <= 0 {
		return "", nil
	}

	baseURL := assetPluginExternalServiceBaseURL("OVERPASS_API_URL", assetPluginOverpassDefaultURL)
	requestURL, err := assetPluginExternalServiceURL(
		baseURL,
		"/api/interpreter",
		url.Values{"data": []string{assetPluginOverpassPOIQuery(lat, lon, radiusMeters)}},
	)
	if err != nil {
		return "", err
	}

	var result assetPluginOverpassResponse
	if err := assetPluginFetchJSON(ctx, requestURL, &result); err != nil {
		return "", err
	}
	return assetPluginBestOverpassPOIName(result.Elements, lat, lon, radius), nil
}

func assetPluginOverpassPOIQuery(lat float64, lon float64, radiusMeters int) string {
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

func assetPluginBestOverpassPOIName(elements []assetPluginOverpassElement, lat float64, lon float64, radius float64) string {
	bestName := ""
	bestScore := 0
	bestDistance := math.MaxFloat64
	for _, element := range elements {
		name := assetPluginCleanWaypointName(element.Tags["name"])
		if name == "" {
			continue
		}
		poiLat, poiLon, ok := assetPluginOverpassElementCoord(element)
		if !ok {
			continue
		}
		distance := util.HaversineDistanceMeters(lat, lon, poiLat, poiLon)
		if radius > 0 && distance > radius {
			continue
		}
		score := assetPluginPOIScore(element.Tags)
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

func assetPluginOverpassElementCoord(element assetPluginOverpassElement) (float64, float64, bool) {
	if element.Type == "node" {
		return element.Lat, element.Lon, element.Lat >= -90 && element.Lat <= 90 && element.Lon >= -180 && element.Lon <= 180
	}
	if element.Center != nil {
		return element.Center.Lat, element.Center.Lon, element.Center.Lat >= -90 && element.Center.Lat <= 90 && element.Center.Lon >= -180 && element.Center.Lon <= 180
	}
	return 0, 0, false
}

func assetPluginPOIScore(tags map[string]string) int {
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

func assetPluginNominatimReverseName(ctx context.Context, lat float64, lon float64) (string, error) {
	baseURL := assetPluginExternalServiceBaseURL("NOMINATIM_URL", assetPluginNominatimDefaultURL)
	var lastErr error
	for _, zoom := range assetPluginNominatimZoomLevels {
		name, err := assetPluginNominatimReverseNameAtZoom(ctx, baseURL, lat, lon, zoom)
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

func assetPluginNominatimReverseNameAtZoom(ctx context.Context, baseURL string, lat float64, lon float64, zoom int) (string, error) {
	requestURL, err := assetPluginExternalServiceURL(
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

	if err := assetPluginNominatimRateLimit(ctx, baseURL); err != nil {
		return "", err
	}

	var result assetPluginNominatimReverseResponse
	if err := assetPluginFetchJSON(ctx, requestURL, &result); err != nil {
		return "", err
	}
	return assetPluginNominatimWaypointName(result), nil
}

func assetPluginNominatimWaypointName(result assetPluginNominatimReverseResponse) string {
	if name := assetPluginCleanWaypointName(result.Name); name != "" {
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
		if name := assetPluginCleanWaypointName(result.Address[key]); name != "" {
			return name
		}
	}

	for _, part := range strings.Split(result.DisplayName, ",") {
		name := assetPluginCleanWaypointName(part)
		if name != "" && !assetPluginLooksLikeHouseNumber(name) {
			return name
		}
	}
	return ""
}

func assetPluginNominatimRateLimit(ctx context.Context, baseURL string) error {
	if !strings.Contains(baseURL, "nominatim.openstreetmap.org") {
		return nil
	}

	assetPluginNominatimMu.Lock()
	defer assetPluginNominatimMu.Unlock()

	wait := assetPluginNominatimMinDelay - time.Since(assetPluginLastNominatimCall)
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	assetPluginLastNominatimCall = time.Now()
	return nil
}

func assetPluginFetchJSON(ctx context.Context, requestURL string, out any) error {
	headers := assetPluginFetchHeaders()
	if assetPluginWaypointNameHTTPClient != assetPluginWaypointNameDefaultHTTPClient {
		return assetPluginFetchJSONWithClient(ctx, requestURL, out, headers)
	}

	fetched, err := util.FetchPublicURLWithHeaders(ctx, requestURL, assetPluginHTTPResponseLimit, headers)
	if err != nil {
		return err
	}
	return json.Unmarshal(fetched.Body, out)
}

func assetPluginFetchJSONWithClient(ctx context.Context, requestURL string, out any, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := assetPluginWaypointNameHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("GET %s failed with status %d: %s", requestURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(io.LimitReader(resp.Body, assetPluginHTTPResponseLimit)).Decode(out)
}

func assetPluginFetchHeaders() map[string]string {
	headers := map[string]string{
		"Accept":     "application/json",
		"User-Agent": "wanderer",
	}
	if origin := strings.TrimRight(os.Getenv("ORIGIN"), "/"); origin != "" {
		headers["Referer"] = origin
	}
	return headers
}

func assetPluginExternalServiceBaseURL(key string, fallback string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("PUBLIC_" + key))
	}
	if raw == "" {
		raw = fallback
	}
	return assetPluginNormalizeBaseURL(raw)
}

func assetPluginNormalizeBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(raw), "http://") && !strings.HasPrefix(strings.ToLower(raw), "https://") {
		return "https://" + raw
	}
	return raw
}

func assetPluginExternalServiceURL(baseURL string, path string, params url.Values) (string, error) {
	baseURL = assetPluginNormalizeBaseURL(baseURL)
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

func assetPluginCleanWaypointName(name string) string {
	name = strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if len([]rune(name)) <= 120 {
		return name
	}
	runes := []rune(name)
	return strings.TrimSpace(string(runes[:120]))
}

func assetPluginCoordinateWaypointName(lat float64, lon float64) string {
	return fmt.Sprintf("%.5f, %.5f", lat, lon)
}

func assetPluginLooksLikeHouseNumber(name string) bool {
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

func assetPluginLogResolved(logger *slog.Logger, source string, name string, lat float64, lon float64, radius float64, err error) {
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
	logger.Info("asset plugin waypoint name resolved", args...)
}
