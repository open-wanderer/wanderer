package importer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"
)

func TestPhotoOmissionLogsKeepURLFreeFailureCauses(t *testing.T) {
	oldFetch, oldDelays := fetchPublicPluginMedia, pluginMediaRetryDelays
	t.Cleanup(func() { fetchPublicPluginMedia = oldFetch; pluginMediaRetryDelays = oldDelays })
	pluginMediaRetryDelays = nil
	for _, test := range []struct {
		name, reason, cause, sourceType, sourceURL string
		status                                     int
		expired                                    bool
		failure                                    func(string) error
		retryable                                  bool
	}{
		{name: "rate limit", reason: "download_failed", cause: "HTTP status 429", status: 429, retryable: true},
		{name: "provider outage", reason: "download_failed", cause: "HTTP status 503", status: 503, retryable: true},
		{name: "inner network error", reason: "download_failed", cause: "lookup b.example: no such host", retryable: true, failure: func(raw string) error {
			return &url.Error{Op: "Get", URL: raw, Err: &net.DNSError{Err: "no such host", Name: "b.example", IsNotFound: true}}
		}},
		{name: "local error", reason: "download_failed", cause: "photo file unavailable", retryable: true, failure: func(string) error { return errors.New("photo file unavailable") }},
		{name: "local URL context", reason: "download_failed", cause: "photo file unavailable at [url]", retryable: true, failure: func(raw string) error { return fmt.Errorf("photo file unavailable at %s", raw) }},
		{name: "expired source", reason: "source_expired", cause: "photo source expired", expired: true},
		{name: "empty source URL", reason: "invalid_source", cause: "invalid photo source: URL is empty", sourceURL: "empty"},
		{name: "invalid source URL", reason: "invalid_source", cause: "invalid URL escape \"%zz\"", sourceURL: "https://%zz/photo.jpg?key=photo-credential"},
		{name: "invalid source scheme", reason: "invalid_source", cause: "invalid photo source: unsupported media URL scheme \"ftp\"", sourceURL: "ftp://b.example/photo.jpg?key=photo-credential"},
		{name: "unknown source type", reason: "invalid_source", cause: "invalid photo source: unsupported source type \"other\"", sourceType: "other"},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := newPhotoAssetImportTestApp(t)
			var output bytes.Buffer
			app := photoOmissionLogApp{App: base, log: slog.New(slog.NewJSONHandler(&output, nil))}
			photo := pluginsystem.Photo{ExternalID: "camera-photo", Filename: "camera.jpg", Source: pluginsystem.MediaSource{Type: "url", URL: "https://camera-user:camera-password@b.example/photo.jpg?key=photo-credential"}}
			if test.sourceType != "" {
				photo.Source.Type = test.sourceType
			}
			if test.sourceURL == "empty" {
				photo.Source.URL = ""
			} else if test.sourceURL != "" {
				photo.Source.URL = test.sourceURL
			}
			if test.expired {
				past := time.Now().Add(-time.Hour)
				photo.Source.ExpiresAt = &past
			}
			fetches := 0
			fetchPublicPluginMedia = func(_ context.Context, raw string, _ int64) (*util.SafeFetchResult, error) {
				fetches++
				if test.status > 0 {
					return nil, util.HTTPStatusError{StatusCode: test.status, Message: "provider response from " + raw}
				}
				if test.failure != nil {
					return nil, test.failure(raw)
				}
				return nil, errors.New("unexpected fetch")
			}
			_, omitted, err := preparePhotoAssets(context.Background(), app, []pluginsystem.Photo{photo}, Options{}, &pluginMediaBudget{}, photoImportContext{Provider: "strava", TrailExternalID: "activity-1", TrailName: "Morning walk", WaypointName: "Summit"})
			if err != nil {
				t.Fatal(err)
			}
			logged := output.String()
			rows := photoOmissionLogs(t, &output)
			if len(omitted) != 1 || len(rows) != 1 {
				t.Fatalf("omission/log count=%d/%d", len(omitted), len(rows))
			}
			row := rows[0]
			if row["cause"] != test.cause || row["reason"] != test.reason || row["retryable"] != test.retryable {
				t.Fatalf("omission cause/reason/retryable=%v/%v/%v", row["cause"], row["reason"], row["retryable"])
			}
			if test.status > 0 {
				if row["status"] != float64(test.status) {
					t.Fatalf("omission status=%v want=%d", row["status"], test.status)
				}
			} else if _, ok := row["status"]; ok {
				t.Fatal("non-HTTP omission has a status")
			}
			if row["filename"] != "camera.jpg" || row["source_type"] != photo.Source.Type || row["provider"] != "strava" || row["trail_external_id"] != "activity-1" || row["trail_name"] != "Morning walk" || row["waypoint_name"] != "Summit" {
				t.Fatal("omission lost filename/source/import context")
			}
			for _, secret := range []string{"photo-credential", "camera-password", "camera-user", "https://", "ftp://"} {
				if strings.Contains(logged, secret) {
					t.Fatal("omission log contains URL credentials or a raw URL")
				}
			}
			if test.reason != "download_failed" && fetches != 0 {
				t.Fatalf("invalid/expired source made%d requests", fetches)
			}
			if test.reason == "download_failed" && row["cause"] == "" {
				t.Fatal("download failure has no diagnostic cause")
			}
		})
	}
}
