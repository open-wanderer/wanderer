package util

import (
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"strings"
)

func AssetPublicMediaURL(record *core.Record, origin string) string {
	if fileURL := AssetPublicFileURL(record, origin); fileURL != "" {
		return fileURL
	}
	if record == nil {
		return ""
	}
	if record.GetString("storage_mode") == "" || record.GetString("storage_mode") == "copy" {
		return ""
	}
	return withOrigin(origin, fmt.Sprintf("/api/v1/assets/%s/file", record.Id))
}

func AssetPublicFileURL(record *core.Record, origin string) string {
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
	return withOrigin(origin, fmt.Sprintf("/api/v1/files/%s/%s/%s", collectionID, record.Id, file))
}

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

func withOrigin(origin string, path string) string {
	if origin == "" {
		return path
	}
	return strings.TrimRight(origin, "/") + path
}
