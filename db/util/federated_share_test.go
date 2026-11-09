package util

import (
	"context"
	"errors"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"net/url"
	"strings"
	"testing"
)

func TestFederatedShareCredentialsStayAtTheirOriginAndOutOfIdentities(t *testing.T) {
	token := "abcdefghijklmnopqrstuvwxyz012345"
	stable := "https://a.example/photo?z=1&a=2"
	if got := WithoutFederatedShareToken(stable); got != stable {
		t.Fatalf("unrelated identity query changed: %s", got)
	}
	ctx := WithFederatedShareToken(context.Background(), "https://a.example/api/v1/trail/one", token)
	for _, raw := range []string{"https://a.example/api/v1/files/assets/photo/camera.png", "https://b.example/api/v1/files/assets/photo/camera.png"} {
		parsed, _ := url.Parse(FederatedShareURL(ctx, raw))
		want := ""
		if parsed.Host == "a.example" {
			want = token
		}
		if parsed.Query().Get("share") != want {
			t.Fatalf("wrong credential origin: %s", parsed.Host)
		}
	}
	for _, identity := range []string{"", "https://a.example/camera.png?share=" + token} {
		photo, ok := FederatedPhotoFromAttachment(&pub.Object{ID: pub.IRI(identity), Type: pub.ImageType, URL: pub.IRI("https://a.example/camera.png?share=" + token)})
		if !ok || photo.CanonicalID != "https://a.example/camera.png" || !strings.Contains(photo.FileURL, token) || photo.SourceName != "camera.png" {
			t.Fatalf("wrong attachment identity: %+v", photo)
		}
	}

	cause := errors.New("network unavailable")
	err := RedactFederatedShareError(fmt.Errorf("fetch https://a.example/file?share=%s: %w", token, cause))
	if strings.Contains(err.Error(), token) || !errors.Is(err, cause) {
		t.Fatalf("credential in error or lost cause: %v", err)
	}
}
