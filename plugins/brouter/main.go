//go:build tinygo

package main

import (
	"encoding/json"

	"github.com/extism/go-pdk"
)

func main() {}

//export route_v1
func routeV1() int32 {
	var input routingRouteInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid route input: "+err.Error())
	}
	output, err := handleRoute(input)
	if err != nil {
		output = routeOutput{Error: &pluginError{Code: errorCode(err), Message: err.Error()}}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export round_trip_v1
func roundTripV1() int32 {
	var input routingRoundTripInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid round-trip input: "+err.Error())
	}
	output, err := handleRoundTrip(input)
	if err != nil {
		output = roundTripOutput{Error: &pluginError{Code: errorCode(err), Message: err.Error()}}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export profile_introspect_v1
func profileIntrospectV1() int32 {
	var input routingProfileIntrospectInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid profile introspection input: "+err.Error())
	}
	output, err := handleProfileIntrospect(input)
	if err != nil {
		output = profileIntrospectOutput{Error: &pluginError{Code: errorCode(err), Message: err.Error()}}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export profile_prepare_v1
func profilePrepareV1() int32 {
	var input routingProfilePrepareInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid profile preparation input: "+err.Error())
	}
	output, err := handleProfilePrepare(input)
	if err != nil {
		output = profilePrepareOutput{Error: &pluginError{Code: errorCode(err), Message: err.Error()}}
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
