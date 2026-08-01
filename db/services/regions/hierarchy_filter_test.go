package regions

import (
	"reflect"
	"testing"
)

func TestAncestorGroupPaths(t *testing.T) {
	set := func(paths ...string) map[string]struct{} {
		m := make(map[string]struct{}, len(paths))
		for _, p := range paths {
			m[p] = struct{}{}
		}
		return m
	}

	cases := []struct {
		name  string
		leafs []string
		want  map[string]struct{}
	}{
		{"no leaves -> empty", nil, set()},
		{"depth-0 leaf has no ancestors", []string{"canada"}, set()},
		{
			"deep leaf yields every prefix but itself",
			[]string{"canada.alberta.south"},
			set("canada", "canada.alberta"),
		},
		{
			"siblings under one parent dedup shared ancestors",
			[]string{"canada.alberta.south", "canada.alberta.north"},
			set("canada", "canada.alberta"),
		},
		{
			"unrelated branches keep their own chains",
			[]string{"canada.alberta.south", "germany.bavaria"},
			set("canada", "canada.alberta", "germany"),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := AncestorGroupPaths(c.leafs); !reflect.DeepEqual(got, c.want) {
				t.Errorf("AncestorGroupPaths(%v) = %v, want %v", c.leafs, got, c.want)
			}
		})
	}
}
