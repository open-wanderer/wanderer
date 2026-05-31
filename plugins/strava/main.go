//go:build tinygo

package main

import (
	"encoding/json"

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

func fail(code string, message string) int32 {
	data, err := json.Marshal(pluginError{Code: code, Message: message})
	if err != nil {
		pdk.SetErrorString(message)
		return 1
	}
	pdk.SetErrorString(string(data))
	return 1
}
