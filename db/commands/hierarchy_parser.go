package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// HierarchyNode is one flattened entry from a parsed CoMaps hierarchy.txt
// indentation tree: a country, or a state/province/district nested beneath
// one. See ParseHierarchy.
type HierarchyNode struct {
	ComapsID       string
	ParentComapsID string
	Path           string
	Name           string
	Kind           string
	Depth          int
	SortOrder      int
}

// ParseHierarchy parses a CoMaps data/hierarchy.txt file (semicolon-
// delimited, one leading space of indentation per depth level) into a flat,
// parent-linked, path/depth/kind-annotated node list.
//
// Format (verified against the real CoMaps file):
//
//	World
//	WorldCoasts
//	Germany;Q183;de;de
//	 Germany_Baden-Wurttemberg;Q985
//	  Germany_Baden-Wurttemberg_Regierungsbezirk Freiburg;Q2833
//
// The "World"/"WorldCoasts" header lines (zero ';' characters) are skipped
// and never appear as nodes. Depth is the exact count of
// leading space characters. comaps_id is the first ';'-delimited field with
// only the leading indentation trimmed — internal spaces are preserved
// verbatim. A node's parent is the most recently seen node at
// depth-1 (empty ParentComapsID at depth 0). Name is comaps_id with the
// exact "ParentComapsID_" prefix stripped (full comaps_id at depth 0). Path
// is the materialized dotted path of slug(comaps_id) segments (slug =
// lowercase + spaces -> underscores). Kind is "group" iff the immediately
// following node is at depth+1, otherwise "leaf" (the last node is always
// leaf). SortOrder is the zero-based index among siblings sharing the same
// parent, in file order.
//
// This does not fetch or parse countries.txt — hierarchy.txt's indentation
// alone determines group/leaf. ParseHierarchy returns a descriptive error,
// and never aborts the
// process, if a line's depth jumps by more than +1 relative to the
// previous node — an indentation shape the parent-tracking below cannot
// resolve.
func ParseHierarchy(data []byte) ([]HierarchyNode, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var nodes []HierarchyNode
	stack := []string{}     // stack[d] = most recent comaps_id seen at depth d
	pathStack := []string{} // pathStack[d] = that node's materialized path
	siblingCounter := map[string]int{}
	prevDepth := -1
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		if strings.Count(line, ";") == 0 {
			continue // World/WorldCoasts header lines
		}

		depth := countLeadingSpaces(line)
		rest := line[depth:]
		semi := strings.IndexByte(rest, ';')
		if semi < 0 {
			continue // defensive: unreachable given the Count check above
		}
		comapsID := rest[:semi]
		if comapsID == "" {
			continue // defensive: skip a blank id
		}

		if depth > prevDepth+1 {
			return nil, fmt.Errorf("hierarchy_parser: line %d: depth %d is not reachable from previous depth %d (id %q)", lineNo, depth, prevDepth, comapsID)
		}

		var parentID, parentPath string
		if depth > 0 {
			parentID = stack[depth-1]
			parentPath = pathStack[depth-1]
		}

		name := comapsID
		if depth > 0 {
			prefix := parentID + "_"
			if strings.HasPrefix(comapsID, prefix) {
				name = comapsID[len(prefix):]
			}
		}

		slug := strings.ToLower(strings.ReplaceAll(comapsID, " ", "_"))
		path := slug
		if depth > 0 {
			path = parentPath + "." + slug
		}

		sortOrder := siblingCounter[parentID]
		siblingCounter[parentID]++

		nodes = append(nodes, HierarchyNode{
			ComapsID:       comapsID,
			ParentComapsID: parentID,
			Path:           path,
			Name:           name,
			Depth:          depth,
			SortOrder:      sortOrder,
		})

		if depth < len(stack) {
			stack = stack[:depth]
			pathStack = pathStack[:depth]
		}
		stack = append(stack, comapsID)
		pathStack = append(pathStack, path)
		prevDepth = depth
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("hierarchy_parser: scan error: %w", err)
	}

	// Kind: a node is "group" iff the immediately following node is one
	// depth deeper; the last node is always "leaf".
	for i := range nodes {
		if i+1 < len(nodes) && nodes[i+1].Depth == nodes[i].Depth+1 {
			nodes[i].Kind = "group"
		} else {
			nodes[i].Kind = "leaf"
		}
	}

	return nodes, nil
}

// countLeadingSpaces returns the number of leading U+0020 space bytes at
// the start of s.
func countLeadingSpaces(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}
