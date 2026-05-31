//go:build tinygo

package sdk

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/extism/go-pdk"
)

//go:wasmimport wanderer http_request
func wandererHTTPRequest(uint64) uint64

func HostRequest(spec HostRequestSpec) (HostResponse, []byte, error) {
	requestMemory, err := pdk.AllocateJSON(spec)
	if err != nil {
		return HostResponse{}, nil, err
	}
	defer requestMemory.Free()

	responsePointer := wandererHTTPRequest(requestMemory.Offset())
	if responsePointer == 0 {
		return HostResponse{}, nil, fmt.Errorf("host http request returned no response")
	}
	responseMemory := pdk.FindMemory(responsePointer)
	var response HostResponse
	if err := json.Unmarshal(responseMemory.ReadBytes(), &response); err != nil {
		return HostResponse{}, nil, err
	}
	if response.Error != nil {
		return response, nil, fmt.Errorf("%s: %s", response.Error.Code, response.Error.Message)
	}
	body, err := base64.StdEncoding.DecodeString(response.BodyBase64)
	if err != nil {
		return response, nil, err
	}
	return response, body, nil
}

func Get(url string, headers map[string]string, expect ResponseExpect) (HostResponse, []byte, error) {
	return HostRequest(HostRequestSpec{
		Method:  "GET",
		URL:     url,
		Headers: headers,
		Expect:  expect,
	})
}

func PostJSON(url string, headers map[string]string, body any, expect ResponseExpect) (HostResponse, []byte, error) {
	return HostRequest(HostRequestSpec{
		Method:  "POST",
		URL:     url,
		Headers: headers,
		Body: &HostRequestBody{
			Type: HostRequestBodyTypeJSON,
			JSON: body,
		},
		Expect: expect,
	})
}
