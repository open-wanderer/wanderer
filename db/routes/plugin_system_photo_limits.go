package routes

import (
	"pocketbase/plugins/importer"
	"pocketbase/services/pluginhost"
)

func pluginPhotoImportLimits(hostConfig map[string]any) importer.PhotoImportLimits {
	return pluginhost.PhotoImportLimits(hostConfig)
}
