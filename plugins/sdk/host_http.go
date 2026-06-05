//go:build tinygo

package sdk

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/extism/go-pdk"
)

//go:wasmimport wanderer http_request
func wandererHTTPRequest(uint64) uint64

//go:wasmimport wanderer log
func wandererLog(uint64)

func Log(level LogLevel, message string) {
	entry := HostLogEntry{
		Level:   level,
		Message: message,
	}
	memory, err := pdk.AllocateJSON(entry)
	if err != nil {
		return
	}
	defer memory.Free()
	wandererLog(memory.Offset())
}

func LogDebug(message string) {
	Log(LogLevelDebug, message)
}

func LogInfo(message string) {
	Log(LogLevelInfo, message)
}

func LogWarn(message string) {
	Log(LogLevelWarn, message)
}

func LogError(message string) {
	Log(LogLevelError, message)
}

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

func ConnectorRequest(method string, connector string, path string, query []QueryParam, headers map[string]string, expect ResponseExpect) (HostResponse, []byte, error) {
	return HostRequest(HostRequestSpec{
		Method: method,
		Target: RequestTarget{
			Type:      "connector",
			Connector: connector,
			Path:      path,
			Query:     query,
		},
		Headers: headers,
		Expect:  expect,
	})
}

func Get(connector string, path string, query []QueryParam, headers map[string]string, expect ResponseExpect) (HostResponse, []byte, error) {
	return ConnectorRequest("GET", connector, path, query, headers, expect)
}

func PostJSON(connector string, path string, query []QueryParam, headers map[string]string, body any, expect ResponseExpect) (HostResponse, []byte, error) {
	return HostRequest(HostRequestSpec{
		Method: "POST",
		Target: RequestTarget{
			Type:      "connector",
			Connector: connector,
			Path:      path,
			Query:     query,
		},
		Headers: headers,
		Body: &HostRequestBody{
			Type: HostRequestBodyTypeJSON,
			JSON: body,
		},
		Expect: expect,
	})
}

func PostForm(connector string, path string, query []QueryParam, headers map[string]string, form []FormField, expect ResponseExpect) (HostResponse, []byte, error) {
	return HostRequest(HostRequestSpec{
		Method: "POST",
		Target: RequestTarget{
			Type:      "connector",
			Connector: connector,
			Path:      path,
			Query:     query,
		},
		Headers: headers,
		Body: &HostRequestBody{
			Type: HostRequestBodyTypeForm,
			Form: form,
		},
		Expect: expect,
	})
}

func (r HostResponse) FirstHeader(name string) string {
	values := r.HeaderValuesFor(name)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (r HostResponse) HeaderValuesFor(name string) []string {
	if r.HeaderValues == nil {
		return nil
	}
	if values, ok := r.HeaderValues[name]; ok {
		return values
	}
	for key, values := range r.HeaderValues {
		if strings.EqualFold(key, name) {
			return values
		}
	}
	return nil
}

func Bool(value bool) *bool {
	return &value
}
