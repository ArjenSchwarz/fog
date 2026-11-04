package cmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// renderReportForTest builds the report for the shared Mermaid test stack using
// the current viper output settings and renders it through renderReport with
// the given frontmatter (nil for none). It returns what was written to stdout.
func renderReportForTest(t *testing.T, frontMatter map[string]string) string {
	t.Helper()

	reportFlags = ReportFlags{}
	reportFlags.HasMermaid = reportHasMermaidFormat()

	awsConfig := config.AWSConfig{AccountID: "111111111111", Region: "us-east-1"}
	doc := output.New()
	if err := generateStackReport(context.Background(), newMermaidFileFormatStack(), doc, awsConfig); err != nil {
		t.Fatalf("generateStackReport: %v", err)
	}

	// Redirect stdout before creating the output options, because the
	// stdout writer captures os.Stdout when it is constructed.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldStdout })

	outputOptions, err := getReportOutputOptions(awsConfig, frontMatter)
	if err != nil {
		t.Fatalf("getReportOutputOptions: %v", err)
	}
	renderErr := renderReport(doc.Build(), outputOptions, frontMatter)

	_ = w.Close()
	os.Stdout = oldStdout
	stdout, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading stdout: %v", err)
	}
	if renderErr != nil {
		t.Fatalf("renderReport: %v", renderErr)
	}
	return string(stdout)
}

// TestRenderReport_MermaidOnlyInMermaidDestinations is a regression test for
// T-1389. The Gantt chart is added to the shared report document whenever any
// destination supports Mermaid, so renderReport must keep it out of
// destinations that don't (such as table stdout) while still writing it to a
// Markdown/HTML file.
func TestRenderReport_MermaidOnlyInMermaidDestinations(t *testing.T) {
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })

	t.Cleanup(func() {
		viper.Set("output", "")
		viper.Set("output-file", "")
		viper.Set("output-file-format", "")
	})

	tests := map[string]struct {
		consoleFormat    string
		fileName         string
		fileFormat       string
		wantStdoutChart  bool
		wantFileChart    bool
		stdoutChartToken string
	}{
		"table stdout with markdown file": {
			consoleFormat:    "table",
			fileName:         "report.md",
			fileFormat:       outputFormatMarkdown,
			wantStdoutChart:  false,
			wantFileChart:    true,
			stdoutChartToken: "gantt",
		},
		"table stdout with html file": {
			consoleFormat:    "table",
			fileName:         "report.html",
			fileFormat:       outputFormatHTML,
			wantStdoutChart:  false,
			wantFileChart:    true,
			stdoutChartToken: "gantt",
		},
		"markdown stdout with table file": {
			consoleFormat:    outputFormatMarkdown,
			fileName:         "report.txt",
			fileFormat:       "table",
			wantStdoutChart:  true,
			wantFileChart:    false,
			stdoutChartToken: "```mermaid",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), tc.fileName)
			viper.Set("output", tc.consoleFormat)
			viper.Set("output-file", filePath)
			viper.Set("output-file-format", tc.fileFormat)

			stdout := renderReportForTest(t, nil)

			// The report data itself must still reach stdout.
			if !strings.Contains(stdout, "MyResource") {
				t.Errorf("stdout is missing the events table; got:\n%s", stdout)
			}
			if got := strings.Contains(stdout, tc.stdoutChartToken); got != tc.wantStdoutChart {
				t.Errorf("stdout contains %q = %v, want %v; stdout:\n%s", tc.stdoutChartToken, got, tc.wantStdoutChart, stdout)
			}
			if !tc.wantStdoutChart && strings.Contains(stdout, "Chart Type") {
				t.Errorf("stdout contains a text dump of the chart; stdout:\n%s", stdout)
			}

			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("reading output file: %v", err)
			}
			file := string(fileContent)
			if !strings.Contains(file, "MyResource") {
				t.Errorf("output file is missing the events table; got:\n%s", file)
			}
			if got := strings.Contains(file, "gantt"); got != tc.wantFileChart {
				t.Errorf("output file contains gantt chart = %v, want %v; file:\n%s", got, tc.wantFileChart, file)
			}
		})
	}
}
