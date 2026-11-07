package commands

import "testing"

// hierarchyFixture mirrors the Germany subtree from a real CoMaps
// hierarchy.txt excerpt, plus the World/WorldCoasts header lines.
const hierarchyFixture = `World
WorldCoasts
Germany;Q183;de;de
 Germany_Baden-Wurttemberg;Q985
  Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg;Q2833
  Germany_Baden-Wurttemberg_Regierungsbezirk Karlsruhe;Q8165
 Germany_Berlin;Q64
 Germany_Free State of Bavaria;Q980
`

func findNode(t *testing.T, nodes []HierarchyNode, comapsID string) HierarchyNode {
	t.Helper()
	for _, n := range nodes {
		if n.ComapsID == comapsID {
			return n
		}
	}
	t.Fatalf("node %q not found", comapsID)
	return HierarchyNode{}
}

func TestParseHierarchy(t *testing.T) {
	t.Run("World and WorldCoasts headers are skipped", func(t *testing.T) {
		nodes, err := ParseHierarchy([]byte(hierarchyFixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, n := range nodes {
			if n.ComapsID == "World" || n.ComapsID == "WorldCoasts" {
				t.Fatalf("found header line as a node: %+v", n)
			}
		}
	})

	t.Run("Germany root node has expected depth/parent/name/path/kind", func(t *testing.T) {
		nodes, err := ParseHierarchy([]byte(hierarchyFixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		germany := findNode(t, nodes, "Germany")

		if germany.Depth != 0 {
			t.Errorf("got Depth %d, want 0", germany.Depth)
		}
		if germany.ParentComapsID != "" {
			t.Errorf("got ParentComapsID %q, want empty", germany.ParentComapsID)
		}
		if germany.Name != "Germany" {
			t.Errorf("got Name %q, want Germany", germany.Name)
		}
		if germany.Path != "germany" {
			t.Errorf("got Path %q, want germany", germany.Path)
		}
		if germany.Kind != "group" {
			t.Errorf("got Kind %q, want group", germany.Kind)
		}
	})

	t.Run("leaf-like Free State of Bavaria has expected name/parent/path", func(t *testing.T) {
		nodes, err := ParseHierarchy([]byte(hierarchyFixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bavaria := findNode(t, nodes, "Germany_Free State of Bavaria")

		if bavaria.Name != "Free State of Bavaria" {
			t.Errorf("got Name %q, want Free State of Bavaria", bavaria.Name)
		}
		if bavaria.ParentComapsID != "Germany" {
			t.Errorf("got ParentComapsID %q, want Germany", bavaria.ParentComapsID)
		}
		if bavaria.Path != "germany.germany_free_state_of_bavaria" {
			t.Errorf("got Path %q, want germany.germany_free_state_of_bavaria", bavaria.Path)
		}
		// Last node in the fixture -> always leaf, regardless of lookahead.
		if bavaria.Kind != "leaf" {
			t.Errorf("got Kind %q, want leaf", bavaria.Kind)
		}
	})

	t.Run("depth-2 leaves have correct depth/parent/path/kind", func(t *testing.T) {
		nodes, err := ParseHierarchy([]byte(hierarchyFixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		freiburg := findNode(t, nodes, "Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg")

		if freiburg.Depth != 2 {
			t.Errorf("got Depth %d, want 2", freiburg.Depth)
		}
		if freiburg.ParentComapsID != "Germany_Baden-Wurttemberg" {
			t.Errorf("got ParentComapsID %q, want Germany_Baden-Wurttemberg", freiburg.ParentComapsID)
		}
		if freiburg.Name != "Regierungsbezirk Freiburg" {
			t.Errorf("got Name %q, want Regierungsbezirk Freiburg", freiburg.Name)
		}
		if freiburg.Path != "germany.germany_baden-wurttemberg.germany_baden-wurttemberg_regierungsbezirk_freiburg" {
			t.Errorf("got Path %q", freiburg.Path)
		}
		if freiburg.Kind != "leaf" {
			t.Errorf("got Kind %q, want leaf", freiburg.Kind)
		}
	})

	t.Run("Baden-Wurttemberg is a group (has depth-2 children)", func(t *testing.T) {
		nodes, err := ParseHierarchy([]byte(hierarchyFixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bw := findNode(t, nodes, "Germany_Baden-Wurttemberg")
		if bw.Kind != "group" {
			t.Errorf("got Kind %q, want group", bw.Kind)
		}
		if bw.Depth != 1 {
			t.Errorf("got Depth %d, want 1", bw.Depth)
		}
	})

	t.Run("sibling sort_order values are 0,1,2 in file order under a shared parent", func(t *testing.T) {
		nodes, err := ParseHierarchy([]byte(hierarchyFixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bw := findNode(t, nodes, "Germany_Baden-Wurttemberg")
		berlin := findNode(t, nodes, "Germany_Berlin")
		bavaria := findNode(t, nodes, "Germany_Free State of Bavaria")

		if bw.SortOrder != 0 {
			t.Errorf("got Baden-Wurttemberg SortOrder %d, want 0", bw.SortOrder)
		}
		if berlin.SortOrder != 1 {
			t.Errorf("got Berlin SortOrder %d, want 1", berlin.SortOrder)
		}
		if bavaria.SortOrder != 2 {
			t.Errorf("got Bavaria SortOrder %d, want 2", bavaria.SortOrder)
		}

		freiburg := findNode(t, nodes, "Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg")
		karlsruhe := findNode(t, nodes, "Germany_Baden-Wurttemberg_Regierungsbezirk Karlsruhe")
		if freiburg.SortOrder != 0 {
			t.Errorf("got Freiburg SortOrder %d, want 0", freiburg.SortOrder)
		}
		if karlsruhe.SortOrder != 1 {
			t.Errorf("got Karlsruhe SortOrder %d, want 1", karlsruhe.SortOrder)
		}
	})

	t.Run("a depth jump of more than +1 returns an error", func(t *testing.T) {
		malformed := `Germany;Q183
  Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg;Q2833
`
		_, err := ParseHierarchy([]byte(malformed))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
