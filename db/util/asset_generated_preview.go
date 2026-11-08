package util

import (
	"github.com/pocketbase/pocketbase/core"
	"regexp"
	"strings"
)

func AssetFilename(record *core.Record, metadata map[string]any) string {
	if record == nil {
		return ""
	}
	return assetFilename(metadata, record.GetString("file"), record.GetString("external_id"), record.Id)
}

func assetFilename(metadata map[string]any, file string, externalID string, assetID string) string {
	if remoteFilename := AssetMetadataString(metadata, "remote", "filename"); remoteFilename != "" {
		return remoteFilename
	}
	if sourceFile := AssetMetadataString(metadata, "source_file"); sourceFile != "" {
		return sourceFile
	}
	if file != "" {
		return file
	}
	if externalID != "" {
		return externalID
	}
	return assetID
}

func IsGeneratedRoutePreviewAsset(record *core.Record) bool {
	if record == nil {
		return false
	}
	metadata := map[string]any{}
	_ = record.UnmarshalJSONField("metadata", &metadata)
	return isGeneratedRoutePreview(metadata, AssetFilename(record, metadata), record.GetString("external_provider"), !record.GetDateTime("taken_at").IsZero())
}

var generatedRoutePreviewFilename = regexp.MustCompile(`^(?:wanderer-route-preview\.(?:webp|png)|wanderer_route_preview_[a-z0-9]{10}(?:_[a-z0-9]{10})*\.(?:webp|png))$`)

// isGeneratedRoutePreview recognises the route preview the trail editor
// generates for a trail without photos. Generic route_* upload names are
// ordinary photos; legacy previews are classified once by the migration.
func isGeneratedRoutePreview(metadata map[string]any, filename string, externalProvider string, hasTakenAt bool) bool {
	if AssetMetadataString(metadata, "generated", "kind") == "route-preview" {
		return true
	}
	filename = strings.ToLower(filename)
	hasLegacyGeneratedName := generatedRoutePreviewFilename.MatchString(filename)
	return hasLegacyGeneratedName && externalProvider == "" && !hasTakenAt
}
