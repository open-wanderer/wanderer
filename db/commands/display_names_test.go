package commands

import "testing"

func TestApplyDisplayNames(t *testing.T) {
	names, err := ParseDisplayNames([]byte(`{
		"Austria": {"defaultMessage": "Austria"},
		"Austria_Styria_Graz": {"defaultMessage": "Styria — Graz"},
		"Russia_Altai Krai": {"defaultMessage": "Altai Krai"},
		"Empty": {"defaultMessage": "  "}
	}`))
	if err != nil {
		t.Fatal(err)
	}

	nodes := []HierarchyNode{
		{ComapsID: "Austria", Name: "Austria"},
		{ComapsID: "Austria_Styria_Graz", Name: "Styria_Graz"},
		{ComapsID: "Russia_Altai Krai", Name: "Russia_Altai Krai"},
		{ComapsID: "Empty", Name: "Empty"},
		{ComapsID: "Unknown_Region", Name: "Region"},
	}
	if missing := ApplyDisplayNames(nodes, names); missing != 2 {
		t.Errorf("missing = %d, want 2", missing)
	}

	want := []string{"Austria", "Styria — Graz", "Altai Krai", "Empty", "Region"}
	for i, node := range nodes {
		if node.Name != want[i] {
			t.Errorf("node %s name = %q, want %q", node.ComapsID, node.Name, want[i])
		}
	}
}

func TestParseDisplayNamesRejectsMalformedJSON(t *testing.T) {
	if _, err := ParseDisplayNames([]byte(`[`)); err == nil {
		t.Fatal("ParseDisplayNames accepted malformed JSON")
	}
}
