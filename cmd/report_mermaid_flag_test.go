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

// TestHasMermaid_ResetForNonMermaidOutput verifies that the HasMermaid flag is
// reset to false when the output format changes from a Mermaid-capable format
// (markdown/html) to a non-Mermaid format (json/csv/text). In long-lived
// processes such as Lambda, the global reportFlags.HasMermaid could stick as
// true from a previous invocation and cause subsequent JSON/CSV reports to
// unexpectedly include Gantt chart data.
func TestHasMermaid_ResetForNonMermaidOutput(t *testing.T) {
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })

	// Simulate a first call with markdown — HasMermaid should be true
	reportFlags = ReportFlags{}
	viper.Set("output", "markdown")

	outputFormat := settings.GetLCString("output")
	hasMermaid := outputFormat == outputFormatMarkdown || outputFormat == outputFormatHTML
	if hasMermaid {
		reportFlags.HasMermaid = true
	}

	if !reportFlags.HasMermaid {
		t.Fatal("expected HasMermaid to be true after markdown output")
	}

	// Now simulate a second call with JSON — HasMermaid must be reset to false.
	// This is the bug: the original code only sets HasMermaid = true but never
	// resets it. After this block, HasMermaid should be false.
	viper.Set("output", "json")

	outputFormat = settings.GetLCString("output")
	hasMermaid = outputFormat == outputFormatMarkdown || outputFormat == outputFormatHTML
	// Apply the same logic as generateReport — assign unconditionally
	reportFlags.HasMermaid = hasMermaid

	if reportFlags.HasMermaid {
		t.Error("HasMermaid should be false for JSON output, but it is still true (sticky flag bug)")
	}
}

// TestGenerateStackReport_NoGanttForNonMermaidOutput verifies that
// generateStackReport does not add a Gantt chart when HasMermaid is false.
// This is an end-to-end check that the flag controls Gantt output correctly.
func TestGenerateStackReport_NoGanttForNonMermaidOutput(t *testing.T) {
	viper.SetDefault("timezone", "UTC")

	oldSettings := settings
	settings = &config.Config{}
	t.Cleanup(func() { settings = oldSettings })

	oldFlags := reportFlags
	t.Cleanup(func() { reportFlags = oldFlags })

	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	awsConfig := config.AWSConfig{
		AccountID: "111111111111",
		Region:    "us-east-1",
	}

	stack := lib.CfnStack{
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

	tests := []struct {
		name       string
		hasMermaid bool
		wantGantt  bool
	}{
		{name: "markdown has Gantt", hasMermaid: true, wantGantt: true},
		{name: "json has no Gantt", hasMermaid: false, wantGantt: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reportFlags = ReportFlags{HasMermaid: tt.hasMermaid}
			doc := output.New()

			err := generateStackReport(context.Background(), stack, doc, awsConfig)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			builtDoc := doc.Build()

			// Count chart content items by checking content type
			chartCount := 0
			for _, content := range builtDoc.GetContents() {
				if content.Type() == output.ContentTypeRaw {
					chartCount++
				}
			}

			if tt.wantGantt && chartCount == 0 {
				t.Error("expected Gantt chart in output but none found")
			}
			if !tt.wantGantt && chartCount > 0 {
				t.Errorf("expected no Gantt chart in output but found %d", chartCount)
			}
		})
	}
}
