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

//export refresh_session_v1
func refreshSessionV1() int32 {
	var input refreshSessionInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid refresh_session input: "+err.Error())
	}

	email := stringField(input.Auth, "email")
	password := stringField(input.Auth, "password")
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

//export prepare_send_route_v1
func prepareSendRouteV1() int32 {
	var input sendRouteInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid prepare_send_route input: "+err.Error())
	}
	if input.Route.Format != "gpx" || input.Route.ContentBase64 == "" {
		return fail("invalid_request", "a GPX route is required")
	}

	userID, err := userIDForUpload(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}

	plan := uploadPlan{
		Request: sdk.HostRequestSpec{
			Method: "POST",
			URL:    fmt.Sprintf("https://dashboard.hammerhead.io/v1/users/%s/routes/import/file", userID),
			Auth:   "provider_session",
			Body: &sdk.HostRequestBody{
				Type: sdk.HostRequestBodyTypeMultipart,
				Parts: []sdk.MultipartPart{
					{
						Name:        "file",
						Source:      sdk.MultipartSourceRoute,
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

func stringField(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func listRoutes(client hammerheadClient, input listInput) (listOutput, error) {
	page := intState(input.State, "page", 0)
	limit := syncLimit(input)
	rows, totalPages, err := client.tours(page, limit)
	if err != nil {
		return listOutput{}, err
	}

	after := stringField(input.Options, "after")
	known := knownIDs(input.RecentExternalIDs)
	items := make([]trailImport, 0, min(limit, len(rows)))
	for _, row := range rows {
		if known[row.ID] {
			continue
		}
		detail, err := client.tour(row.ID)
		if err != nil {
			continue
		}
		if after != "" && detail.CreatedAt < after {
			return listOutput{Items: items}, nil
		}
		item, err := tourImport(detail)
		if err == nil {
			items = append(items, item)
		}
		if len(items) >= limit {
			break
		}
	}

	nextPage := page + 1
	return listOutput{
		Items:   items,
		State:   map[string]any{"page": nextPage},
		HasMore: nextPage <= totalPages,
	}, nil
}

func listActivities(client hammerheadClient, input listInput) (listOutput, error) {
	page := intState(input.State, "page", 0)
	limit := syncLimit(input)
	rows, totalPages, err := client.activities(page, limit)
	if err != nil {
		return listOutput{}, err
	}

	after := stringField(input.Options, "after")
	known := knownIDs(input.RecentExternalIDs)
	items := make([]trailImport, 0, min(limit, len(rows)))
	for _, row := range rows {
		if known[row.ID] {
			continue
		}
		detail, err := client.activity(row.ID)
		if err != nil {
			continue
		}
		if after != "" && detail.ActivityData.CreatedAt < after {
			return listOutput{Items: items}, nil
		}
		item, err := activityImport(detail)
		if err == nil {
			items = append(items, item)
		}
		if len(items) >= limit {
			break
		}
	}

	nextPage := page + 1
	return listOutput{
		Items:   items,
		State:   map[string]any{"page": nextPage},
		HasMore: nextPage <= totalPages,
	}, nil
}

func tourImport(tour *tour) (trailImport, error) {
	gpxData, err := tourGPX(tour)
	if err != nil {
		return trailImport{}, err
	}
	return trailImport{
		Source: trailImportSource{
			Provider:   "hammerhead",
			ExternalID: tour.ID,
		},
		Kind:         "planned",
		Name:         tour.Name,
		StartedAt:    tour.CreatedAt,
		ActivityType: "biking",
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
	}, nil
}

func activityImport(activity *activity) (trailImport, error) {
	gpxData, err := activityGPX(activity)
	if err != nil {
		return trailImport{}, err
	}
	return trailImport{
		Source: trailImportSource{
			Provider:   "hammerhead",
			ExternalID: activity.ActivityData.ID,
		},
		Kind:         "completed",
		Name:         activity.ActivityData.Name,
		StartedAt:    activity.ActivityData.CreatedAt,
		ActivityType: "biking",
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
	}, nil
}

func syncLimit(input listInput) int {
	if input.Limits.MaxItems > 0 {
		return input.Limits.MaxItems
	}
	return 10
}

func intState(state map[string]any, key string, fallback int) int {
	switch value := state[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return int(parsed)
		}
	}
	return fallback
}

func knownIDs(ids []string) map[string]bool {
	known := make(map[string]bool, len(ids))
	for _, id := range ids {
		known[id] = true
	}
	return known
}
