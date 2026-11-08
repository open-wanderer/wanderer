package hooks

import (
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

// Migration source fields identify previously published URLs. Only internal
// writes may establish or change them; metadata sent through the records API
// must retain the original trusted mapping and cannot create a new one.
func PreserveAssetSourceMetadata() func(*core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		metadata, err := assetMetadataMap(e.Record)
		if err != nil {
			return err
		}
		original := util.AssetProvenanceMetadata(e.Record.Original())
		copyID, _ := metadata["copied_from"].(string)
		filename, _ := metadata["original_filename"].(string)
		for key := range metadata {
			if util.IsAssetProvenanceKey(key) {
				delete(metadata, key)
			}
		}
		// The records API cannot write the hidden field directly, including
		// through superuser requests. Only the model hook may populate it.
		e.Record.Set(util.AssetProvenanceField, e.Record.Original().Get(util.AssetProvenanceField))
		for key, value := range original {
			metadata[key] = value
		}
		if e.Record.Original().Id == "" && copyID != "" {
			source, err := e.App.FindRecordById("assets", copyID)
			if err != nil {
				return e.NotFoundError("Photo source not found", err)
			}
			info, err := e.RequestInfo()
			if err != nil {
				return err
			}
			allowed, err := e.App.CanAccessRecord(source, info, source.Collection().ViewRule)
			if err != nil || !allowed || source.GetString("type") != "photo" {
				return e.NotFoundError("Photo source not found", err)
			}
			metadata["copied_from"] = source.Id
			if filename != "" {
				metadata["original_filename"] = filename
			}
		}
		e.Record.Set("metadata", metadata)
		return e.Next()
	}
}

// StoreAssetSourceMetadata handles trusted migration/import/merge writes.
// Public JSON contains only metadata that may also be queried by a viewer.
func StoreAssetSourceMetadata() func(*core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		metadata, err := assetMetadataMap(e.Record)
		if err != nil {
			metadata = map[string]any{}
		}
		provenance := util.AssetProvenanceMetadata(e.Record)
		for key := range metadata {
			if util.IsAssetProvenanceKey(key) {
				delete(metadata, key)
			}
		}
		e.Record.Set("metadata", metadata)
		e.Record.Set(util.AssetProvenanceField, provenance)
		return e.Next()
	}
}
