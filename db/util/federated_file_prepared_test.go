package util

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestPreparedFederatedFilesNeverFallBackToNetwork(t *testing.T) {
	file, err := filesystem.NewFileFromBytes([]byte("image"), "image.png")
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithPreparedFederatedFiles(t.Context(), []PreparedFederatedFile{{URL: "https://peer.example/media.png", OriginIRI: "https://peer.example/api/v1/trail/a", File: file}})
	got, err := PreparedRemoteSyncFile(ctx, "https://peer.example/media.png", "http://peer.example/api/v1/waypoint/b", 5)
	if err != nil || got != file {
		t.Fatalf("prepared file unavailable: %v %v", got, err)
	}
	for _, tc := range []struct {
		url, origin string
		limit       int64
	}{
		{"https://peer.example/media.png", "https://peer.example:8443/trail", 5},
		{"https://peer.example/media.png", "ftp://peer.example/trail", 5},
		{"https://peer.example/media.png", "https://peer.example/trail", 4},
		{"http://127.0.0.1/unprepared", "https://peer.example/trail", 5},
	} {
		_, err := PreparedRemoteSyncFile(ctx, tc.url, tc.origin, tc.limit)
		if err == nil || (!strings.Contains(err.Error(), "prepared") && !strings.Contains(err.Error(), "size limit")) {
			t.Fatalf("lookup fell through to network: %v", err)
		}
	}
}
