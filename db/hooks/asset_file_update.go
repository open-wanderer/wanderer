package hooks

import "github.com/pocketbase/pocketbase/core"

// Request modifiers have already been resolved into the record here. Internal
// model writes can still replace files, for example while materializing media.
func PreventAssetFileReplacement() func(*core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if e.Auth != nil && e.Auth.IsSuperuser() {
			return e.Next()
		}
		info, err := e.RequestInfo()
		if err != nil {
			return err
		}
		for _, field := range []string{"file", "file+", "+file", "file-"} {
			if _, present := info.Body[field]; present {
				return e.BadRequestError("Asset files cannot be replaced", nil)
			}
		}
		if len(e.Record.GetUnsavedFiles("file")) > 0 || e.Record.GetString("file") != e.Record.Original().GetString("file") {
			return e.BadRequestError("Asset files cannot be replaced", nil)
		}
		return e.Next()
	}
}
