package util

import (
	"fmt"
	"github.com/pocketbase/pocketbase/core"
)

func AssetFileRedirectURL(record *core.Record) string {
	if record == nil {
		return ""
	}
	file := record.GetString("file")
	if file == "" {
		return ""
	}
	collectionID := assetCollectionID(record)
	if collectionID == "" {
		return ""
	}
	return fmt.Sprintf("/api/files/%s/%s/%s", collectionID, record.Id, file)
}

func assetCollectionID(record *core.Record) string {
	if record == nil || record.Collection() == nil {
		return ""
	}
	return record.Collection().Id
}
