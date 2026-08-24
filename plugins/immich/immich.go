//go:build tinygo

package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/open-wanderer/wanderer/plugins/sdk"
)

const (
	immichConnector          = "api"
	immichAuth               = "api_key"
	immichJSONBytes          = 16 * 1024 * 1024
	immichImportSizeOriginal = "original"
	immichImportSizePreview  = "preview"
)

var immichJSONTypes = []string{"application/json"}

func handleAssetLibrary(input assetLibraryInput) (assetLibraryOutput, error) {
	client := immichClient{}
	switch input.Request.Action {
	case "", "candidates":
		return client.candidates(input)
	case "check":
		userID, err := client.check()
		if err != nil {
			return assetLibraryOutput{}, err
		}
		return assetLibraryOutput{UserID: userID}, nil
	case "import":
		return client.importAssets(input)
	case "thumbnail":
		return client.thumbnail(input)
	default:
		return assetLibraryOutput{}, fmt.Errorf("unsupported asset action %q", input.Request.Action)
	}
}

type immichClient struct{}

func (c immichClient) check() (string, error) {
	var user currentUserResponse
	if err := c.getJSON("/api/users/me", nil, &user); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	_, _, err := c.searchAssets(searchTimeWindow{
		takenAfter:  now.Add(-time.Hour).Format(time.RFC3339),
		takenBefore: now.Format(time.RFC3339),
	}, 1)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (c immichClient) candidates(input assetLibraryInput) (assetLibraryOutput, error) {
	req := input.Request
	cfg := configFromInput(input)
	window := candidateWindow(req, cfg)
	candidates, state, hasMore, scanned, err := c.searchAssetCandidates(window, req, cfg, input.State, input.Search)
	if err != nil {
		return assetLibraryOutput{}, err
	}

	return assetLibraryOutput{
		Candidates:    candidates,
		State:         state,
		HasMore:       hasMore,
		Stats:         &sdk.AssetSearchStats{ScannedItems: scanned},
		TakenAfter:    window.takenAfter,
		HasTimestamps: len(req.Points) > 0 && hasPointTimestamps(req.Points),
	}, nil
}

func (c immichClient) importAssets(input assetLibraryInput) (assetLibraryOutput, error) {
	if len(input.Request.AssetIDs) == 0 {
		return assetLibraryOutput{}, nil
	}
	cfg := configFromInput(input)
	assets, err := c.assetsByID(input.Request.AssetIDs)
	if err != nil {
		return assetLibraryOutput{}, err
	}
	photos := make([]sdk.Photo, 0, len(assets))
	omitted := make([]sdk.OmittedAsset, 0)
	for index, asset := range assets {
		photo, ok := photoFromAsset(asset, cfg.ImportSize)
		if ok {
			photos = append(photos, photo)
			continue
		}
		omitted = append(omitted, sdk.OmittedAsset{
			AssetID: input.Request.AssetIDs[index],
			Reason:  "asset cannot be converted to a photo",
		})
	}
	return assetLibraryOutput{Photos: photos, OmittedAssets: omitted}, nil
}

func (c immichClient) thumbnail(input assetLibraryInput) (assetLibraryOutput, error) {
	if len(input.Request.AssetIDs) == 0 {
		return assetLibraryOutput{}, fmt.Errorf("assetIds is required")
	}
	assetID := input.Request.AssetIDs[0]
	return assetLibraryOutput{
		Photos: []sdk.Photo{{
			ExternalID: assetID,
			Filename:   assetID + ".jpg",
			Source: sdk.MediaSource{
				Type: "connector",
				MediaRef: &sdk.MediaRef{
					Connector: immichConnector,
					Auth:      immichAuth,
					Path:      "/api/assets/" + url.PathEscape(assetID) + "/thumbnail",
					Query: []sdk.QueryParam{
						{Name: "size", Value: "preview"},
					},
					AssetID: assetID,
				},
			},
		}},
	}, nil
}

type searchTimeWindow struct {
	takenAfter  string
	takenBefore string
}

func (c immichClient) searchAssets(window searchTimeWindow, maxPages int) ([]immichAsset, bool, error) {
	if maxPages <= 0 {
		maxPages = 1
	}
	assets := []immichAsset{}
	hasMore := false
	page := 1
	for page <= maxPages {
		var response metadataSearchResponse
		request := metadataSearchRequest{
			TakenAfter:  window.takenAfter,
			TakenBefore: window.takenBefore,
			Type:        "IMAGE",
			WithExif:    true,
			IsArchived:  false,
			Page:        page,
			Size:        250,
		}
		err := c.postJSON("/api/search/metadata", request, &response)
		if err != nil {
			return nil, false, err
		}
		assets = append(assets, response.Assets.Items...)
		if response.Assets.NextPage == nil || *response.Assets.NextPage == "" {
			return assets, false, nil
		}
		next, err := strconv.Atoi(*response.Assets.NextPage)
		if err != nil || next <= page {
			return assets, true, nil
		}
		page = next
		hasMore = true
	}
	return assets, hasMore, nil
}

func (c immichClient) searchAssetCandidates(
	window searchTimeWindow,
	req assetLibraryRequest,
	cfg immichConfig,
	state map[string]any,
	limits sdk.AssetSearchLimits,
) ([]assetCandidate, map[string]any, bool, int, error) {
	result, err := searchAssetCandidatePages(req, cfg, state, assetCandidateSearchLimits{
		MaxItems:            limits.MaxItems,
		MaxScannedItems:     limits.MaxScannedItems,
		MaxProviderRequests: limits.MaxProviderRequests,
	}, func(page int) (assetCandidateProviderPage, error) {
		var response metadataSearchResponse
		request := metadataSearchRequest{
			TakenAfter:  window.takenAfter,
			TakenBefore: window.takenBefore,
			Type:        "IMAGE",
			WithExif:    true,
			IsArchived:  false,
			Page:        page,
			Size:        250,
		}
		if err := c.postJSON("/api/search/metadata", request, &response); err != nil {
			return assetCandidateProviderPage{}, err
		}
		return assetCandidateProviderPage{Items: response.Assets.Items, NextPage: response.Assets.NextPage}, nil
	})
	if err != nil {
		return nil, nil, false, 0, err
	}
	return result.Candidates, result.State, result.HasMore, result.ScannedItems, nil
}

func (c immichClient) assetsByID(ids []string) ([]immichAsset, error) {
	result := []immichAsset{}
	for _, id := range ids {
		var asset immichAsset
		if err := c.getJSON("/api/assets/"+url.PathEscape(id), nil, &asset); err != nil {
			return nil, err
		}
		result = append(result, asset)
	}
	return result, nil
}

func (c immichClient) getJSON(path string, query []sdk.QueryParam, out any) error {
	response, body, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "GET",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: immichConnector,
			Path:      path,
			Query:     query,
		},
		Auth: immichAuth,
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Expect: sdk.ResponseExpect{ContentTypes: immichJSONTypes, MaxBytes: immichJSONBytes},
	})
	if err != nil {
		return err
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("immich request failed (%d): %s", response.Status, string(body))
	}
	return json.Unmarshal(body, out)
}

func (c immichClient) postJSON(path string, body any, out any) error {
	response, responseBody, err := sdk.HostRequest(sdk.HostRequestSpec{
		Method: "POST",
		Target: sdk.RequestTarget{
			Type:      "connector",
			Connector: immichConnector,
			Path:      path,
		},
		Auth: immichAuth,
		Headers: map[string]string{
			"Accept": "application/json",
		},
		Body: &sdk.HostRequestBody{
			Type: sdk.HostRequestBodyTypeJSON,
			JSON: body,
		},
		Expect: sdk.ResponseExpect{ContentTypes: immichJSONTypes, MaxBytes: immichJSONBytes},
	})
	if err != nil {
		return err
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("immich request failed (%d): %s", response.Status, string(responseBody))
	}
	return json.Unmarshal(responseBody, out)
}

func configFromInput(input assetLibraryInput) immichConfig {
	return immichConfig{
		TimeWindowMinutes: intConfig(input.Config, "timeWindowMinutes", 30),
		MaxDistanceMeters: intConfig(input.Config, "maxDistanceMeters", 150),
		ImportSize:        normalizedImportSize(stringConfig(input.Config, "importSize")),
		OwnedOnly:         boolConfig(input.Config, "ownedOnly", false),
		UserID:            stringConfig(input.Config, "userId"),
	}
}

func candidateWindow(req assetLibraryRequest, cfg immichConfig) searchTimeWindow {
	if window, ok := explicitRequestTimeWindow(req); ok {
		return window
	}
	if start, end, ok := requestTrailTimeWindow(req); ok {
		window := time.Duration(cfg.TimeWindowMinutes) * time.Minute
		return searchTimeWindow{
			takenAfter:  start.Add(-window).Format(time.RFC3339),
			takenBefore: end.Add(window).Format(time.RFC3339),
		}
	}
	return searchTimeWindow{}
}

func explicitRequestTimeWindow(req assetLibraryRequest) (searchTimeWindow, bool) {
	window := searchTimeWindow{}
	if value := strings.TrimSpace(req.TakenAfter); value != "" {
		takenAfter, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return searchTimeWindow{}, false
		}
		window.takenAfter = takenAfter.Format(time.RFC3339)
	}
	if value := strings.TrimSpace(req.TakenBefore); value != "" {
		takenBefore, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return searchTimeWindow{}, false
		}
		window.takenBefore = takenBefore.Format(time.RFC3339)
	}
	return window, window.takenAfter != "" || window.takenBefore != ""
}

func requestTrailTimeWindow(req assetLibraryRequest) (time.Time, time.Time, bool) {
	startValue := strings.TrimSpace(req.StartedAt)
	endValue := strings.TrimSpace(req.EndedAt)
	if startValue == "" || endValue == "" {
		return time.Time{}, time.Time{}, false
	}
	start, startErr := time.Parse(time.RFC3339, startValue)
	end, endErr := time.Parse(time.RFC3339, endValue)
	if startErr != nil || endErr != nil {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func photoFromAsset(asset immichAsset, importSize string) (sdk.Photo, bool) {
	if asset.ID == "" {
		return sdk.Photo{}, false
	}
	var lat *float64
	var lon *float64
	if asset.ExifInfo.Latitude != nil && asset.ExifInfo.Longitude != nil {
		lat = asset.ExifInfo.Latitude
		lon = asset.ExifInfo.Longitude
	}
	filename := asset.OriginalFileName
	if strings.TrimSpace(filename) == "" {
		filename = asset.ID + ".jpg"
	}
	if importSize == immichImportSizePreview {
		filename = previewFilename(asset)
	}
	return sdk.Photo{
		ExternalID:  asset.ID,
		Filename:    filename,
		ContentType: contentTypeForImportSize(importSize),
		TakenAt:     asset.FileCreatedAt,
		Lat:         lat,
		Lon:         lon,
		Source:      mediaSourceForAsset(asset.ID, importSize),
	}, true
}

func mediaSourceForAsset(assetID string, importSize string) sdk.MediaSource {
	ref := &sdk.MediaRef{
		Connector: immichConnector,
		Auth:      immichAuth,
		AssetID:   assetID,
	}
	if importSize == immichImportSizeOriginal {
		ref.Path = "/api/assets/" + url.PathEscape(assetID) + "/original"
	} else {
		ref.Path = "/api/assets/" + url.PathEscape(assetID) + "/thumbnail"
		ref.Query = []sdk.QueryParam{{Name: "size", Value: "preview"}}
	}
	return sdk.MediaSource{
		Type:     "connector",
		MediaRef: ref,
	}
}

func contentTypeForImportSize(importSize string) string {
	if importSize == immichImportSizePreview {
		return "image/jpeg"
	}
	return ""
}

func previewFilename(asset immichAsset) string {
	filename := strings.TrimSpace(asset.OriginalFileName)
	if filename == "" {
		return asset.ID + ".jpg"
	}
	if dot := strings.LastIndex(filename, "."); dot > 0 {
		filename = filename[:dot]
	}
	if strings.TrimSpace(filename) == "" {
		return asset.ID + ".jpg"
	}
	return filename + ".jpg"
}

func normalizedImportSize(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), immichImportSizeOriginal) {
		return immichImportSizeOriginal
	}
	return immichImportSizePreview
}

func hasPointTimestamps(points []trackPoint) bool {
	for _, point := range points {
		if point.Timestamp != "" {
			return true
		}
	}
	return false
}

func stringConfig(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func boolConfig(values map[string]any, key string, fallback bool) bool {
	value, ok := values[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func intConfig(values map[string]any, key string, fallback int) int {
	switch value := values[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
