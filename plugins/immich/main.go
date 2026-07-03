//go:build tinygo

package main

import (
	"encoding/json"

	"github.com/extism/go-pdk"
)

func main() {}

//export asset_library_v1
func assetLibraryV1() int32 {
	var input assetLibraryInput
	if err := pdk.InputJSON(&input); err != nil {
		return fail("invalid_request", "invalid asset_library input: "+err.Error())
	}

	output, err := handleAssetLibrary(input)
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
