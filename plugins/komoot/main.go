//go:build tinygo

package main

import (
	"encoding/json"
	"errors"

	"github.com/extism/go-pdk"
)

func main() {}

//export list_routes_v1
func listRoutesV1() int32 {
	return listTours("planned")
}

//export list_activities_v1
func listActivitiesV1() int32 {
	return listTours("completed")
}

//export get_route_detail_v1
func getRouteDetailV1() int32 {
	return getTourDetail("planned")
}

//export get_activity_detail_v1
func getActivityDetailV1() int32 {
	return getTourDetail("completed")
}

//export refresh_session_v1
func refreshSessionV1() int32 {
	var input refreshSessionInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid refresh_session input: "+err.Error())
	}

	client, err := loginClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}

	if err := pdk.OutputJSON(refreshSessionOutput{
		Token:  client.token,
		Scheme: "Basic",
	}); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

func getTourDetail(kind string) int32 {
	var input detailInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid detail input: "+err.Error())
	}
	client, err := loginClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	item, err := tourDetail(client, input.Summary.Source.ExternalID, kind)
	if err != nil {
		if errors.Is(err, errTourKindMismatch) {
			return fail("not_importable", err.Error())
		}
		return fail("provider_unavailable", err.Error())
	}
	if err := pdk.OutputJSON(detailOutput{Item: item}); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

func listTours(kind string) int32 {
	var input listInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid list input: "+err.Error())
	}
	client, err := loginClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}
	output, err := syncTours(client, input, kind)
	if err != nil {
		return fail("provider_unavailable", err.Error())
	}
	if err := pdk.OutputJSON(output); err != nil {
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
