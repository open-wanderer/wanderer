package util

import "testing"

func TestCanonicalMediaExtensions(t *testing.T) {
	for contentType, want := range map[string]string{
		"image/jpeg": ".jpg", "image/jpeg; charset=binary": ".jpg",
		"image/png": ".png", "image/webp": ".webp", "video/mp4": ".mp4",
		"application/gpx+xml": ".gpx", "application/unknown": "",
	} {
		if got := MediaFileExtension(contentType); got != want {
			t.Errorf("%s: %s, want %s", contentType, got, want)
		}
	}
}
