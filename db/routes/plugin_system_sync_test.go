package routes

import "testing"

func TestCategoryMappingPreservesExplicitEmptyMap(t *testing.T) {
	mapping := categoryMapping(map[string]any{
		"categoryMapping": map[string]any{},
	})
	if mapping == nil {
		t.Fatal("expected explicit empty category mapping to be preserved")
	}
	if len(mapping) != 0 {
		t.Fatalf("expected empty category mapping, got %#v", mapping)
	}
}

func TestCategoryMappingNilWhenMissing(t *testing.T) {
	if mapping := categoryMapping(map[string]any{}); mapping != nil {
		t.Fatalf("expected missing category mapping to be nil, got %#v", mapping)
	}
}

func TestCategoryMappingPreservesBlankProviderMapping(t *testing.T) {
	mapping := categoryMapping(map[string]any{
		"categoryMapping": map[string]any{
			"Ride": "",
		},
	})
	if mapping == nil {
		t.Fatal("expected category mapping")
	}
	if value, ok := mapping["Ride"]; !ok || value != "" {
		t.Fatalf("expected blank provider mapping to be preserved, got %#v", mapping)
	}
}
