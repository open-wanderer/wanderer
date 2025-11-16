package shared

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type CategoryIntegrationEntry struct {
	IntegrationType    string   `json:"integrationType"`
	ProviderCategories []string `json:"providerCategories"`
}

func LoadIntegrationCategoryMappings(app core.App, integrationType string) (map[string]string, error) {
	categories := []*core.Record{}
	if err := app.RecordQuery("categories").All(&categories); err != nil {
		return nil, err
	}

	mappings := map[string]string{}

	for _, category := range categories {
		parsed, err := parseCategoryIntegrations(category)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("unable to parse category integrations for '%s': %v", category.GetString("name"), err))
			continue
		}

		for _, entry := range parsed {
			if !strings.EqualFold(entry.IntegrationType, integrationType) {
				continue
			}

			for _, providerCategory := range entry.ProviderCategories {
				key := strings.ToLower(strings.TrimSpace(providerCategory))
				if key == "" {
					continue
				}

				if existing, exists := mappings[key]; exists && existing != category.Id {
					app.Logger().Warn(fmt.Sprintf("provider category '%s' already mapped to category '%s', skipping duplicate in '%s'", providerCategory, existing, category.GetString("name")))
					continue
				}

				mappings[key] = category.Id
			}
		}
	}

	return mappings, nil
}

func parseCategoryIntegrations(category *core.Record) ([]CategoryIntegrationEntry, error) {
	raw := category.Get("integrations")
	if raw == nil {
		return nil, nil
	}

	var data []byte

	switch v := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || trimmed == "null" {
			return nil, nil
		}
		data = []byte(trimmed)
	case []byte:
		if len(v) == 0 {
			return nil, nil
		}
		data = v
	default:
		marshaled, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if len(marshaled) == 0 || string(marshaled) == "null" {
			return nil, nil
		}
		data = marshaled
	}

	var integrations []CategoryIntegrationEntry
	if err := json.Unmarshal(data, &integrations); err != nil {
		return nil, err
	}

	return integrations, nil
}
