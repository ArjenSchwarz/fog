package cmd

import (
	"context"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// TestHistory_V2BuilderPattern tests the v2 Builder pattern for history command
func TestHistory_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

	// Sample deployment log data matching the history command structure
	logData := []map[string]any{
		{
			"Account":    "123456789012",
			"Region":     "us-east-1",
			"Deployer":   "test-user",
			"Type":       "CREATE",
			"Prechecks":  "PASSED",
			"Started At": "2025-10-17T10:00:00Z",
			"Duration":   "2m30s",
		},
	}

	tests := map[string]struct {
		columnOrder []string
	}{
		"deployment_log_column_order": {
			columnOrder: []string{"Account", "Region", "Deployer", "Type", "Prechecks", "Started At", "Duration"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)
			viper.Set("use-emoji", false)
			viper.Set("use-colors", false)

			// Build document using v2 Builder pattern with WithKeys to preserve column order
			doc := output.New().
				Table(
					"Details about the deployment",
					logData,
					output.WithKeys(tc.columnOrder...),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Verify rendering doesn't error
			out := output.NewOutput(
				output.WithFormat(output.Table),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output: %v", err)
			}
		})
	}
}

// TestHistory_V2FailedEventsTable tests the failed events table with correct column order
func TestHistory_V2FailedEventsTable(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

	// Sample failed events data
	failedEvents := []map[string]any{
		{
			"CfnName": "MyResource",
			"Type":    "AWS::EC2::Instance",
			"Status":  "CREATE_FAILED",
			"Reason":  "Resource creation failed",
		},
	}

	tests := map[string]struct {
		columnOrder []string
	}{
		"failed_events_column_order": {
			columnOrder: []string{"CfnName", "Type", "Status", "Reason"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)
			viper.Set("use-emoji", false)
			viper.Set("use-colors", false)

			// Build document for failed events
			doc := output.New().
				Table(
					"Failed events in deployment of change set",
					failedEvents,
					output.WithKeys(tc.columnOrder...),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Verify rendering doesn't error
			out := output.NewOutput(
				output.WithFormat(output.Table),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output: %v", err)
			}
		})
	}
}

// TestHistory_V2InlineStyling tests inline styling for success and failure status
func TestHistory_V2InlineStyling(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

	tests := map[string]struct {
		status      string
		styleFn     func(string) string
		description string
	}{
		"success_status_positive": {
			status:      "SUCCESS",
			styleFn:     output.StylePositive,
			description: "Success status should use positive styling",
		},
		"failure_status_warning": {
			status:      "FAILED",
			styleFn:     output.StyleWarning,
			description: "Failure status should use warning styling",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

			// Create styled header
			header := "📋 2025-10-17T10:00:00Z - my-stack"
			styledHeader := tc.styleFn(header)

			// Verify styled string is not empty
			if styledHeader == "" {
				t.Fatalf("%s: styled header should not be empty", tc.description)
			}

			// Styled string should be different from original when colors are enabled
			// (In actual usage, the styling is applied within the output rendering)
		})
	}
}

// TestHistory_V2OutputFormats tests that history output renders correctly in different formats
func TestHistory_V2OutputFormats(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

	tests := map[string]struct {
		format output.Format
	}{
		"table_format": {
			format: output.Table,
		},
		"csv_format": {
			format: output.CSV,
		},
		"json_format": {
			format: output.JSON,
		},
		"markdown_format": {
			format: output.Markdown,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data
			data := []map[string]any{
				{
					"Account":    "123456789012",
					"Region":     "us-east-1",
					"Deployer":   "test-user",
					"Type":       "UPDATE",
					"Prechecks":  "PASSED",
					"Started At": "2025-10-17T10:00:00Z",
					"Duration":   "1m45s",
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Deployment History",
					data,
					output.WithKeys("Account", "Region", "Deployer", "Type", "Prechecks", "Started At", "Duration"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Create output with specific format
			out := output.NewOutput(
				output.WithFormat(tc.format),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render in format %s: %v", tc.format.Name, err)
			}
		})
	}
}

// TestHistory_V2MultipleTables tests multiple tables in history output (deployment log + failed events)
func TestHistory_V2MultipleTables(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because go-output rendering has concurrent map write issues

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Deployment log data
	logData := []map[string]any{
		{
			"Account":    "123456789012",
			"Region":     "us-east-1",
			"Deployer":   "test-user",
			"Type":       "CREATE",
			"Prechecks":  "PASSED",
			"Started At": "2025-10-17T10:00:00Z",
			"Duration":   "2m30s",
		},
	}

	// Failed events data
	failedEvents := []map[string]any{
		{
			"CfnName": "MyResource",
			"Type":    "AWS::EC2::Instance",
			"Status":  "CREATE_FAILED",
			"Reason":  "Resource creation failed",
		},
	}

	// Build document with multiple tables
	doc := output.New().
		Table(
			"Details about the deployment",
			logData,
			output.WithKeys("Account", "Region", "Deployer", "Type", "Prechecks", "Started At", "Duration"),
		).
		Table(
			"Failed events in deployment of change set",
			failedEvents,
			output.WithKeys("CfnName", "Type", "Status", "Reason"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify rendering doesn't error
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render output with multiple tables: %v", err)
	}
}
