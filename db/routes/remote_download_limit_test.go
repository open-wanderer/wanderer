package routes

import (
	"pocketbase/util"
	"testing"

	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestRemoteDownloadAdapterRetainsGPXSizeLimit(t *testing.T) {
	for _, size := range []int64{remoteGPXMaxBytes, remoteGPXMaxBytes + 1} {
		file, err := filesystem.NewFileFromBytes(make([]byte, size), "route.gpx")
		if err != nil {
			t.Fatal(err)
		}
		origin := "https://peer.example:8443"
		rawURL := origin + "/api/v1/files/trails/remote/route.gpx"
		ctx := util.WithPreparedFederatedFiles(t.Context(), []util.PreparedFederatedFile{{URL: rawURL, OriginIRI: origin, File: file}})
		got, err := downloadFile(ctx, origin, "trails", "remote", "route.gpx")
		if size == remoteGPXMaxBytes && (err != nil || got != file) {
			t.Fatalf("exact-limit file rejected: %v", err)
		}
		if size > remoteGPXMaxBytes && err == nil {
			t.Fatal("downloadFile accepted bytes beyond the GPX field limit")
		}
	}
}
