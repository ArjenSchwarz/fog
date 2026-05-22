package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestHistory_V2BuilderPattern tests the v2 Builder pattern for history command
func TestHistory_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

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
			// NOTE: Cannot use t.Parallel() because viper uses global state

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
				output.WithFormat(output.Table()),
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
	// NOTE: Cannot use t.Parallel() because viper uses global state

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
			// NOTE: Cannot use t.Parallel() because viper uses global state

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
				output.WithFormat(output.Table()),
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
	// NOTE: Cannot use t.Parallel() because viper uses global state

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
			// NOTE: Cannot use t.Parallel() because viper uses global state

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
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		format output.Format
	}{
		"table_format": {
			format: output.Table(),
		},
		"csv_format": {
			format: output.CSV(),
		},
		"json_format": {
			format: output.JSON(),
		},
		"markdown_format": {
			format: output.Markdown(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

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
	// NOTE: Cannot use t.Parallel() because viper uses global state

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
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render output with multiple tables: %v", err)
	}
}

// newFailedDeploymentLog builds a FAILED deployment log with a single failed event.
func newFailedDeploymentLog() lib.DeploymentLog {
	started := time.Date(2025, 10, 17, 10, 0, 0, 0, time.UTC)
	return lib.DeploymentLog{
		Account:        "123456789012",
		Region:         "us-east-1",
		Deployer:       "test-user",
		DeploymentType: lib.DeploymentTypeCreateStack,
		StackName:      "failed-stack",
		Status:         lib.DeploymentLogStatusFailed,
		PreChecks:      "PASSED",
		StartedAt:      started,
		UpdatedAt:      started.Add(time.Minute),
		Failures: []map[string]any{
			{
				colCfnName: "MyResource",
				"Type":     "AWS::EC2::Instance",
				colStatus:  "CREATE_FAILED",
				colReason:  "Resource creation failed",
			},
		},
	}
}

// TestPrintLog_FailedEventsRespectStderr verifies that for a FAILED deployment log
// the failed-events table follows the useStderr flag: stderr-only in deploy context,
// stdout in the normal history command path.
func TestPrintLog_FailedEventsRespectStderr(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	setupViperDefaults()
	t.Cleanup(viper.Reset) // restore global state for other cmd tests
	viper.Set("output-file", "")

	// Failed-event cell values are preserved verbatim by the table renderer,
	// unlike the header text which gets uppercased.
	const failedResource = "MyResource"
	const failedReason = "Resource creation failed"

	t.Run("useStderr true routes failed events to stderr", func(t *testing.T) {
		stdout, stderr := captureBothStreams(func() {
			printLog(newFailedDeploymentLog(), true)
		})

		// Failed-events section must stay on stderr in deploy context
		assert.Contains(t, stderr, failedResource)
		assert.Contains(t, stderr, failedReason)

		// stdout must stay clean so only final structured output reaches it
		assert.NotContains(t, stdout, failedResource)
		assert.NotContains(t, stdout, failedReason)
	})

	t.Run("useStderr false routes failed events to stdout", func(t *testing.T) {
		stdout, stderr := captureBothStreams(func() {
			printLog(newFailedDeploymentLog(), false)
		})

		// Normal history command path renders to stdout via renderDocument
		assert.Contains(t, stdout, failedResource)
		assert.Contains(t, stdout, failedReason)

		// stderr must stay clean so failed events don't leak to both streams
		assert.NotContains(t, stderr, failedResource)
		assert.NotContains(t, stderr, failedReason)
	})
}
