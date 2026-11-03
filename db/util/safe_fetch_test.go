package util

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestNewSafeURLClientBlocksIPv4CompatibleAddress checks that the
// IPv4-compatible forms of 127.0.0.1 and 169.254.169.254 are refused.
func TestNewSafeURLClientBlocksIPv4CompatibleAddress(t *testing.T) {
	client := NewSafeURLClient(time.Second, nil)

	for _, target := range []string{"http://[::7f00:1]/", "http://[::a9fe:a9fe]/"} {
		t.Run(target, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, target, nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
			if err == nil {
				t.Fatalf("request to %s succeeded; want a policy rejection", target)
			}
			if !strings.Contains(err.Error(), "not found in allowlist") {
				t.Fatalf("error = %v; want policy rejection containing %q", err, "not found in allowlist")
			}
		})
	}
}
