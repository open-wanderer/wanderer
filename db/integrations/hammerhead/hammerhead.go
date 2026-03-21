package hammerhead

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/tkrajina/gpxgo/gpx"
)

func SyncHammerhead(app core.App) error {
	integrations, err := app.FindAllRecords("integrations", dbx.NewExp("true"))
	if err != nil {
		return err
	}

	for _, i := range integrations {
		encryptionKey := os.Getenv("POCKETBASE_ENCRYPTION_KEY")
		if len(encryptionKey) == 0 {
			return errors.New("POCKETBASE_ENCRYPTION_KEY not set")
		}

		userId := i.GetString("user")
		actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userId)
		if err != nil {
			warning := fmt.Sprintf("no actor found for user: %s\n", userId)
			fmt.Print(warning)
			app.Logger().Warn(warning)
			continue
		}
		actorId := actor.Id

		hammerheadString := i.GetString("hammerhead")
		hammerheadIntegration := HammerheadIntegration{
			Planned:   true,
			Completed: true,
		}
		json.Unmarshal([]byte(hammerheadString), &hammerheadIntegration)

		if !hammerheadIntegration.Active || hammerheadIntegration.Email == "" || hammerheadIntegration.Password == "" {
			continue
		}
		h := &HammerheadApi{}

		decryptedPassword, err := security.Decrypt(hammerheadIntegration.Password, encryptionKey)
		if err != nil {
			warning := fmt.Sprintf("unable to decrypt password: %v\n", err)
			fmt.Print(warning)
			app.Logger().Warn(warning)
			continue
		}

		err = h.Login(hammerheadIntegration.Email, string(decryptedPassword))
		if err != nil {
			warning := fmt.Sprintf("Hammerhead login failed: %v\n", err)
			fmt.Print(warning)
			app.Logger().Warn(warning)
			continue
		}

		page := 0
		totalPages := 0
		stopped := false

		var after int64 = 0
		if hammerheadIntegration.After != "" {
			t, err := time.Parse("2006-01-02", hammerheadIntegration.After)
			if err != nil {
				return err
			}
			t = t.UTC()

			after = t.Unix()
		}

		if hammerheadIntegration.Planned {
			page = 0
			totalPages = 0
			stopped = false

			for page <= totalPages && !stopped {
				curTotalPages := totalPages
				tours, curTotalPages, err := h.fetchTours(page)
				if err != nil {
					warning := fmt.Sprintf("error fetching tours from Hammerhead: %v\n", err)
					fmt.Print(warning)
					app.Logger().Warn(warning)
					break
				}

				if curTotalPages > totalPages {
					totalPages = curTotalPages
				}

				err, stopped = syncTrailWithTours(app, h, actorId, tours, after)
				if err != nil {
					warning := fmt.Sprintf("error syncing Hammerhead tours with trails: %v\n", err)
					fmt.Print(warning)
					app.Logger().Warn(warning)
					break
				}

				page += 1
			}
		}

		if hammerheadIntegration.Completed {
			page = 0
			totalPages = 0
			stopped = false

			for page <= totalPages && !stopped {
				curTotalPages := totalPages
				tours, curTotalPages, err := h.fetchActivities(page)
				if err != nil {
					warning := fmt.Sprintf("error fetching tours from Hammerhead: %v\n", err)
					fmt.Print(warning)
					app.Logger().Warn(warning)
					break
				}

				if curTotalPages > totalPages {
					totalPages = curTotalPages
				}

				err, stopped = syncTrailWithActivities(app, h, actorId, tours, after)
				if err != nil {
					warning := fmt.Sprintf("error syncing Hammerhead tours with trails: %v\n", err)
					fmt.Print(warning)
					app.Logger().Warn(warning)
					break
				}

				page += 1
			}
		}
	}

	return nil
}

type BasicAuthToken struct {
	Key   string
	Value string
}

func (b BasicAuthToken) Apply(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+b.Value)
}

type HammerheadApi struct {
	UserID string
	Token  string
}

func (h *HammerheadApi) buildHeader() *BasicAuthToken {
	if h.UserID != "" && h.Token != "" {
		return &BasicAuthToken{h.UserID, h.Token}
	}
	return nil
}

func getToken(uri string, auth *BasicAuthToken) ([]byte, error) {
	client := &http.Client{}

	var jsonStr = []byte(`{"grant_type": "password", "username": "` + auth.Key + `", "password": "` + auth.Value + `"}`)

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error retrieving auth token from Hammerhead (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (h *HammerheadApi) UploadActivities(e *core.RequestEvent) error {
	files, err := e.FindUploadedFiles("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return apis.NewBadRequestError("file field is required", err)
		}
		return apis.NewBadRequestError("invalid multipart payload", err)
	}

	if len(files) == 0 {
		return apis.NewBadRequestError("file field is required", nil)
	}

	fileToUpload := files[0]
	reader, err := fileToUpload.Reader.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", fileToUpload.OriginalName)
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, reader); err != nil {
		return err
	}

	contentType := writer.FormDataContentType()

	if err := writer.Close(); err != nil {
		return err
	}

	currentURI := fmt.Sprintf("https://dashboard.hammerhead.io/v1/users/%s/routes/import/file", h.UserID)

	if _, err := sendPostRequest(currentURI, &buf, contentType, h.buildHeader()); err != nil {
		return err
	}

	return nil
}

func sendPostRequest(url string, body io.Reader, contentType string, auth *BasicAuthToken) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if auth != nil {
		auth.Apply(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error sending request to Hammerhead (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func sendGetRequest(url string, auth *BasicAuthToken) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if auth != nil {
		auth.Apply(req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error sending request to Hammerhead (%d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

func (h *HammerheadApi) Login(email, password string) error {
	url := "https://dashboard.hammerhead.io/v1/auth/token"

	body, err := getToken(url, &BasicAuthToken{email, password})
	if err != nil {
		return err
	}

	var data LoginResponse
	json.Unmarshal(body, &data)

	h.Token = data.Token
	derivedUserID, err := extractUserIDFromToken(data.Token)
	if err != nil {
		return fmt.Errorf("unable to determine Hammerhead user id automatically: %w", err)
	}
	h.UserID = derivedUserID

	return nil
}

func extractUserIDFromToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", errors.New("token is not a JWT")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("unable to decode JWT payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("unable to decode JWT claims: %w", err)
	}

	if value, ok := claims["sub"].(string); ok && value != "" {
		return value, nil
	}

	return "", errors.New("no sub claim found in token")
}

func (h *HammerheadApi) fetchActivities(page int) ([]HammerheadActivityResponse, int, error) {

	currentUri := fmt.Sprintf("https://dashboard.hammerhead.io/v1/users/%s/activities?perPage=50&page=%d&search=&orderBy=NEWEST&ascending=true", h.UserID, page)

	body, err := sendGetRequest(currentUri, h.buildHeader())
	if err != nil {
		return nil, 0, err
	}

	var data HammerheadActivitiesResponse
	json.Unmarshal(body, &data)

	tours := data.Tours

	return tours, data.Pages, nil
}

func (h *HammerheadApi) fetchTours(page int) ([]HammerheadTourResponse, int, error) {

	currentUri := fmt.Sprintf("https://dashboard.hammerhead.io/v1/users/%s/routes?perPage=50&page=%d&search=&orderBy=NEWEST&ascending=true&exclude=archive", h.UserID, page)
	body, err := sendGetRequest(currentUri, h.buildHeader())
	if err != nil {
		return nil, 0, err
	}

	var data HammerheadToursResponse
	json.Unmarshal(body, &data)

	tours := data.Data

	return tours, data.TotalPages, nil
}

func (h *HammerheadApi) fetchDetailedActivity(tour HammerheadActivityResponse) (*HammerheadActivity, error) {

	url := fmt.Sprintf("https://dashboard.hammerhead.io/v1/users/%s/activities/%s/details", h.UserID, tour.ID)
	body, err := sendGetRequest(url, h.buildHeader())
	if err != nil {
		return nil, err
	}

	var data *HammerheadActivity
	json.Unmarshal(body, &data)
	return data, nil
}

func (h *HammerheadApi) fetchDetailedTour(tour HammerheadTourResponse) (*HammerheadTour, error) {

	url := fmt.Sprintf("https://dashboard.hammerhead.io/v1/users/%s/routes/%s", h.UserID, tour.ID)
	body, err := sendGetRequest(url, h.buildHeader())
	if err != nil {
		return nil, err
	}

	var data *HammerheadTour
	json.Unmarshal(body, &data)
	return data, nil
}

func syncTrailWithTours(app core.App, k *HammerheadApi, actor string, tours []HammerheadTourResponse, after int64) (error, bool) {
	for _, tour := range tours {

		trails, err := app.FindRecordsByFilter("trails", "external_id = {:id}", "", 1, 0, dbx.Params{"id": tour.ID})
		if err != nil {
			return err, true
		}

		if len(trails) != 0 {
			continue
		}

		detailedTour, err := k.fetchDetailedTour(tour)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to fetch details for tour '%s': %v", tour.Name, err))
			continue
		}

		if detailedTour.CreatedAt.Unix() < after {
			return nil, true
		}

		if detailedTour.Distance <= 0 {
			app.Logger().Warn(fmt.Sprintf("Skipping Hammerhead tour '%s' with zero distance", tour.Name))
			continue
		}

		gpx, err := generateTourGPX(detailedTour)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to generate GPX for tour '%s': %v", tour.Name, err))
			continue
		}

		_, err = createTrailFromTour(app, detailedTour, gpx, actor)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to create trail for tour '%s': %v", tour.Name, err))
			continue
		}
	}

	return nil, false
}

func syncTrailWithActivities(app core.App, k *HammerheadApi, actor string, tours []HammerheadActivityResponse, after int64) (error, bool) {
	for _, tour := range tours {

		trails, err := app.FindRecordsByFilter("trails", "external_id = {:id}", "", 1, 0, dbx.Params{"id": tour.ID})
		if err != nil {
			return err, true
		}

		if len(trails) != 0 {
			continue
		}

		detailedTour, err := k.fetchDetailedActivity(tour)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to fetch details for tour '%s': %v", tour.Name, err))
			continue
		}

		if detailedTour.ActivityData.CreatedAt.Unix() < after {
			return nil, true
		}

		distance, ok := activityDistance(detailedTour)
		if !ok || distance <= 0 {
			app.Logger().Warn(fmt.Sprintf("Skipping Hammerhead activity '%s' with zero distance", tour.Name))
			continue
		}

		gpx, err := generateActivityGPX(detailedTour)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to generate GPX for tour '%s': %v", tour.Name, err))
			continue
		}

		_, err = createTrailFromActivity(app, detailedTour, gpx, actor)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to create trail for tour '%s': %v", tour.Name, err))
			continue
		}
	}

	return nil, false
}

func activityDistance(detailedTour *HammerheadActivity) (float64, bool) {
	idDistance := slices.IndexFunc(detailedTour.ActivityData.ActivityInfo, func(c HammerheadInfo) bool { return c.Key == "TYPE_DISTANCE_ID" })
	if idDistance < 0 {
		return 0, false
	}

	return detailedTour.ActivityData.ActivityInfo[idDistance].Value.Value, true
}

func createTrailFromActivity(app core.App, detailedTour *HammerheadActivity, gpx *filesystem.File, actor string) (string, error) {
	trailid := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	collection, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)

	category, _ := app.FindFirstRecordByData("categories", "name", "Biking" /*ToDo: Mapping*/)
	categoryId := ""
	if category != nil {
		categoryId = category.Id
	}

	diffculty := "easy" // ToDo: calculate difficulty

	idDistance := slices.IndexFunc(detailedTour.ActivityData.ActivityInfo, func(c HammerheadInfo) bool { return c.Key == "TYPE_DISTANCE_ID" })
	idElevationGain := slices.IndexFunc(detailedTour.ActivityData.ActivityInfo, func(c HammerheadInfo) bool { return c.Key == "TYPE_ELEVATION_GAIN_ID" })
	idElevationLoss := slices.IndexFunc(detailedTour.ActivityData.ActivityInfo, func(c HammerheadInfo) bool { return c.Key == "TYPE_ELEVATION_LOSS_ID" })

	duration := 0
	for _, lap := range detailedTour.ActivityData.Laps {
		duration += lap.ActiveTime
	}

	startLat := float64(0)
	startLng := float64(0)
	for i, lat := range detailedTour.RecordData.Lat {
		if lat != float64(0) {
			startLat = lat
			startLng = detailedTour.RecordData.Lng[i]
			break
		}
	}

	record.Load(map[string]any{
		"id":                trailid,
		"name":              detailedTour.ActivityData.Name,
		"public":            false,
		"distance":          detailedTour.ActivityData.ActivityInfo[idDistance].Value.Value,
		"elevation_gain":    detailedTour.ActivityData.ActivityInfo[idElevationGain].Value.Value,
		"elevation_loss":    detailedTour.ActivityData.ActivityInfo[idElevationLoss].Value.Value,
		"duration":          duration / 1000,
		"date":              detailedTour.ActivityData.CreatedAt,
		"external_provider": "hammerhead",
		"external_id":       detailedTour.ActivityData.ID,
		"lat":               startLat,
		"lon":               startLng,
		"difficulty":        diffculty,
		"category":          categoryId,
		"author":            actor,
	})

	if gpx != nil {
		record.Set("gpx", gpx)
	}

	if err := app.Save(record); err != nil {
		return "", err
	}

	collection, err = app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		return "", err
	}

	summitLogRecord := core.NewRecord(collection)
	summitLogRecord.Load(map[string]any{
		"distance":       detailedTour.ActivityData.ActivityInfo[idDistance].Value.Value,
		"elevation_gain": detailedTour.ActivityData.ActivityInfo[idElevationGain].Value.Value,
		"elevation_loss": detailedTour.ActivityData.ActivityInfo[idElevationLoss].Value.Value,
		"duration":       duration / 1000,
		"date":           detailedTour.ActivityData.CreatedAt,
		"author":         actor,
		"trail":          trailid,
	})
	if err := app.Save(summitLogRecord); err != nil {
		return "", err
	}

	return trailid, nil
}

func createTrailFromTour(app core.App, detailedTour *HammerheadTour, gpx *filesystem.File, actor string) (string, error) {
	trailid := security.RandomStringWithAlphabet(core.DefaultIdLength, core.DefaultIdAlphabet)

	collection, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		return "", err
	}

	record := core.NewRecord(collection)

	category, _ := app.FindFirstRecordByData("categories", "name", "Biking" /*ToDo: Mapping*/)
	categoryId := ""
	if category != nil {
		categoryId = category.Id
	}

	diffculty := "easy" // ToDo: calculate difficulty

	record.Load(map[string]any{
		"id":                trailid,
		"name":              detailedTour.Name,
		"public":            detailedTour.IsPublic,
		"distance":          detailedTour.Distance,
		"elevation_gain":    detailedTour.Elevation.Gain,
		"elevation_loss":    detailedTour.Elevation.Loss,
		"date":              detailedTour.CreatedAt,
		"external_provider": "hammerhead",
		"external_id":       detailedTour.ID,
		"lat":               detailedTour.StartLocation.Lat,
		"lon":               detailedTour.StartLocation.Lng,
		"difficulty":        diffculty,
		"category":          categoryId,
		"author":            actor,
	})

	if gpx != nil {
		record.Set("gpx", gpx)
	}

	if err := app.Save(record); err != nil {
		return "", err
	}

	return trailid, nil
}

func generateActivityGPX(detailedTour *HammerheadActivity) (*filesystem.File, error) {
	times := len(detailedTour.RecordData.Timestamp)
	if times == 0 {
		return nil, nil
	}

	var points []gpx.GPXPoint
	const zeroEps = 1e-4

	// iterate over timestamps and only add points when lat/lng exist for the same index
	for i := 0; i < times; i++ {
		// ensure we have latitude and longitude for this index
		if i < len(detailedTour.RecordData.Lat) && i < len(detailedTour.RecordData.Lng) {
			lat := detailedTour.RecordData.Lat[i]
			lng := detailedTour.RecordData.Lng[i]

			// exclude near (0,0) garbage points
			if math.Abs(lat) < zeroEps && math.Abs(lng) < zeroEps {
				continue
			}

			t := detailedTour.RecordData.Timestamp[i]

			elevation := float64(0)
			if i < len(detailedTour.RecordData.Elevation) {
				elevation = detailedTour.RecordData.Elevation[i] / 1000.0
			}

			points = append(points, gpx.GPXPoint{
				Point: gpx.Point{
					Latitude:  lat,
					Longitude: lng,
					Elevation: *gpx.NewNullableFloat64(elevation),
				},
				Timestamp: time.Unix(int64(t), 0),
			})
		}
	}

	if len(points) == 0 {
		return nil, nil
	}

	gpxData := &gpx.GPX{
		Version: "1.1",
		Creator: "Hammerhead GPX Exporter",
		Tracks: []gpx.GPXTrack{
			{
				Name: detailedTour.ActivityData.Name,
				Segments: []gpx.GPXTrackSegment{
					{
						Points: points,
					},
				},
			},
		},
	}
	gpxAsXML, err := gpxData.ToXml(gpx.ToXmlParams{Version: "1.1", Indent: true})
	if err != nil {
		return nil, err
	}

	gpxFile, err := filesystem.NewFileFromBytes(gpxAsXML, detailedTour.ActivityData.Name+".gpx")
	if err != nil {
		return nil, err
	}

	return gpxFile, nil
}

func generateTourGPX(detailedTour *HammerheadTour) (*filesystem.File, error) {

	poly := detailedTour.RoutePolyline
	coords, err := decodePolyline(poly)
	if err != nil {
		return nil, fmt.Errorf("decode polyline: %w", err)
	}
	if len(coords) == 0 {
		return nil, nil
	}

	// try to get elevation polyline (adjust field path if your struct differs)
	elevations := []float64{}
	// precision 100 is common for Valhalla elevation encodings; change if needed
	if decoded, err := decodeElevations(detailedTour.Elevation.Polyline, 100000); err == nil {
		elevations = decoded
	}

	// Heuristic: detect if coords are (lng,lat) instead of (lat,lng).
	// Count how many points look valid in each orientation and pick the best.
	validAsLat := 0
	validAsLng := 0
	for _, c := range coords {
		// treat c[0] as lat, c[1] as lng
		if c[0] >= -90 && c[0] <= 90 && c[1] >= -180 && c[1] <= 180 {
			validAsLat++
		}
		// treat c[1] as lat, c[0] as lng (swapped)
		if c[1] >= -90 && c[1] <= 90 && c[0] >= -180 && c[0] <= 180 {
			validAsLng++
		}
	}
	swap := false
	if validAsLng > validAsLat {
		swap = true
	}

	var points []gpx.GPXPoint
	for i, c := range coords {
		lat := c[0]
		lng := c[1]
		if swap {
			lat, lng = c[1], c[0]
		}

		// choose elevation:
		elevation := 0.0
		if len(elevations) == len(coords) {
			elevation = elevations[i]
		} else if len(elevations) > 0 {
			// map index proportionally if lengths differ
			j := int(math.Round(float64(i) * float64(len(elevations)-1) / float64(len(coords)-1)))
			if j < 0 {
				j = 0
			}
			if j >= len(elevations) {
				j = len(elevations) - 1
			}
			elevation = elevations[j]
		}

		points = append(points, gpx.GPXPoint{
			Point: gpx.Point{
				Latitude:  lat,
				Longitude: lng,
				Elevation: *gpx.NewNullableFloat64(elevation),
			},
		})
	}

	gpxData := &gpx.GPX{
		Version: "1.1",
		Creator: "Hammerhead GPX Exporter",
		Tracks: []gpx.GPXTrack{
			{
				Name: detailedTour.Name,
				Segments: []gpx.GPXTrackSegment{
					{
						Points: points,
					},
				},
			},
		},
	}
	gpxAsXML, err := gpxData.ToXml(gpx.ToXmlParams{Version: "1.1", Indent: true})
	if err != nil {
		return nil, err
	}

	gpxFile, err := filesystem.NewFileFromBytes(gpxAsXML, detailedTour.Name+".gpx")
	if err != nil {
		return nil, err
	}

	return gpxFile, nil
}

// decodePolyline decodes an encoded polyline string (Google Polyline Algorithm)
// returns slice of [lat, lng] pairs (precision 1e5).
func decodePolyline(s string) ([][2]float64, error) {
	if s == "" {
		return nil, nil
	}
	var coords [][2]float64
	index := 0
	lat := 0
	lng := 0
	for index < len(s) {
		// decode latitude
		result := 0
		shift := uint(0)
		for {
			if index >= len(s) {
				return nil, fmt.Errorf("invalid polyline encoding")
			}
			b := int(s[index]) - 63
			index++
			result |= (b & 0x1F) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		dlat := (result >> 1) ^ (-(result & 1))
		lat += dlat

		// decode longitude
		result = 0
		shift = 0
		for {
			if index >= len(s) {
				return nil, fmt.Errorf("invalid polyline encoding")
			}
			b := int(s[index]) - 63
			index++
			result |= (b & 0x1F) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		dlng := (result >> 1) ^ (-(result & 1))
		lng += dlng

		coords = append(coords, [2]float64{float64(lat) / 1e5, float64(lng) / 1e5})
	}

	// Auto-normalize scale if values are out of realistic lat/lon ranges.
	// Some providers use different precision/scales; repeatedly divide by 10
	// until all values fit into valid ranges.
	if len(coords) > 0 {
		maxLat := 0.0
		maxLng := 0.0
		for _, c := range coords {
			if abs := math.Abs(c[0]); abs > maxLat {
				maxLat = abs
			}
			if abs := math.Abs(c[1]); abs > maxLng {
				maxLng = abs
			}
		}
		// If values are too large (e.g. > 90 lat or > 180 lon), rescale down.
		for (maxLat > 90.0 || maxLng > 180.0) && (maxLat > 0 && maxLng > 0) {
			for i := range coords {
				coords[i][0] /= 10.0
				coords[i][1] /= 10.0
			}
			maxLat /= 10.0
			maxLng /= 10.0
		}
	}

	return coords, nil
}

// decodeElevations decodes a single-dimension delta-encoded polyline string.
// precision is the divisor (e.g. 100 for centi-meters -> meters). Returns elevation values in same units as precision (meters if precision=100).
func decodeElevations(s string, precision float64) ([]float64, error) {
	if s == "" {
		return nil, nil
	}
	var elevs []float64
	index := 0
	val := 0
	for index < len(s) {
		result := 0
		shift := uint(0)
		for {
			if index >= len(s) {
				return nil, fmt.Errorf("invalid elevation encoding")
			}
			b := int(s[index]) - 63
			index++
			result |= (b & 0x1F) << shift
			shift += 5
			if b < 0x20 {
				break
			}
		}
		d := (result >> 1) ^ (-(result & 1))
		val += d
		elevs = append(elevs, float64(val)/precision)
	}
	return elevs, nil
}
