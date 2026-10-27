package util

import (
	"errors"
	"testing"

	"github.com/doyensec/safeurl"
)

func TestFederatedMediaReusesThePublicFileConfiguration(t *testing.T) {
	for _, origin := range []string{
		"https://peer.example:8443/api/v1/trail/one",
		"http://peer.example:3000/api/v1/summit-log/one",
		"https://peer.example/api/v1/list/one",
		"https://peer.example:443/api/v1/trail/one",
		"https://peer.example:65535/api/v1/trail/one",
	} {
		config, err := federatedMediaHTTPConfig(origin)
		if err != nil || config != federatedFileHTTPConfig {
			t.Fatalf("origin %q did not reuse the file policy: %v", origin, err)
		}
	}
	for _, origin := range []string{"", "/relative", "ftp://peer.example:8443", "https://user@peer.example:8443", "https://peer.example:0", "https://peer.example:65536", "https://peer.example:word"} {
		if _, err := federatedMediaHTTPConfig(origin); err == nil {
			t.Errorf("invalid origin %q was accepted", origin)
		}
	}
}

func TestFederatedMediaPortRetainsPublicAddressChecks(t *testing.T) {
	for _, rawURL := range []string{
		"https://127.0.0.1:8443/file.jpg",
		"https://10.0.0.1:8443/file.jpg",
		"https://169.254.169.254:8443/latest/meta-data",
		"https://[::1]:8443/file.jpg",
		"https://[::7f00:1]:8443/file.jpg",
		"https://[64:ff9b::7f00:1]:8443/file.jpg",
	} {
		_, cleanup, err := DownloadRemoteSyncFile(t.Context(), rawURL, "https://peer.example:8443/api/v1/trail/one", "photo", 1024)
		cleanup()
		var blockedIP *safeurl.AllowedIPError
		if !errors.As(err, &blockedIP) {
			t.Fatalf("%s was not blocked by the public-IP policy: %v", rawURL, err)
		}
	}
	_, cleanup, err := DownloadRemoteSyncFile(t.Context(), "https://127.0.0.1:8444/file.jpg", "https://peer.example:8443/api/v1/trail/one", "photo", 1024)
	cleanup()
	var blockedIP *safeurl.AllowedIPError
	if !errors.As(err, &blockedIP) {
		t.Fatalf("a different file port did not retain the address policy: %v", err)
	}
}
