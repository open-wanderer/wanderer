//go:build tinygo

package main

import (
	"github.com/extism/go-pdk"
	"github.com/open-wanderer/wanderer/plugins/sdk"
)

func main() {}

//export route_v1
func routeV1() int32 {
	var input routingRouteInput
	if err := pdk.InputJSON(&input); err != nil {
		return sdk.Fail("invalid_request", "invalid route input: "+err.Error())
	}
	output, err := handleRoute(input)
	if err != nil {
		output = routeOutput{Error: brouterPluginError(err)}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return sdk.Fail("internal_error", err.Error())
	}
	return 0
}

//export round_trip_v1
func roundTripV1() int32 {
	var input routingRoundTripInput
	if err := pdk.InputJSON(&input); err != nil {
		return sdk.Fail("invalid_request", "invalid round-trip input: "+err.Error())
	}
	output, err := handleRoundTrip(input)
	if err != nil {
		output = roundTripOutput{Error: brouterPluginError(err)}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return sdk.Fail("internal_error", err.Error())
	}
	return 0
}

//export profile_introspect_v1
func profileIntrospectV1() int32 {
	var input routingProfileIntrospectInput
	if err := pdk.InputJSON(&input); err != nil {
		return sdk.Fail("invalid_request", "invalid profile introspection input: "+err.Error())
	}
	output, err := handleProfileIntrospect(input)
	if err != nil {
		output = profileIntrospectOutput{Error: brouterPluginError(err)}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return sdk.Fail("internal_error", err.Error())
	}
	return 0
}

//export profile_prepare_v1
func profilePrepareV1() int32 {
	var input routingProfilePrepareInput
	if err := pdk.InputJSON(&input); err != nil {
		return sdk.Fail("invalid_request", "invalid profile preparation input: "+err.Error())
	}
	output, err := handleProfilePrepare(input)
	if err != nil {
		output = profilePrepareOutput{Error: brouterPluginError(err)}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return sdk.Fail("internal_error", err.Error())
	}
	return 0
}
