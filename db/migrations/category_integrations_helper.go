package migrations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

type categoryIntegration struct {
	IntegrationType    string   `json:"integrationType"`
	ProviderCategories []string `json:"providerCategories"`
}

func applyCategoryIntegrationMappings(app core.App, integrationType string, mappings map[string][]string, apply bool) error {
	for categoryName, providerCategories := range mappings {
		record, err := app.FindFirstRecordByData("categories", "name", categoryName)
		if err != nil {
			return fmt.Errorf("unable to find category '%s': %w", categoryName, err)
		}

		integrations, err := parseRecordIntegrations(record)
		if err != nil {
			return fmt.Errorf("unable to parse integrations for '%s': %w", categoryName, err)
		}

		if apply {
			found := false
			for idx := range integrations {
				if strings.EqualFold(integrations[idx].IntegrationType, integrationType) {
					integrations[idx].ProviderCategories = providerCategories
					found = true
					break
				}
			}
			if !found {
				integrations = append(integrations, categoryIntegration{
					IntegrationType:    integrationType,
					ProviderCategories: providerCategories,
				})
			}
		} else {
			filtered := integrations[:0]
			for _, entry := range integrations {
				if strings.EqualFold(entry.IntegrationType, integrationType) {
					continue
				}
				filtered = append(filtered, entry)
			}
			integrations = filtered
		}

		if len(integrations) == 0 {
			record.Set("integrations", nil)
		} else {
			record.Set("integrations", integrations)
		}

		if err := app.Save(record); err != nil {
			return fmt.Errorf("unable to save category '%s': %w", categoryName, err)
		}
	}

	return nil
}

func parseRecordIntegrations(record *core.Record) ([]categoryIntegration, error) {
	raw := record.Get("integrations")
	if raw == nil {
		return []categoryIntegration{}, nil
	}

	var data []byte
	switch v := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || trimmed == "null" {
			return []categoryIntegration{}, nil
		}
		data = []byte(trimmed)
	case []byte:
		if len(v) == 0 {
			return []categoryIntegration{}, nil
		}
		data = v
	default:
		marshaled, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if len(marshaled) == 0 || string(marshaled) == "null" {
			return []categoryIntegration{}, nil
		}
		data = marshaled
	}

	var integrations []categoryIntegration
	if err := json.Unmarshal(data, &integrations); err != nil {
		return nil, err
	}

	return integrations, nil
}

func buildCategoryMappingsFromMap(source map[string]string) map[string][]string {
	result := map[string][]string{}
	for provider, category := range source {
		result[category] = append(result[category], provider)
	}

	for _, providers := range result {
		sort.Strings(providers)
	}

	return result
}
