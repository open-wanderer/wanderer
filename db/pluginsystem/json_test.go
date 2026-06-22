package pluginsystem

import "testing"

func TestMergePluginConfigEmptyCategoryMappingOverridesDefaultMap(t *testing.T) {
	dst := map[string]any{
		"host": map[string]any{
			"categoryMapping": map[string]any{
				"hike": "hiking",
				"bike": "biking",
			},
			"privacy": "public",
		},
	}
	src := map[string]any{
		"host": map[string]any{
			"categoryMapping": map[string]any{},
		},
	}

	MergePluginConfig(dst, src)

	host := dst["host"].(map[string]any)
	mapping := host["categoryMapping"].(map[string]any)
	if len(mapping) != 0 {
		t.Fatalf("expected empty category mapping override, got %#v", mapping)
	}
	if host["privacy"] != "public" {
		t.Fatalf("expected sibling defaults to remain, got %#v", host)
	}
}

func TestMergePluginConfigCategoryMappingReplacesDefaultMap(t *testing.T) {
	dst := map[string]any{
		"host": map[string]any{
			"categoryMapping": map[string]any{
				"hike": "hiking",
				"bike": "biking",
			},
		},
	}
	src := map[string]any{
		"host": map[string]any{
			"categoryMapping": map[string]any{
				"hike": "custom",
			},
		},
	}

	MergePluginConfig(dst, src)

	mapping := dst["host"].(map[string]any)["categoryMapping"].(map[string]any)
	if len(mapping) != 1 || mapping["hike"] != "custom" {
		t.Fatalf("expected category mapping to replace defaults, got %#v", mapping)
	}
}

func TestDeepMergeConfigNonEmptyMapStillMergesByDefault(t *testing.T) {
	dst := map[string]any{
		"host": map[string]any{
			"categoryMapping": map[string]any{
				"hike": "hiking",
				"bike": "biking",
			},
		},
	}
	src := map[string]any{
		"host": map[string]any{
			"categoryMapping": map[string]any{
				"hike": "custom",
			},
		},
	}

	DeepMergeConfig(dst, src)

	mapping := dst["host"].(map[string]any)["categoryMapping"].(map[string]any)
	if mapping["hike"] != "custom" || mapping["bike"] != "biking" {
		t.Fatalf("expected generic merge to keep sibling defaults, got %#v", mapping)
	}
}
