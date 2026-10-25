package routes

import "github.com/pocketbase/pocketbase/core"

// Remote documents supply field values. Record identities, local sync state,
// downloaded files and relation setters are controlled by the sync itself.
func loadRemoteRecordFields(record *core.Record, data map[string]any, protected ...string) {
	fields := make(map[string]any, len(data))
	for key, value := range data {
		switch key {
		case "id", "iri", "author", "trail", "needs_full_sync", "full_sync_completed":
			continue
		}
		field := record.Collection().Fields.GetByName(key)
		if field == nil || field.GetName() != key {
			continue
		}
		if _, isFile := field.(*core.FileField); isFile {
			continue
		}
		excluded := false
		for _, name := range protected {
			if key == name {
				excluded = true
				break
			}
		}
		if !excluded {
			fields[key] = value
		}
	}
	record.Load(fields)
}
