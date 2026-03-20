package lib

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

func TestGlobToRegex(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pattern     string
		shouldMatch []string
		shouldNot   []string
	}{
		"plain name without wildcard": {
			pattern:     "my-stack",
			shouldMatch: []string{"my-stack"},
			shouldNot:   []string{"my-stackx", "xmy-stack", "other"},
		},
		"trailing wildcard": {
			pattern:     "my-stack-*",
			shouldMatch: []string{"my-stack-", "my-stack-v1", "my-stack-prod-v2"},
			shouldNot:   []string{"my-stack", "other-stack-v1"},
		},
		"leading wildcard": {
			pattern:     "*-stack",
			shouldMatch: []string{"-stack", "my-stack", "other-stack"},
			shouldNot:   []string{"stack", "stack-other"},
		},
		"middle wildcard": {
			pattern:     "my-*-stack",
			shouldMatch: []string{"my--stack", "my-prod-stack", "my-prod-v2-stack"},
			shouldNot:   []string{"mystack", "other-prod-stack"},
		},
		"dot in pattern is literal not regex any-char": {
			pattern:     "stack.v2-*",
			shouldMatch: []string{"stack.v2-", "stack.v2-prod"},
			shouldNot:   []string{"stackXv2-prod", "stack-v2-prod"},
		},
		"plus in pattern is literal": {
			pattern:     "stack+v2",
			shouldMatch: []string{"stack+v2"},
			shouldNot:   []string{"stackkv2", "stackv2"},
		},
		"brackets in pattern are literal": {
			pattern:     "stack[1]-*",
			shouldMatch: []string{"stack[1]-prod", "stack[1]-"},
			shouldNot:   []string{"stack1-prod", "stacka-prod"},
		},
		"parentheses in pattern are literal": {
			pattern:     "stack(prod)-*",
			shouldMatch: []string{"stack(prod)-v1"},
			shouldNot:   []string{"stackprod-v1"},
		},
		"question mark in pattern is literal not regex optional": {
			pattern:     "stack?name",
			shouldMatch: []string{"stack?name"},
			shouldNot:   []string{"stackname", "stackXname"},
		},
		"caret and dollar in pattern are literal": {
			pattern:     "^stack$",
			shouldMatch: []string{"^stack$"},
			shouldNot:   []string{"stack"},
		},
		"pipe in pattern is literal not regex alternation": {
			pattern:     "stack|other",
			shouldMatch: []string{"stack|other"},
			shouldNot:   []string{"stack", "other"},
		},
		"backslash in pattern is literal": {
			pattern:     `stack\name`,
			shouldMatch: []string{`stack\name`},
			shouldNot:   []string{"stackname"},
		},
		"multiple wildcards": {
			pattern:     "*-stack-*",
			shouldMatch: []string{"-stack-", "my-stack-v1", "x-stack-y"},
			shouldNot:   []string{"stackv1"},
		},
		"empty pattern": {
			pattern:     "",
			shouldMatch: []string{""},
			shouldNot:   []string{"anything"},
		},
		"only wildcard": {
			pattern:     "*",
			shouldMatch: []string{"", "anything", "stack.v2-prod"},
			shouldNot:   []string{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			re := GlobToRegex(tc.pattern)

			for _, s := range tc.shouldMatch {
				assert.True(t, re.MatchString(s), "expected %q to match pattern %q", s, tc.pattern)
			}
			for _, s := range tc.shouldNot {
				assert.False(t, re.MatchString(s), "expected %q NOT to match pattern %q", s, tc.pattern)
			}
		})
	}
}

// TestGetOutputsForStack_MetacharacterInFilter verifies that regex metacharacters
// in stack and export filter patterns are treated as literals, not regex operators.
// This is a regression test for T-443.
func TestGetOutputsForStack_MetacharacterInFilter(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stackName    string
		stackFilter  string
		exportFilter string
		exportName   string
		wantCount    int
	}{
		"dot in stack filter is literal": {
			stackName:   "stackXv2-prod",
			stackFilter: "stack.v2-*",
			exportName:  "Export1",
			wantCount:   0, // should NOT match because . must be literal
		},
		"dot in stack filter matches literal dot": {
			stackName:   "stack.v2-prod",
			stackFilter: "stack.v2-*",
			exportName:  "Export1",
			wantCount:   1, // should match because actual dot
		},
		"dot in export filter is literal": {
			stackName:    "my-stack",
			stackFilter:  "",
			exportFilter: "export.name-*",
			exportName:   "exportXname-v1",
			wantCount:    0, // should NOT match
		},
		"dot in export filter matches literal dot": {
			stackName:    "my-stack",
			stackFilter:  "",
			exportFilter: "export.name-*",
			exportName:   "export.name-v1",
			wantCount:    1, // should match
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stack := makeStackWithExport(tc.stackName, tc.exportName)
			got := getOutputsForStack(stack, tc.stackFilter, tc.exportFilter, true)
			assert.Len(t, got, tc.wantCount)
		})
	}
}

func makeStackWithExport(stackName, exportName string) types.Stack {
	return types.Stack{
		StackName: &stackName,
		Outputs: []types.Output{
			{
				OutputKey:   strPtr("Key1"),
				OutputValue: strPtr("Val1"),
				ExportName:  &exportName,
			},
		},
	}
}
