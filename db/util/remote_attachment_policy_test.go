package util

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/doyensec/safeurl"
)

func TestRemoteFileFetchersApplyIPPolicyOnAllowedPorts(t *testing.T) {
	t.Cleanup(SetRemoteAttachmentClientForTesting(nil))
	for _, rawURL := range []string{
		"http://127.0.0.1:80/attachment-policy-test",
		"https://127.0.0.1:443/attachment-policy-test",
		"http://[::1]:80/attachment-policy-test",
		"https://[::1]:443/attachment-policy-test",
	} {
		for _, fetcher := range []string{"attachment", "public"} {
			t.Run(fetcher+"/"+rawURL, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				var err error
				if fetcher == "attachment" {
					_, cleanup, fetchErr := DownloadRemoteFile(ctx, rawURL, 16)
					defer cleanup()
					err = fetchErr
				} else {
					_, err = FetchPublicURL(ctx, rawURL, 16)
				}
				var policyError *safeurl.AllowedIPError
				if !errors.As(err, &policyError) {
					t.Fatalf("expected an IP policy refusal on an allowed port, got %v", err)
				}
			})
		}
	}
}
