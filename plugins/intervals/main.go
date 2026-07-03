//go:build tinygo

package main

import (
	"encoding/json"
	"errors"

	"github.com/extism/go-pdk"
)

func main() {}

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
		return handleErr(err)
	}
	if err := pdk.OutputJSON(output); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

//export get_activity_detail_v1
func getActivityDetailV1() int32 {
	var input detailInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid detail input: "+err.Error())
	}
	client, err := newClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}

	activity, err := client.activity(input.Summary.Source.ExternalID)
	if err != nil {
		return handleErr(err)
	}

	gpxData, err := client.activityGPX(input.Summary.Source.ExternalID)
	if err != nil {
		return handleErr(err)
	}

	item, err := activityImport(activity, gpxData)
	if err != nil {
		return handleErr(err)
	}

	if err := pdk.OutputJSON(detailOutput{Item: item}); err != nil {
		return fail("internal_error", err.Error())
	}
	return 0
}

func handleErr(err error) int32 {
	var rlErr rateLimitError
	if errors.As(err, &rlErr) {
		return failWithRetry("rate_limited", "intervals.icu rate limit exceeded", rlErr.retryAfter)
	}
	return fail("provider_unavailable", err.Error())
}

func failWithRetry(code string, message string, retryAfter int) int32 {
	errObj := pluginError{Code: code, Message: message}
	if retryAfter > 0 {
		errObj.RetryAfterSeconds = &retryAfter
	}
	data, err := json.Marshal(errObj)
	if err != nil {
		pdk.SetErrorString(message)
		return 1
	}
	pdk.SetErrorString(string(data))
	return 1
}

//export refresh_session_v1
func refreshSessionV1() int32 {
	var input refreshSessionInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid refresh_session input: "+err.Error())
	}

	client, err := newClient(input.Auth)
	if err != nil {
		return fail("auth_failed", err.Error())
	}

	// Test credentials by listing activities
	_, err = client.activities("2020-01-01")
	if err != nil {
		return fail("auth_failed", "failed to verify credentials: "+err.Error())
	}

	if err := pdk.OutputJSON(refreshSessionOutput{
		Token:  client.apiKey,
		Scheme: "Basic",
	}); err != nil {
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
