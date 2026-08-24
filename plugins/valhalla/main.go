//go:build tinygo

package main

import (
	"encoding/json"
	"errors"

	"github.com/extism/go-pdk"
	"github.com/open-wanderer/wanderer/plugins/sdk"
	valhallacore "github.com/open-wanderer/wanderer/plugins/valhalla/core"
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
		output = routeOutput{Error: &pluginError{Code: "provider_unavailable", Message: err.Error()}}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export elevation_v1
func elevationV1() int32 {
	var input routingElevationInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid elevation input: "+err.Error())
	}
	output, err := handleElevation(input)
	if err != nil {
		output = elevationOutput{Error: &pluginError{Code: "provider_unavailable", Message: err.Error()}}
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export maneuvers_v1
func maneuversV1() int32 {
	var input routingManeuverInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid maneuver input: "+err.Error())
	}
	output, err := handleManeuvers(input)
	if err != nil {
		code := "provider_unavailable"
		var adapterErr *valhallacore.AdapterError
		if errors.As(err, &adapterErr) && adapterErr.Code != "" {
			code = adapterErr.Code
		}
		output = sdk.ManeuverResult{Error: &pluginError{Code: code, Message: err.Error()}}
	}
	if maximum := input.Request.Limits.MaxResponseBytes; maximum > 0 {
		payload, marshalErr := json.Marshal(output)
		if marshalErr != nil {
			return fail("internal_error", marshalErr.Error())
		}
		if int64(len(payload)) > maximum {
			output = sdk.ManeuverResult{Error: &pluginError{Code: "response_limit_exceeded", Message: "maneuver result exceeds the host response limit"}}
		}
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
