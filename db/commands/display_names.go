package commands

import (
	"encoding/json"
	"fmt"
	"strings"
)

// displayNamesPath is CoMaps' English country/region name table, relative to
// the repository's data/ directory. It is keyed by the same ids as
// hierarchy.txt and holds the names the CoMaps app itself shows, for example
// "Austria_Styria_Graz" -> "Styria — Graz" or "Russia_Altai Krai" ->
// "Altai Krai".
const displayNamesPath = "translations/countries-strings/en.json/localize.json"

// ParseDisplayNames decodes CoMaps' localize.json into id -> display name.
func ParseDisplayNames(data []byte) (map[string]string, error) {
	var entries map[string]struct {
		DefaultMessage string `json:"defaultMessage"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("display_names: %w", err)
	}

	names := make(map[string]string, len(entries))
	for id, entry := range entries {
		if name := strings.TrimSpace(entry.DefaultMessage); name != "" {
			names[id] = name
		}
	}
	return names, nil
}

// ApplyDisplayNames replaces each node's derived name with its CoMaps display
// name. Nodes without one keep the name ParseHierarchy derived from the id. It
// returns how many nodes had no display name.
func ApplyDisplayNames(nodes []HierarchyNode, names map[string]string) int {
	missing := 0
	for i := range nodes {
		if name, ok := names[nodes[i].ComapsID]; ok {
			nodes[i].Name = name
		} else {
			missing++
		}
	}
	return missing
}
