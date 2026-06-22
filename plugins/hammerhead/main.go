//go:build tinygo

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/extism/go-pdk"
	"github.com/open-wanderer/wanderer/plugins/sdk"
)

func main() {}

//export list_routes_v1
func listRoutesV1() int32 {
	var input listInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid list_routes input: "+err.Error())
	}
	client, err := loginClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	output, err := listRoutes(client, input)
	if err != nil {
		return fail("provider_unavailable", err.Error())
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export list_activities_v1
func listActivitiesV1() int32 {
	var input listInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid list_activities input: "+err.Error())
	}
	client, err := loginClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	output, err := listActivities(client, input)
	if err != nil {
		return fail("provider_unavailable", err.Error())
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export get_route_detail_v1
func getRouteDetailV1() int32 {
	return getTrailDetail("planned")
}

//export get_activity_detail_v1
func getActivityDetailV1() int32 {
	return getTrailDetail("completed")
}

//export refresh_session_v1
func refreshSessionV1() int32 {
	var input refreshSessionInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid refresh_session input: "+err.Error())
	}

	email := sdk.StringField(input.Auth, "email")
	password := sdk.StringField(input.Auth, "password")
	if email == "" || password == "" {
		return fail("auth_failed", "email and password are required")
	}

	token, err := login(email, password)
	if err != nil {
		return fail("auth_failed", err.Error())
	}

	if err := pdk.OutputJSON(refreshSessionOutput{
		Token:  token,
		Scheme: sdk.AuthSchemeBearer,
	}); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

func getTrailDetail(kind string) int32 {
	var input detailInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid detail input: "+err.Error())
	}
	client, err := loginClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	var item trailImport
	switch kind {
	case "planned":
		detail, err := client.tour(input.Summary.Source.ExternalID)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
		item, err = tourImport(detail)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
	case "completed":
		detail, err := client.activity(input.Summary.Source.ExternalID)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
		item, err = activityImport(detail)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
	default:
		return fail("invalid_request", "unsupported detail kind")
	}
	if err := pdk.OutputJSON(detailOutput{Item: item}); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export prepare_trail_send_v1
func prepareTrailSendV1() int32 {
	var input trailSendInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid prepare_trail_send input: "+err.Error())
	}
	if input.Trail.Format != "gpx" || input.Trail.ContentBase64 == "" {
		return fail("invalid_request", "a GPX trail is required")
	}

	userID, err := userIDForUpload(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}

	plan := trailSendPlan{
		Request: sdk.HostRequestSpec{
			Method: "POST",
			Target: sdk.RequestTarget{
				Type:      "connector",
				Connector: "api",
				Path:      fmt.Sprintf("/v1/users/%s/routes/import/file", userID),
			},
			Auth: "provider_session",
			Body: &sdk.HostRequestBody{
				Type: sdk.HostRequestBodyTypeMultipart,
				Parts: []sdk.MultipartPart{
					{
						Name:        "file",
						Source:      sdk.MultipartSourceTrail,
						Filename:    trailGPXFilename(input.Name),
						ContentType: "application/gpx+xml",
					},
				},
			},
			Expect: sdk.ResponseExpect{
				ContentTypes: []string{"application/json"},
				MaxBytes:     1048576,
			},
		},
	}
	if err := pdk.OutputJSON(plan); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

func fail(code string, message string) int32 {
	data, err := json.Marshal(pluginError{Code: code, Message: message})
	if err != nil {
		pdk.SetErrorString(message)
		return 1
	}
	pdk.SetErrorString(string(data))
	return 1
}

func listRoutes(client hammerheadClient, input listInput) (listOutput, error) {
	page := sdk.IntState(input.State, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := sdk.SyncLimit(input)
	rows, totalPages, err := client.tours(page, limit)
	if err != nil {
		return listOutput{}, err
	}

	after := sdk.StringField(input.Options, "after")
	items := make([]trailSummary, 0, min(limit, len(rows)))
	for _, row := range rows {
		if after != "" && row.CreatedAt < after {
			return listOutput{Items: items}, nil
		}
		items = append(items, trailSummary{
			Source: trailImportSource{Provider: "hammerhead", ExternalID: row.ID},
			Kind:   "planned",
		})
		if len(items) >= limit {
			break
		}
	}

	nextPage := page + 1
	hasMore := nextPage <= totalPages
	return listOutput{
		Items:   items,
		State:   sdk.NextPageState(nextPage, hasMore),
		HasMore: hasMore,
	}, nil
}

func listActivities(client hammerheadClient, input listInput) (listOutput, error) {
	page := sdk.IntState(input.State, "page", 1)
	if page <= 0 {
		page = 1
	}
	limit := sdk.SyncLimit(input)
	rows, totalPages, err := client.activities(page, limit)
	if err != nil {
		return listOutput{}, err
	}

	after := sdk.StringField(input.Options, "after")
	items := make([]trailSummary, 0, min(limit, len(rows)))
	for _, row := range rows {
		if after != "" && row.CreatedAt < after {
			return listOutput{Items: items}, nil
		}
		items = append(items, trailSummary{
			Source: trailImportSource{Provider: "hammerhead", ExternalID: row.ID},
			Kind:   "completed",
		})
		if len(items) >= limit {
			break
		}
	}

	nextPage := page + 1
	hasMore := nextPage <= totalPages
	return listOutput{
		Items:   items,
		State:   sdk.NextPageState(nextPage, hasMore),
		HasMore: hasMore,
	}, nil
}

func tourImport(tour *tour) (trailImport, error) {
	gpxData, err := tourGPX(tour)
	if err != nil {
		return trailImport{}, err
	}
	privacy := privacyFromPublic(tour.IsPublic)
	return trailImport{
		Source: trailImportSource{
			Provider:   "hammerhead",
			ExternalID: tour.ID,
		},
		Kind:         "planned",
		Name:         tour.Name,
		StartedAt:    tour.CreatedAt,
		ActivityType: "biking",
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Metadata: map[string]any{
			"distance":         tour.Distance,
			"elevationGain":    tour.Elevation.Gain,
			"elevationLoss":    tour.Elevation.Loss,
			"providerCategory": "biking",
		},
	}, nil
}

func activityImport(activity *activity) (trailImport, error) {
	gpxData, err := activityGPX(activity)
	if err != nil {
		return trailImport{}, err
	}
	privacy := "private"
	return trailImport{
		Source: trailImportSource{
			Provider:   "hammerhead",
			ExternalID: activity.ActivityData.ID,
		},
		Kind:         "completed",
		Name:         activity.ActivityData.Name,
		StartedAt:    activity.ActivityData.CreatedAt,
		ActivityType: "biking",
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Metadata: map[string]any{
			"distance":         infoValueOrZero(activity, "TYPE_DISTANCE_ID"),
			"elevationGain":    infoValueOrZero(activity, "TYPE_ELEVATION_GAIN_ID"),
			"elevationLoss":    infoValueOrZero(activity, "TYPE_ELEVATION_LOSS_ID"),
			"duration":         activityDurationSeconds(activity),
			"providerCategory": "biking",
		},
	}, nil
}

func privacyFromPublic(public bool) string {
	if public {
		return "public"
	}
	return "private"
}

func activityInfoValue(activity *activity, key string) (float64, bool) {
	for _, info := range activity.ActivityData.ActivityInfo {
		if info.Key == key {
			return info.Value.Value, true
		}
	}
	return 0, false
}

func infoValueOrZero(activity *activity, key string) float64 {
	value, _ := activityInfoValue(activity, key)
	return value
}

func activityDurationSeconds(activity *activity) float64 {
	var total int
	for _, lap := range activity.ActivityData.Laps {
		total += lap.ActiveTime
	}
	if total > 0 {
		return float64(total) / 1000
	}
	if activity.ActivityData.Duration.ElapsedTime > 0 {
		return float64(activity.ActivityData.Duration.ElapsedTime) / 1000
	}
	return 0
}
