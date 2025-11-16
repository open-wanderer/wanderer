package migrations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type categoryIntegration struct {
	IntegrationType    string   `json:"integrationType"`
	ProviderCategories []string `json:"providerCategories"`
}

func init() {
	m.Register(func(app core.App) error {
		return updateCategoryIntegrations(app, true)
	}, func(app core.App) error {
		return updateCategoryIntegrations(app, false)
	})
}

func updateCategoryIntegrations(app core.App, apply bool) error {
	komootMappings := map[string][]string{
		"Hiking": {
			"hike",
			"mountaineering",
		},
		"Biking": {
			"racebike",
			"e_racebike",
			"touringbicycle",
			"e_touringbicycle",
			"mtb",
			"e_mtb",
			"mtb_easy",
			"e_mtb_easy",
			"mtb_advanced",
			"e_mtb_advanced",
			"downhillbike",
			"unicycle",
			"citybike",
		},
		"Walking": {
			"jogging",
			"nordicwalking",
			"skaten",
			"other",
		},
		"Climbing": {
			"climbing",
		},
		"Skiing": {
			"nordic",
			"skialpin",
			"skitour",
			"sled",
			"snowboard",
			"snowshoe",
		},
	}

	for categoryName, providerCategories := range komootMappings {
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
				if strings.EqualFold(integrations[idx].IntegrationType, "komoot") {
					integrations[idx].ProviderCategories = providerCategories
					found = true
					break
				}
			}
			if !found {
				integrations = append(integrations, categoryIntegration{
					IntegrationType:    "komoot",
					ProviderCategories: providerCategories,
				})
			}
		} else {
			filtered := integrations[:0]
			for _, entry := range integrations {
				if strings.EqualFold(entry.IntegrationType, "komoot") {
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
		if strings.TrimSpace(v) == "" || strings.TrimSpace(v) == "null" {
			return []categoryIntegration{}, nil
		}
		data = []byte(v)
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
		data = marshaled
	}

	var integrations []categoryIntegration
	if err := json.Unmarshal(data, &integrations); err != nil {
		return nil, err
	}
	return integrations, nil
}
