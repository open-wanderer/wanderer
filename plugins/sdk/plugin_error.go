package sdk

import "encoding/json"

// encodePluginError keeps the wire encoding used by TinyGo failures available
// to the normal Go build, where it can be tested without an Extism runtime.
func encodePluginError(pluginErr PluginError) (string, error) {
	payload, err := json.Marshal(pluginErr)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
