package cmd

import (
	"slices"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
)

// filteredStackNames runs getFilteredStacks the same way showDependencies does
// (deduplicated) and sorts the result so it can be compared regardless of map
// iteration order.
func filteredStackNames(filter string, stacks map[string]lib.CfnStack) []string {
	result := unique(getFilteredStacks(filter, &stacks))
	slices.Sort(result)
	return result
}

// TestGetFilteredStacks covers the regression for T-1470: import cycles between
// stacks must not cause unbounded recursion, while acyclic graphs keep their
// existing filtering results. Each stack's ImportedBy lists the stacks that
// import its exports.
func TestGetFilteredStacks(t *testing.T) {
	tests := map[string]struct {
		stacks map[string]lib.CfnStack
		filter string
		want   []string
	}{
		"acyclic chain includes importers and their dependencies": {
			// web imports from app, app imports from vpc and shared.
			stacks: map[string]lib.CfnStack{
				"vpc":    {Name: "vpc", ImportedBy: []string{"app"}},
				"shared": {Name: "shared", ImportedBy: []string{"app"}},
				"app":    {Name: "app", ImportedBy: []string{"web"}},
				"web":    {Name: "web"},
				"other":  {Name: "other"},
			},
			filter: "vpc",
			want:   []string{"app", "shared", "vpc", "web"},
		},
		"acyclic chain filtered from the end only includes direct dependencies": {
			stacks: map[string]lib.CfnStack{
				"vpc": {Name: "vpc", ImportedBy: []string{"app"}},
				"app": {Name: "app", ImportedBy: []string{"web"}},
				"web": {Name: "web"},
			},
			filter: "web",
			want:   []string{"app", "web"},
		},
		"direct cycle": {
			stacks: map[string]lib.CfnStack{
				"a":     {Name: "a", ImportedBy: []string{"b"}},
				"b":     {Name: "b", ImportedBy: []string{"a"}},
				"other": {Name: "other"},
			},
			filter: "a",
			want:   []string{"a", "b"},
		},
		"indirect cycle": {
			stacks: map[string]lib.CfnStack{
				"a":     {Name: "a", ImportedBy: []string{"b"}},
				"b":     {Name: "b", ImportedBy: []string{"c"}},
				"c":     {Name: "c", ImportedBy: []string{"a"}},
				"other": {Name: "other"},
			},
			filter: "a",
			want:   []string{"a", "b", "c"},
		},
		"self import": {
			stacks: map[string]lib.CfnStack{
				"a": {Name: "a", ImportedBy: []string{"a"}},
			},
			filter: "a",
			want:   []string{"a"},
		},
		"wildcard filter with cycle": {
			stacks: map[string]lib.CfnStack{
				"dev-a":  {Name: "dev-a", ImportedBy: []string{"dev-b"}},
				"dev-b":  {Name: "dev-b", ImportedBy: []string{"dev-a"}},
				"prod-a": {Name: "prod-a"},
			},
			filter: "dev-*",
			want:   []string{"dev-a", "dev-b"},
		},
		"wildcard filter reaching a cycle outside the pattern": {
			stacks: map[string]lib.CfnStack{
				"dev-vpc": {Name: "dev-vpc", ImportedBy: []string{"app"}},
				"app":     {Name: "app", ImportedBy: []string{"web"}},
				"web":     {Name: "web", ImportedBy: []string{"app"}},
				"prod":    {Name: "prod"},
			},
			filter: "dev-*",
			want:   []string{"app", "dev-vpc", "web"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := filteredStackNames(tc.filter, tc.stacks)
			if !slices.Equal(got, tc.want) {
				t.Errorf("getFilteredStacks(%q) = %v, want %v", tc.filter, got, tc.want)
			}
		})
	}
}
