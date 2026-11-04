package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// TestReportFrontMatter_YAMLRoundTrip is a regression test for T-1385. It
// generates frontmatter for a realistic stack, renders it to a Markdown report
// file, and parses the frontmatter block back as YAML. Account IDs with a
// leading zero must survive as exact strings and the summary must be the
// metadata HTML table without stray quote characters.
func TestReportFrontMatter_YAMLRoundTrip(t *testing.T) {
	viper.SetDefault("timezone", "UTC")
	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })
	reportFlags.TargetBucket = ""
	reportFlags.LatestOnly = false

	start := time.Date(2025, 3, 10, 8, 0, 0, 0, time.UTC)
	stack := lib.CfnStack{
		Name: "my-stack",
		Id:   "arn:aws:cloudformation:eu-west-1:012345678901:stack/my-stack/xxx",
		Events: []lib.StackEvent{
			{Type: "Update", Success: true, StartDate: start, EndDate: start.Add(61 * time.Second)},
		},
	}
	stacks := map[string]lib.CfnStack{stack.Id: stack}

	tests := map[string]struct {
		accountAlias     string
		wantAccountAlias string
	}{
		"without account alias": {accountAlias: "", wantAccountAlias: "012345678901"},
		"with account alias":    {accountAlias: "demo", wantAccountAlias: "demo (012345678901)"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			outputFile := filepath.Join(t.TempDir(), "report.md")
			setReportOutputSettings(t, "table", outputFile, outputFormatMarkdown)
			viper.Set("timezone", "UTC")
			t.Cleanup(func() { viper.Set("timezone", "") })

			awsConfig := config.AWSConfig{AccountID: "012345678901", AccountAlias: tc.accountAlias, Region: "eu-west-1"}

			frontMatter, err := generateFrontMatter(context.Background(), stacks, awsConfig)
			if err != nil {
				t.Fatalf("generateFrontMatter() error = %v", err)
			}
			if err := renderReportWithFrontMatter(output.New().Header("Fog report").Build(), awsConfig, frontMatter); err != nil {
				t.Fatalf("renderReport() error = %v", err)
			}

			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read report file: %v", err)
			}
			block, ok := extractFrontMatterBlock(string(content))
			if !ok {
				t.Fatalf("no frontmatter block in report file:\n%s", content)
			}

			var got map[string]any
			if err := yaml.Unmarshal([]byte(block), &got); err != nil {
				t.Fatalf("frontmatter is not valid YAML: %v\n%s", err, block)
			}

			_, metadata := createMetadataTable(stack, stack.Events[0], awsConfig)
			wantStrings := map[string]string{
				"account":      "012345678901",
				"accountalias": tc.wantAccountAlias,
				"region":       "eu-west-1",
				"stack":        "my-stack",
				"date":         "2025-03-10T08:00:00Z",
				"duration":     "1m1s",
				"eventtype":    "Update",
				"summary":      buildSimpleHTMLTable(metadata[0]),
			}
			for key, want := range wantStrings {
				value, isString := got[key].(string)
				if !isString || value != want {
					t.Errorf("frontmatter %q = %#v (%T), want string %q\n%s", key, got[key], got[key], want, block)
				}
			}
			if got["success"] != true {
				t.Errorf("frontmatter success = %#v, want true", got["success"])
			}
		})
	}
}

// extractFrontMatterBlock returns the YAML between the opening and closing
// "---" lines at the start of a Markdown document.
func extractFrontMatterBlock(doc string) (string, bool) {
	rest, ok := strings.CutPrefix(doc, "---\n")
	if !ok {
		return "", false
	}
	block, _, found := strings.Cut(rest, "\n---\n")
	return block, found
}
