package util

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

const AssetProvenanceField = "source_metadata"

func IsAssetProvenanceKey(key string) bool {
	return strings.HasPrefix(key, "source_") || key == "copied_from" || key == "original_filename" || key == "merged_assets"
}

// AssetProvenanceMetadata is an internal mapping. The schema hides it from
// both responses and user-supplied filters, including relation traversals.
func AssetProvenanceMetadata(record *core.Record) map[string]any {
	result := map[string]any{}
	_ = record.UnmarshalJSONField(AssetProvenanceField, &result)
	if result == nil {
		result = map[string]any{}
	}
	// Internal migration/import callers can still use the legacy input form.
	legacy := map[string]any{}
	if record.UnmarshalJSONField("metadata", &legacy) == nil {
		for key, value := range legacy {
			if IsAssetProvenanceKey(key) {
				result[key] = value
			}
		}
	}
	return result
}

func AssetMetadataWithProvenance(record *core.Record) map[string]any {
	result := map[string]any{}
	_ = record.UnmarshalJSONField("metadata", &result)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range AssetProvenanceMetadata(record) {
		result[key] = value
	}
	return result
}
