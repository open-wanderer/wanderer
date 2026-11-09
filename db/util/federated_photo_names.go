package util

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/pocketbase/pocketbase/core"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
)

// FederatedPhotoMatchesAsset recognizes copies retained across an origin upgrade.
// Only the same author's photos can be adopted; unrelated legacy photos remain.
func FederatedPhotoMatchesAsset(asset *core.Record, author string, photo FederatedPhoto) bool {
	if asset.GetString("author") != author || IsGeneratedRoutePreviewAsset(asset) || photo.SourceName == "" {
		return false
	}
	provider := asset.GetString("external_provider")
	if provider == "" {
		return RemotePhotoNameMatches([]string{asset.GetString("file")}, photo.SourceName)
	}
	if provider != FederationAssetProvider {
		return false
	}
	legacy, err := url.Parse(asset.GetString("external_id"))
	modern, modernErr := url.Parse(photo.CanonicalID)
	if err != nil || modernErr != nil || legacy.Scheme != modern.Scheme || !strings.EqualFold(legacy.Host, modern.Host) {
		return false
	}
	if !strings.Contains(legacy.Path, "/api/v1/files/") && !strings.Contains(legacy.Path, "/api/files/") {
		return false
	}
	return path.Base(legacy.Path) == photo.SourceName
}

// FederatedSourceFileName names staged copies without truncating source distinctions.
func FederatedSourceFileName(name string) string {
	sum := sha256.Sum256([]byte(name))
	ext := path.Ext(name)
	if len(ext) > 20 || strings.IndexFunc(ext, func(r rune) bool {
		return !(r == '.' || r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) >= 0 {
		ext = ""
	}
	return "remote_" + hex.EncodeToString(sum[:]) + ext
}

// RemotePhotoNameMatches includes names produced by earlier PocketBase copies.
var remoteCopyNameSuffix = regexp.MustCompile(`^[a-z0-9]{10}$`)
var remoteCopyExtInvalid = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
var remoteCopyDetectedExt = regexp.MustCompile(`^\.[a-z0-9]{1,5}$`)

func RemotePhotoNameMatches(saved []string, source string) bool {
	stable := FederatedSourceFileName(source)
	// PocketBase keeps the last two extension components when adding its suffix.
	ext := ""
	if dot := strings.LastIndex(source, "."); dot >= 0 {
		if previous := strings.LastIndex(source[:dot], "."); previous >= 0 {
			dot = previous
		}
		ext = source[dot:]
	}
	base := strings.TrimSuffix(source, ext)
	if len(base) > 100 {
		base = base[:100]
	}
	exts := []string{ext}
	// attachmentFileName normalizes punctuation before PocketBase trims dots.
	normalized := remoteCopyExtInvalid.ReplaceAllString(ext, "_")
	for _, candidate := range []string{normalized, "." + strings.Trim(ext, "."), "." + strings.Trim(normalized, ".")} {
		if candidate != "." && !slices.Contains(exts, candidate) {
			exts = append(exts, candidate)
		}
	}
	for _, name := range saved {
		if name == stable || name == source {
			return true
		}
		// Earlier copies used PocketBase's extra ten-character suffix.
		for _, ext := range exts {
			if strings.HasPrefix(name, base+"_") && strings.HasSuffix(name, ext) && len(name) == len(base)+11+len(ext) && remoteCopyNameSuffix.MatchString(name[len(base)+1:len(base)+11]) {
				return true
			}
		}
		// An extension of only dots is trimmed before PocketBase detects one
		// from the contents. Its base and generated suffix still identify it.
		if ext != "" && strings.Trim(ext, ".") == "" && strings.HasPrefix(name, base+"_") && len(name) > len(base)+11 && remoteCopyNameSuffix.MatchString(name[len(base)+1:len(base)+11]) && remoteCopyDetectedExt.MatchString(name[len(base)+11:]) {
			return true
		}
	}
	return false
}
