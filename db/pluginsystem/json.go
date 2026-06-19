package pluginsystem

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
)

// JSONMapFromRecord reads a PocketBase JSON field into a map. Invalid, empty,
// or null values are treated as an empty object because plugin config/state/auth
// fields should be tolerant of partially edited records.
func JSONMapFromRecord(record *core.Record, field string) map[string]any {
	if record == nil {
		return map[string]any{}
	}
	value := record.GetString(field)
	if value == "" {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(value), &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

// DeepMergeConfig recursively overlays src onto dst and clones JSON-like values
// so caller-owned config maps cannot be mutated through shared references.
func DeepMergeConfig(dst map[string]any, src map[string]any) {
	DeepMergeConfigWithReplaceKeys(dst, src, nil)
}

// DeepMergeConfigWithReplaceKeys behaves like DeepMergeConfig, but map values
// whose key is listed in replaceKeys replace the destination map instead of
// being recursively merged.
func DeepMergeConfigWithReplaceKeys(dst map[string]any, src map[string]any, replaceKeys map[string]bool) {
	for key, value := range src {
		srcMap, srcIsMap := value.(map[string]any)
		dstMap, dstIsMap := dst[key].(map[string]any)
		if srcIsMap && dstIsMap {
			if replaceKeys[key] {
				dst[key] = CloneJSONMap(srcMap)
				continue
			}
			DeepMergeConfigWithReplaceKeys(dstMap, srcMap, replaceKeys)
			continue
		}
		dst[key] = CloneJSONValue(value)
	}
}

func CloneJSONMap(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = CloneJSONValue(value)
	}
	return cloned
}

func CloneJSONValue(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var cloned any
	if err := json.Unmarshal(data, &cloned); err != nil {
		return value
	}
	return cloned
}
