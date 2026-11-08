package hooks

import (
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

// Migration provenance may identify private records whose photo was later
// copied or shared. Only the asset's author and administrators may export it.
func HideAssetSourceMetadata() func(*core.RecordEnrichEvent) error {
	return func(e *core.RecordEnrichEvent) error {
		if e.RequestInfo != nil && e.RequestInfo.Auth != nil && e.RequestInfo.Auth.Id != "" {
			if e.RequestInfo.Auth.IsSuperuser() {
				e.Record.Set("metadata", util.AssetMetadataWithProvenance(e.Record))
				return e.Next()
			}
			if assetAuthorOwnedForRequest(e.App, e.RequestInfo, e.Record.GetString("author")) {
				e.Record.Set("metadata", util.AssetMetadataWithProvenance(e.Record))
				return e.Next()
			}
		}
		metadata, err := assetMetadataMap(e.Record)
		if err != nil {
			// Malformed optional metadata must not hide an entire requested
			// relation during PocketBase enrichment. Fail closed on metadata
			// while preserving the record and the remaining child expands.
			metadata = map[string]any{}
		}
		for key := range metadata {
			if util.IsAssetProvenanceKey(key) {
				delete(metadata, key)
			}
		}
		e.Record.Set("metadata", metadata)
		return e.Next()
	}
}
