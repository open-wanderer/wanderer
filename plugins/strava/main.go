//go:build tinygo

package main

import (
	"encoding/json"
	"strconv"

	"github.com/extism/go-pdk"
)

func main() {}

//export list_routes_v1
func listRoutesV1() int32 {
	var input listInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid list_routes input: "+err.Error())
	}
	client, err := newClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	output, err := syncRoutes(client, input)
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
	client, err := newClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	output, err := syncActivities(client, input)
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

func getTrailDetail(kind string) int32 {
	var input detailInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid detail input: "+err.Error())
	}
	client, err := newClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	var item trailImport
	switch kind {
	case "planned":
		route, err := client.route(input.Summary.Source.ExternalID)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
		gpxData, err := client.routeGPX(input.Summary.Source.ExternalID)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
		item, err = routeImport(*route, gpxData)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
	case "completed":
		id, err := strconv.ParseInt(input.Summary.Source.ExternalID, 10, 64)
		if err != nil {
			return fail("invalid_request", "invalid activity external id")
		}
		detail, err := client.activity(id)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
		var photos []activityPhoto
		if detail.Photos.Count > 0 {
			photos, _ = client.activityPhotos(id)
		}
		streams, err := client.activityStreams(id)
		if err != nil {
			return fail("provider_unavailable", err.Error())
		}
		item, err = activityImport(detail, streams, photos)
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

func fail(code string, message string) int32 {
	data, err := json.Marshal(pluginError{Code: code, Message: message})
	if err != nil {
		pdk.SetErrorString(message)
		return 1
	}
	pdk.SetErrorString(string(data))
	return 1
}
