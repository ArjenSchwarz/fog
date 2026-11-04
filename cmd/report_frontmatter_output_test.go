package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// setReportOutputSettings points the global settings at the given console and
// file output configuration and restores the previous state after the test.
func setReportOutputSettings(t *testing.T, consoleFormat, outputFile, fileFormat string) {
	t.Helper()
	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() {
		settings = oldSettings
		viper.Set("output", "")
		viper.Set("output-file", "")
		viper.Set("output-file-format", "")
	})
	viper.Set("output", consoleFormat)
	viper.Set("output-file", outputFile)
	viper.Set("output-file-format", fileFormat)
}

// renderReportWithFrontMatter renders doc through the production path used by
// generateReport: output options from getReportOutputOptions, then renderReport.
func renderReportWithFrontMatter(doc *output.Document, awsConfig config.AWSConfig, frontMatter map[string]string) error {
	outputOptions, err := getReportOutputOptions(awsConfig, frontMatter)
	if err != nil {
		return err
	}
	return renderReport(doc, outputOptions, frontMatter)
}

// TestReportHasMarkdownFormat verifies that frontmatter generation is enabled
// when either the console or the file output is Markdown (T-1385), and stays
// disabled for non-Markdown combinations.
func TestReportHasMarkdownFormat(t *testing.T) {
	tests := map[string]struct {
		output        string
		outputFile    string
		outputFileFmt string
		want          bool
	}{
		"console markdown":                          {output: outputFormatMarkdown, want: true},
		"console table only":                        {output: "table", want: false},
		"console html only":                         {output: outputFormatHTML, want: false},
		"table console with markdown file":          {output: "table", outputFile: "report.md", outputFileFmt: outputFormatMarkdown, want: true},
		"table console with html file":              {output: "table", outputFile: "report.html", outputFileFmt: outputFormatHTML, want: false},
		"markdown file format ignored without file": {output: "table", outputFileFmt: outputFormatMarkdown, want: false},
		"file format defaults to console":           {output: outputFormatMarkdown, outputFile: "report.md", want: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			setReportOutputSettings(t, tc.output, tc.outputFile, tc.outputFileFmt)

			if got := reportHasMarkdownFormat(); got != tc.want {
				t.Errorf("reportHasMarkdownFormat() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRenderReport_FrontMatterInFileOutput is a regression test for T-1385.
// It renders a report through renderReport and checks that the written file
// contains frontmatter only when the file format is Markdown, regardless of
// the console format.
func TestRenderReport_FrontMatterInFileOutput(t *testing.T) {
	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })
	reportFlags.TargetBucket = ""

	frontMatter := map[string]string{"stack": "my-stack", "eventtype": "Update"}

	tests := map[string]struct {
		output        string
		fileName      string
		outputFileFmt string
		wantFM        bool
	}{
		"table console with markdown file":    {output: "table", fileName: "report.md", outputFileFmt: outputFormatMarkdown, wantFM: true},
		"markdown console with markdown file": {output: outputFormatMarkdown, fileName: "report.md", wantFM: true},
		"markdown console with json file":     {output: outputFormatMarkdown, fileName: "report.json", outputFileFmt: "json", wantFM: false},
		"table console with html file":        {output: "table", fileName: "report.html", outputFileFmt: outputFormatHTML, wantFM: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			outputFile := filepath.Join(t.TempDir(), tc.fileName)
			setReportOutputSettings(t, tc.output, outputFile, tc.outputFileFmt)

			doc := output.New().Header("Fog report for stack my-stack. Single event.").Build()
			if err := renderReportWithFrontMatter(doc, config.AWSConfig{}, frontMatter); err != nil {
				t.Fatalf("renderReport() error = %v", err)
			}

			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("failed to read report file: %v", err)
			}
			got := string(content)

			hasFM := strings.HasPrefix(got, "---\n") &&
				strings.Contains(got, "stack: \"my-stack\"\n") &&
				strings.Contains(got, "eventtype: Update\n")
			if hasFM != tc.wantFM {
				t.Errorf("frontmatter present = %v, want %v; file content:\n%s", hasFM, tc.wantFM, got)
			}
			if !strings.Contains(got, "my-stack") {
				t.Errorf("report body missing from file; content:\n%s", got)
			}
		})
	}
}

// TestRenderReport_TableStdoutMarkdownFileWithFrontMatterAndChart covers the
// combination of T-1385 and T-1389: with table console output and a Markdown
// file, stdout must contain neither the Mermaid chart nor frontmatter, while
// the Markdown file must contain both.
func TestRenderReport_TableStdoutMarkdownFileWithFrontMatterAndChart(t *testing.T) {
	viper.SetDefault("timezone", "UTC")
	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })
	reportFlags = ReportFlags{FrontMatter: true}

	filePath := filepath.Join(t.TempDir(), "report.md")
	setReportOutputSettings(t, "table", filePath, outputFormatMarkdown)

	if !reportHasMarkdownFormat() {
		t.Fatal("reportHasMarkdownFormat() = false, want true for a Markdown file")
	}
	stack := newMermaidFileFormatStack()
	// Same account and region that renderReportForTest uses for the body.
	awsConfig := config.AWSConfig{AccountID: "111111111111", Region: "us-east-1"}
	frontMatter, err := generateFrontMatter(t.Context(), map[string]lib.CfnStack{stack.Id: stack}, awsConfig)
	if err != nil {
		t.Fatalf("generateFrontMatter() error = %v", err)
	}

	stdout := renderReportForTest(t, frontMatter)

	if !strings.Contains(stdout, "MyResource") {
		t.Errorf("stdout is missing the events table; got:\n%s", stdout)
	}
	for _, token := range []string{"gantt", "Chart Type", "```mermaid", "eventtype:"} {
		if strings.Contains(stdout, token) {
			t.Errorf("table stdout contains %q; stdout:\n%s", token, stdout)
		}
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}
	file := string(content)
	if !strings.HasPrefix(file, "---\n") || !strings.Contains(file, "stack: \"test-stack\"\n") || !strings.Contains(file, "account: \"111111111111\"\n") {
		t.Errorf("Markdown file is missing the frontmatter; file:\n%s", file)
	}
	if !strings.Contains(file, "```mermaid") || !strings.Contains(file, "gantt") {
		t.Errorf("Markdown file is missing the Mermaid chart; file:\n%s", file)
	}
	if !strings.Contains(file, "MyResource") {
		t.Errorf("Markdown file is missing the events table; file:\n%s", file)
	}
}
