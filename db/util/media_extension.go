package util

import (
	"mime"
	"path/filepath"
	"strings"
)

// MediaFileName keeps descriptive dots, but ensures that a known media
// extension describes the actual response rather than its provider's source.
func MediaFileName(name, contentType string) string {
	want := MediaFileExtension(contentType)
	got := strings.ToLower(filepath.Ext(name))
	if want == "" {
		return strings.TrimSuffix(name, ".")
	}
	if got == want {
		return name
	}
	if isMediaFileExtension(got) {
		return strings.TrimSuffix(name, filepath.Ext(name)) + want
	}
	return strings.TrimSuffix(name, ".") + want
}

func isMediaFileExtension(extension string) bool {
	switch extension {
	case ".jpg", ".jpeg", ".jpe", ".jfif", ".png", ".webp", ".gif", ".avif", ".bmp", ".ico", ".svg",
		".heic", ".heif", ".tif", ".tiff", ".dng", ".nef", ".cr2", ".cr3", ".arw", ".raf", ".rw2", ".orf", ".pef", ".srw",
		".mp4", ".f4v", ".webm", ".ogv", ".mp3", ".ogg", ".gpx", ".xml", ".json", ".txt":
		return true
	}
	return false
}

// MediaFileExtension is independent of the host's optional MIME database.
func MediaFileExtension(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return ""
	}
	switch strings.ToLower(mediaType) {
	case "image/jpeg":
		return ".jpg"
	case "image/png", "image/vnd.mozilla.apng":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "image/avif":
		return ".avif"
	case "image/svg+xml":
		return ".svg"
	case "image/heic", "image/heif":
		return ".heic"
	case "image/tiff":
		return ".tif"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	case "video/ogg":
		return ".ogv"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "application/gpx+xml":
		return ".gpx"
	case "application/xml", "text/xml":
		return ".xml"
	case "application/json":
		return ".json"
	case "text/plain":
		return ".txt"
	default:
		return ""
	}
}
