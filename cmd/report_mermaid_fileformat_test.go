package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// newMermaidFileFormatStack builds a stack with one event and resource for the
// Mermaid file-format regression tests. Named uniquely to avoid clashing with
// helpers in other report test files.
func newMermaidFileFormatStack() lib.CfnStack {
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	return lib.CfnStack{
		Name: "test-stack",
		Id:   "arn:aws:cloudformation:us-east-1:111111111111:stack/test-stack/aaa",
		Events: []lib.StackEvent{
			{
				Type:      "Create",
				Success:   true,
				StartDate: now,
				EndDate:   now.Add(30 * time.Second),
				ResourceEvents: []lib.ResourceEvent{
					{
						EventType: "Add",
						Resource: lib.CfnResource{
							LogicalID:  "MyResource",
							Type:       "AWS::EC2::Instance",
							ResourceID: "i-12345",
						},
						StartDate:         now.Add(5 * time.Second),
						EndDate:           now.Add(15 * time.Second),
						EndStatus:         "CREATE_COMPLETE",
						ExpectedEndStatus: "CREATE_COMPLETE",
					},
				},
			},
		},
	}
}

// TestReportHasMermaid_FileFormatMarkdown is a regression test for T-1213.
// With a non-Mermaid console format (table) but a Mermaid-capable file format
// (markdown via output-file-format), the report must still include the Gantt
// chart so the written Markdown file is not missing its Mermaid diagram.
func TestReportHasMermaid_FileFormatMarkdown(t *testing.T) {
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

	// Console output is table (no Mermaid), file output is markdown (Mermaid).
	reportFlags = ReportFlags{}
	viper.Set("output", "table")
	viper.Set("output-file", "report.md")
	viper.Set("output-file-format", outputFormatMarkdown)

	reportFlags.HasMermaid = reportHasMermaidFormat()
	if !reportFlags.HasMermaid {
		t.Fatal("expected HasMermaid to be true when output-file-format is markdown")
	}

	awsConfig := config.AWSConfig{AccountID: "111111111111", Region: "us-east-1"}
	doc := output.New()
	if err := generateStackReport(context.Background(), newMermaidFileFormatStack(), doc, awsConfig); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chartCount := 0
	for _, content := range doc.Build().GetContents() {
		if content.Type() == output.ContentTypeRaw {
			chartCount++
		}
	}
	if chartCount == 0 {
		t.Error("expected Gantt chart in document when file-format is markdown, but none found")
	}
}

// TestReportHasMermaidFormat covers the format-combination logic across console
// and file formats so Mermaid is enabled when either destination is capable.
func TestReportHasMermaidFormat(t *testing.T) {
	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	t.Cleanup(func() {
		viper.Set("output", "")
		viper.Set("output-file", "")
		viper.Set("output-file-format", "")
	})

	tests := []struct {
		name           string
		output         string
		outputFile     string
		outputFileFmt  string
		wantHasMermaid bool
	}{
		{name: "console markdown", output: outputFormatMarkdown, wantHasMermaid: true},
		{name: "console html", output: outputFormatHTML, wantHasMermaid: true},
		{name: "console table only", output: "table", wantHasMermaid: false},
		{name: "table console with markdown file", output: "table", outputFile: "report.md", outputFileFmt: outputFormatMarkdown, wantHasMermaid: true},
		{name: "table console with html file", output: "table", outputFile: "report.html", outputFileFmt: outputFormatHTML, wantHasMermaid: true},
		{name: "table console with json file", output: "table", outputFile: "report.json", outputFileFmt: "json", wantHasMermaid: false},
		{name: "file format defaults to console", output: "table", outputFile: "report.txt", outputFileFmt: "", wantHasMermaid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set("output", tt.output)
			viper.Set("output-file", tt.outputFile)
			viper.Set("output-file-format", tt.outputFileFmt)

			if got := reportHasMermaidFormat(); got != tt.wantHasMermaid {
				t.Errorf("reportHasMermaidFormat() = %v, want %v", got, tt.wantHasMermaid)
			}
		})
	}
}
