package srch0

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
)

var aliases = struct {
	sync.RWMutex
	labels map[string]string
}{labels: map[string]string{}}

// ID maps readable corpus labels to valid PocketBase IDs without modifying the
// shared dataset. This bijection affects identity only, never schema validation.
func ID(label string) string {
	if label == "" {
		return ""
	}
	aliases.Lock()
	defer aliases.Unlock()
	if _, ok := aliases.labels[label]; ok {
		return label
	}
	id := fmt.Sprintf("%x", sha256.Sum256([]byte("wanderer.srch0/"+label)))[:15]
	if old, ok := aliases.labels[id]; ok && old != label {
		panic("SRCH0 ID collision")
	}
	aliases.labels[id] = label
	return id
}

// AssetAlias gives an uploaded fixture file its stable corpus label. PocketBase
// generated filenames are intentionally not part of the observed contract.
func AssetAlias(filename, label string) {
	aliases.Lock()
	defer aliases.Unlock()
	aliases.labels[filename] = label
}

// Readable restores labels in captured documents, relation arrays, URL paths
// and tenant filters. Actual SQL records and HTTP payloads retain valid IDs.
func Readable(value any) any {
	aliases.RLock()
	defer aliases.RUnlock()
	return readable(value)
}

func readable(value any) any {
	switch v := value.(type) {
	case string:
		for id, label := range aliases.labels {
			v = strings.ReplaceAll(v, id, label)
		}
		return v
	case map[string]any:
		out := Object{}
		for k, x := range v {
			out[readable(k).(string)] = readable(x)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = readable(x)
		}
		return out
	default:
		return value
	}
}
