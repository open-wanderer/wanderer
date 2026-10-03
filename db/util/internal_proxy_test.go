package util

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestRequireInternalProxy(t *testing.T) {
	tests := []struct {
		name   string
		secret string
		header string
		want   error
	}{
		{"secret not configured", "", "anything", ErrProxySecretNotConfigured},
		{"secret not configured, no header", "", "", ErrProxySecretNotConfigured},
		{"missing header", "s3cret", "", ErrInvalidInternalSecret},
		{"wrong header", "s3cret", "nope", ErrInvalidInternalSecret},
		{"prefix of secret", "s3cret", "s3cre", ErrInvalidInternalSecret},
		{"matching header", "s3cret", "s3cret", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_PROXY_SECRET", tc.secret)
			req := httptest.NewRequest("POST", "/activitypub/instance/inbox", nil)
			if tc.header != "" {
				req.Header.Set("X-Internal-Secret", tc.header)
			}
			err := RequireInternalProxy(req)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("expected %v, got %v", tc.want, err)
			}
		})
	}
}
