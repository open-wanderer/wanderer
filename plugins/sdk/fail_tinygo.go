//go:build tinygo

package sdk

import "github.com/extism/go-pdk"

// Fail returns a structured plugin failure through Extism's error channel.
// Plugin entrypoints can use this instead of maintaining their own
// PluginError marshaling glue.
func Fail(code string, message string) int32 {
	data, err := encodePluginError(PluginError{Code: code, Message: message})
	if err != nil {
		pdk.SetErrorString(message)
		return 1
	}
	pdk.SetErrorString(data)
	return 1
}
