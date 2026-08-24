package sdk

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPluginErrorWireContract(t *testing.T) {
	fixture := maneuverContractFixture(t)
	var expectedFields []string
	if err := json.Unmarshal(fixture["pluginError"], &expectedFields); err != nil {
		t.Fatalf("decode plugin error field contract: %v", err)
	}
	assertJSONTagSet(t, "PluginError", reflect.TypeOf(PluginError{}), expectedFields)
}

func TestEncodePluginError(t *testing.T) {
	retryAfter := 45
	tests := []struct {
		name      string
		pluginErr PluginError
		want      string
	}{
		{
			name:      "code only",
			pluginErr: PluginError{Code: "internal_error"},
			want:      `{"code":"internal_error"}`,
		},
		{
			name:      "message escaping",
			pluginErr: PluginError{Code: "invalid_request", Message: `invalid "profile"`},
			want:      `{"code":"invalid_request","message":"invalid \"profile\""}`,
		},
		{
			name:      "retry hint",
			pluginErr: PluginError{Code: "rate_limited", Message: "slow down", RetryAfterSeconds: &retryAfter},
			want:      `{"code":"rate_limited","message":"slow down","retryAfterSeconds":45}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := encodePluginError(test.pluginErr)
			if err != nil {
				t.Fatalf("encode plugin error: %v", err)
			}
			if got != test.want {
				t.Fatalf("encoded plugin error = %s, want %s", got, test.want)
			}
		})
	}
}
