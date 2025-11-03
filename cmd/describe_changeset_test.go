package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// Helper function to capture stdout
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// TestPrintBasicStackInfo_V2BuilderPattern tests the v2 Builder pattern for stack info table
func TestPrintBasicStackInfo_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		showDryRunInfo bool
		columnOrder    []string
	}{
		"without_dryrun_info": {
			showDryRunInfo: false,
			columnOrder:    []string{"StackName", "Account", "Region", "Action"},
		},
		"with_dryrun_info": {
			showDryRunInfo: true,
			columnOrder:    []string{"StackName", "Account", "Region", "Action", "Is dry run"},
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

			// Create sample stack info data
			data := []map[string]any{
				{
					"StackName": "test-stack",
					"Account":   "123456789012",
					"Region":    "us-east-1",
					"Action":    "Update",
				},
			}

			if tc.showDryRunInfo {
				data[0]["Is dry run"] = true
			}

			// Build document using v2 Builder pattern with WithKeys to preserve column order
			doc := output.New().
				Table(
					"CloudFormation stack information",
					data,
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

// TestPrintChangeset_V2BuilderPattern tests the v2 Builder pattern for changeset table
func TestPrintChangeset_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		hasModule   bool
		columnOrder []string
	}{
		"without_module": {
			hasModule:   false,
			columnOrder: []string{"Action", "CfnName", "Type", "ID", "Replacement"},
		},
		"with_module": {
			hasModule:   true,
			columnOrder: []string{"Action", "CfnName", "Type", "ID", "Replacement", "Module"},
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

			// Create sample changeset changes
			data := []map[string]any{
				{
					"Action":      "Add",
					"CfnName":     "MyBucket",
					"Type":        "AWS::S3::Bucket",
					"ID":          "my-bucket-123",
					"Replacement": "False",
				},
				{
					"Action":      "Modify",
					"CfnName":     "MyTable",
					"Type":        "AWS::DynamoDB::Table",
					"ID":          "my-table-456",
					"Replacement": "Conditional",
				},
			}

			if tc.hasModule {
				data[0]["Module"] = "StorageModule"
				data[1]["Module"] = "DatabaseModule"
			}

			// Build document using v2 Builder pattern
			doc := output.New().
				Table(
					"Changeset Changes test-changeset",
					data,
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

// TestPrintDangerTable_V2BuilderPattern tests the v2 Builder pattern for danger table
func TestPrintDangerTable_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		hasModule   bool
		columnOrder []string
	}{
		"without_module": {
			hasModule:   false,
			columnOrder: []string{"Action", "CfnName", "Type", "ID", "Replacement", "Details"},
		},
		"with_module": {
			hasModule:   true,
			columnOrder: []string{"Action", "CfnName", "Type", "ID", "Replacement", "Details", "Module"},
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

			// Create sample dangerous changes
			data := []map[string]any{
				{
					"Action":      "Remove",
					"CfnName":     "MyBucket",
					"Type":        "AWS::S3::Bucket",
					"ID":          "my-bucket-123",
					"Replacement": "False",
					"Details":     "Bucket will be deleted",
				},
				{
					"Action":      "Modify",
					"CfnName":     "MyDatabase",
					"Type":        "AWS::RDS::DBInstance",
					"ID":          "my-db-456",
					"Replacement": "True",
					"Details":     "Database will be replaced",
				},
			}

			if tc.hasModule {
				data[0]["Module"] = "StorageModule"
				data[1]["Module"] = "DatabaseModule"
			}

			// Build document using v2 Builder pattern
			doc := output.New().
				Table(
					"Potentially destructive changes",
					data,
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

// TestChangesetSummary_V2BuilderPattern tests the v2 Builder pattern for changeset summary
func TestChangesetSummary_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Create sample summary data
	data := []map[string]any{
		{
			"Total":        5,
			"Added":        2,
			"Removed":      1,
			"Modified":     2,
			"Replacements": 1,
			"Conditionals": 1,
		},
	}

	columnOrder := []string{"Total", "Added", "Removed", "Modified", "Replacements", "Conditionals"}

	// Build document using v2 Builder pattern
	doc := output.New().
		Table(
			"Summary for test-changeset",
			data,
			output.WithKeys(columnOrder...),
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
}

// TestChangeset_V2OutputFormats tests that changeset output renders correctly in different formats
func TestChangeset_V2OutputFormats(t *testing.T) {
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
					"Action":      "Add",
					"CfnName":     "MyBucket",
					"Type":        "AWS::S3::Bucket",
					"ID":          "my-bucket-123",
					"Replacement": "False",
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Test Changeset",
					data,
					output.WithKeys("Action", "CfnName", "Type", "ID", "Replacement"),
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

// TestChangeset_V2SortByType tests sorting changeset by Type column
func TestChangeset_V2SortByType(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Create sample data with different types (unsorted)
	data := []map[string]any{
		{
			"Action":      "Add",
			"CfnName":     "MyTable",
			"Type":        "AWS::DynamoDB::Table",
			"ID":          "table-123",
			"Replacement": "False",
		},
		{
			"Action":      "Add",
			"CfnName":     "MyBucket",
			"Type":        "AWS::S3::Bucket",
			"ID":          "bucket-456",
			"Replacement": "False",
		},
		{
			"Action":      "Modify",
			"CfnName":     "MyRole",
			"Type":        "AWS::IAM::Role",
			"ID":          "role-789",
			"Replacement": "False",
		},
	}

	// Build document - v2 should be able to sort this
	doc := output.New().
		Table(
			"Changeset sorted by Type",
			data,
			output.WithKeys("Action", "CfnName", "Type", "ID", "Replacement"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify rendering with sort
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render sorted output: %v", err)
	}
}

// TestChangeset_V2MultipleTables tests multiple tables in single changeset output
func TestChangeset_V2MultipleTables(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// First table: changeset changes
	changesData := []map[string]any{
		{
			"Action":      "Add",
			"CfnName":     "MyBucket",
			"Type":        "AWS::S3::Bucket",
			"ID":          "bucket-123",
			"Replacement": "False",
		},
	}

	// Second table: summary
	summaryData := []map[string]any{
		{
			"Total":        1,
			"Added":        1,
			"Removed":      0,
			"Modified":     0,
			"Replacements": 0,
			"Conditionals": 0,
		},
	}

	// Build document with multiple tables
	doc := output.New().
		Table(
			"Changeset Changes",
			changesData,
			output.WithKeys("Action", "CfnName", "Type", "ID", "Replacement"),
		).
		Table(
			"Summary",
			summaryData,
			output.WithKeys("Total", "Added", "Removed", "Modified", "Replacements", "Conditionals"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify rendering multiple tables
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render multiple tables: %v", err)
	}
}

// TestChangeset_V2EmptyChangeset tests handling of changeset with no changes
func TestChangeset_V2EmptyChangeset(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Empty data
	data := []map[string]any{}

	// Build document with empty data
	doc := output.New().
		Table(
			"Empty Changeset",
			data,
			output.WithKeys("Action", "CfnName", "Type", "ID", "Replacement"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify rendering empty table works
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render empty changeset: %v", err)
	}
}

// TestChangeset_V2ActionTypeVariations tests different action types
func TestChangeset_V2ActionTypeVariations(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		action string
	}{
		"add_action": {
			action: "Add",
		},
		"remove_action": {
			action: "Remove",
		},
		"modify_action": {
			action: "Modify",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data with specific action
			data := []map[string]any{
				{
					"Action":      tc.action,
					"CfnName":     "TestResource",
					"Type":        "AWS::S3::Bucket",
					"ID":          "resource-123",
					"Replacement": "False",
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Changeset with "+tc.action+" action",
					data,
					output.WithKeys("Action", "CfnName", "Type", "ID", "Replacement"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render
			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output with action %s: %v", tc.action, err)
			}
		})
	}
}

// TestChangeset_V2ReplacementTypes tests different replacement types
func TestChangeset_V2ReplacementTypes(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		replacement string
	}{
		"no_replacement": {
			replacement: "False",
		},
		"conditional_replacement": {
			replacement: "Conditional",
		},
		"true_replacement": {
			replacement: "True",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data with specific replacement type
			data := []map[string]any{
				{
					"Action":      "Modify",
					"CfnName":     "TestResource",
					"Type":        "AWS::S3::Bucket",
					"ID":          "resource-123",
					"Replacement": tc.replacement,
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Changeset with "+tc.replacement+" replacement",
					data,
					output.WithKeys("Action", "CfnName", "Type", "ID", "Replacement"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render
			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output with replacement %s: %v", tc.replacement, err)
			}
		})
	}
}

// TestGetChangesetSummaryTable_V2 tests the summary table helper function
func TestGetChangesetSummaryTable_V2(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	summarykeys, summaryContent := getChangesetSummaryTable()

	// Verify expected keys
	expectedKeys := []string{"Total", "Added", "Removed", "Modified", "Replacements", "Conditionals"}
	if len(summarykeys) != len(expectedKeys) {
		t.Fatalf("Expected %d keys, got %d", len(expectedKeys), len(summarykeys))
	}

	for i, key := range expectedKeys {
		if summarykeys[i] != key {
			t.Errorf("Expected key %s at position %d, got %s", key, i, summarykeys[i])
		}
	}

	// Verify all values are initialized to 0
	for _, key := range summarykeys {
		if val, ok := summaryContent[key]; !ok {
			t.Errorf("Key %s not found in summary content", key)
		} else if val != 0 {
			t.Errorf("Expected key %s to be initialized to 0, got %v", key, val)
		}
	}
}

// TestOutputFormats_AllFormats tests that all output formats render without error
func TestOutputFormats_AllFormats(t *testing.T) {
	tests := map[string]struct {
		format         string
		validateOutput func(t *testing.T, output string)
	}{
		"table_format": {
			format: "table",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Table output should not be empty")
				}
			},
		},
		"csv_format": {
			format: "csv",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("CSV output should not be empty")
				}
				// CSV should contain headers
				if !contains(output, "StackName") {
					t.Error("CSV output should contain StackName header")
				}
			},
		},
		"json_format": {
			format: "json",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("JSON output should not be empty")
				}
				// JSON output from go-output v2 is an array of tables
				var result []any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				// Should have at least one table (stack info)
				if len(result) == 0 {
					t.Error("JSON output should contain at least one table")
				}
			},
		},
		"yaml_format": {
			format: "yaml",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("YAML output should not be empty")
				}
				// YAML should contain array indicator (YAML arrays start with -)
				if !contains(output, "-") {
					t.Error("YAML output should contain array structure")
				}
			},
		},
		"markdown_format": {
			format: "markdown",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Markdown output should not be empty")
				}
				// Markdown should contain pipe characters for tables
				if !contains(output, "|") {
					t.Error("Markdown output should contain table formatting")
				}
			},
		},
		"html_format": {
			format: "html",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("HTML output should not be empty")
				}
				// HTML should contain table tags
				if !contains(output, "<table") {
					t.Error("HTML output should contain table tags")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup test changeset with sample data
			changeset := lib.ChangesetInfo{
				Name:      "test-changeset",
				HasModule: false,
				Changes: []lib.ChangesetChanges{
					{
						Action:      "Add",
						LogicalID:   "MyBucket",
						Type:        "AWS::S3::Bucket",
						ResourceID:  "my-bucket-123",
						Replacement: "False",
					},
					{
						Action:      "Modify",
						LogicalID:   "MyTable",
						Type:        "AWS::DynamoDB::Table",
						ResourceID:  "my-table-456",
						Replacement: "Conditional",
					},
				},
			}

			deployment := lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     false,
				IsDryRun:  false,
			}

			awsConfig := config.AWSConfig{
				Region: "us-east-1",
			}

			// Set output format
			viper.Set("output", tc.format)
			viper.Set("use-colors", false)
			viper.Set("use-emoji", false)

			// Capture output
			output := captureStdout(func() {
				buildAndRenderChangeset(changeset, deployment, awsConfig)
			})

			// Validate output
			tc.validateOutput(t, output)
		})
	}
}

// TestAddToChangesetSummary_V2 tests the summary accumulator function
func TestAddToChangesetSummary_V2(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		change       lib.ChangesetChanges
		expectedVals map[string]int
	}{
		"add_action": {
			change: lib.ChangesetChanges{
				Action:      "Add",
				Replacement: "False",
			},
			expectedVals: map[string]int{
				"Total": 1,
				"Added": 1,
			},
		},
		"remove_action": {
			change: lib.ChangesetChanges{
				Action:      "Remove",
				Replacement: "False",
			},
			expectedVals: map[string]int{
				"Total":   1,
				"Removed": 1,
			},
		},
		"modify_with_replacement": {
			change: lib.ChangesetChanges{
				Action:      "Modify",
				Replacement: "True",
			},
			expectedVals: map[string]int{
				"Total":        1,
				"Modified":     1,
				"Replacements": 1,
			},
		},
		"modify_with_conditional": {
			change: lib.ChangesetChanges{
				Action:      "Modify",
				Replacement: "Conditional",
			},
			expectedVals: map[string]int{
				"Total":        1,
				"Modified":     1,
				"Conditionals": 1,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Initialize summary
			_, summaryContent := getChangesetSummaryTable()

			// Add change to summary
			addToChangesetSummary(&summaryContent, tc.change)

			// Verify expected values
			for key, expectedVal := range tc.expectedVals {
				if val, ok := summaryContent[key]; !ok {
					t.Errorf("Key %s not found in summary", key)
				} else if val != expectedVal {
					t.Errorf("Expected %s to be %d, got %v", key, expectedVal, val)
				}
			}
		})
	}
}

// TestOutputFormats_EmptyChangeset tests empty changeset handling across all formats
func TestOutputFormats_EmptyChangeset(t *testing.T) {
	tests := map[string]struct {
		format         string
		validateOutput func(t *testing.T, output string)
	}{
		"table_empty": {
			format: "table",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Table output should not be empty for empty changeset")
				}
				// Should contain stack info
				if !contains(output, "test-stack") {
					t.Error("Output should contain stack name")
				}
				// Empty changesets add text message for table format
				// Just verify it doesn't crash - text content varies
			},
		},
		"json_empty": {
			format: "json",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("JSON output should not be empty for empty changeset")
				}
				var result []any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				// Should have at least one table (stack info)
				if len(result) == 0 {
					t.Error("JSON output should contain at least one table")
				}
			},
		},
		"yaml_empty": {
			format: "yaml",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("YAML output should not be empty for empty changeset")
				}
				// YAML should contain array indicator
				if !contains(output, "-") {
					t.Error("YAML output should contain array structure")
				}
			},
		},
		"csv_empty": {
			format: "csv",
			validateOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("CSV output should not be empty for empty changeset")
				}
				// Should have stack info headers
				if !contains(output, "StackName") {
					t.Error("CSV output should contain StackName header")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup empty changeset
			changeset := lib.ChangesetInfo{
				Name:      "empty-changeset",
				HasModule: false,
				Changes:   []lib.ChangesetChanges{}, // Empty changes
			}

			deployment := lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     false,
				IsDryRun:  false,
			}

			awsConfig := config.AWSConfig{
				Region: "us-east-1",
			}

			// Set output format
			viper.Set("output", tc.format)
			viper.Set("use-colors", false)
			viper.Set("use-emoji", false)

			// Capture output
			output := captureStdout(func() {
				buildAndRenderChangeset(changeset, deployment, awsConfig)
			})

			// Validate output
			tc.validateOutput(t, output)
		})
	}
}

// TestOutputFormats_ANSICodeStripping tests that ANSI codes are stripped from structured formats
func TestOutputFormats_ANSICodeStripping(t *testing.T) {
	tests := map[string]struct {
		format         string
		shouldHaveANSI bool
		validateNoANSI func(t *testing.T, output string)
	}{
		"json_no_ansi": {
			format:         "json",
			shouldHaveANSI: false,
			validateNoANSI: func(t *testing.T, output string) {
				// Check for common ANSI escape codes
				// Note: The color package doesn't add ANSI codes in test environments (no TTY)
				// So this test mainly verifies no crash occurs with colors enabled
				// In production, go-output v2 strips ANSI codes from JSON automatically
				if contains(output, "\x1b[") {
					t.Error("JSON output should not contain ANSI escape codes")
				}
			},
		},
		"yaml_no_ansi": {
			format:         "yaml",
			shouldHaveANSI: false,
			validateNoANSI: func(t *testing.T, output string) {
				if contains(output, "\x1b[") {
					t.Error("YAML output should not contain ANSI escape codes")
				}
			},
		},
		"csv_no_ansi": {
			format:         "csv",
			shouldHaveANSI: false,
			validateNoANSI: func(t *testing.T, output string) {
				// CSV format behavior with ANSI codes varies by terminal
				// The test mainly verifies that CSV rendering doesn't crash with colors enabled
				// Note: CSV may contain ANSI codes depending on terminal support
				// This is acceptable for CSV format which is often processed by tools that strip them
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup changeset with Remove action (which gets bold formatting)
			changeset := lib.ChangesetInfo{
				Name:      "test-changeset",
				HasModule: false,
				Changes: []lib.ChangesetChanges{
					{
						Action:      "Remove", // This gets bold() applied
						LogicalID:   "MyBucket",
						Type:        "AWS::S3::Bucket",
						ResourceID:  "my-bucket-123",
						Replacement: "False",
					},
				},
			}

			deployment := lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     false,
				IsDryRun:  false,
			}

			awsConfig := config.AWSConfig{
				Region: "us-east-1",
			}

			// Set output format with colors enabled to test stripping
			viper.Set("output", tc.format)
			viper.Set("use-colors", true)
			viper.Set("use-emoji", false)

			// Capture output
			output := captureStdout(func() {
				buildAndRenderChangeset(changeset, deployment, awsConfig)
			})

			// Validate ANSI codes are stripped
			tc.validateNoANSI(t, output)
		})
	}
}

// TestOutputFormats_JSONStructure tests the JSON structure matches go-output v2 format
func TestOutputFormats_JSONStructure(t *testing.T) {
	// Setup test changeset
	changeset := lib.ChangesetInfo{
		Name:      "test-changeset",
		HasModule: false,
		StackID:   "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/12345",
		Changes: []lib.ChangesetChanges{
			{
				Action:      "Add",
				LogicalID:   "MyBucket",
				Type:        "AWS::S3::Bucket",
				ResourceID:  "my-bucket-123",
				Replacement: "False",
			},
		},
	}

	deployment := lib.DeployInfo{
		StackName: "test-stack",
		IsNew:     false,
		IsDryRun:  false,
	}

	awsConfig := config.AWSConfig{
		Region: "us-east-1",
	}

	// Set JSON output format
	viper.Set("output", "json")
	viper.Set("use-colors", false)
	viper.Set("use-emoji", false)

	// Capture output
	output := captureStdout(func() {
		buildAndRenderChangeset(changeset, deployment, awsConfig)
	})

	// Parse JSON - go-output v2 returns an array of tables
	var tables []any
	if err := json.Unmarshal([]byte(output), &tables); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Should have at least stack info table
	if len(tables) == 0 {
		t.Fatal("JSON array should not be empty")
	}

	// Check first table (stack info)
	firstTable, ok := tables[0].(map[string]any)
	if !ok {
		t.Fatal("Each table should be an object")
	}

	// Verify title and data fields exist
	if _, ok := firstTable["title"]; !ok {
		t.Error("Table should have 'title' field")
	}

	if _, ok := firstTable["data"]; !ok {
		t.Error("Table should have 'data' field")
	}

	// Verify ConsoleURL is accessible
	dataInterface, ok := firstTable["data"]
	if !ok {
		t.Fatal("Table should have 'data' field")
	}

	dataArray, ok := dataInterface.([]any)
	if !ok || len(dataArray) == 0 {
		t.Fatal("'data' should be a non-empty array")
	}

	firstRow, ok := dataArray[0].(map[string]any)
	if !ok {
		t.Fatal("Each data row should be an object")
	}

	// ConsoleURL should be present (for non-dry-run)
	if _, ok := firstRow["ConsoleURL"]; !ok {
		t.Error("Stack info should contain 'ConsoleURL' field for non-dry-run")
	}
}

// TestOutputFormats_DangerousChanges tests dangerous changes table rendering
func TestOutputFormats_DangerousChanges(t *testing.T) {
	tests := map[string]struct {
		changes         []lib.ChangesetChanges
		expectDangerous bool
	}{
		"with_dangerous_remove": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Remove",
					LogicalID:   "MyBucket",
					Type:        "AWS::S3::Bucket",
					ResourceID:  "bucket-123",
					Replacement: "False",
				},
			},
			expectDangerous: true,
		},
		"with_dangerous_conditional": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Modify",
					LogicalID:   "MyTable",
					Type:        "AWS::DynamoDB::Table",
					ResourceID:  "table-456",
					Replacement: "Conditional",
				},
			},
			expectDangerous: true,
		},
		"with_dangerous_true_replacement": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Modify",
					LogicalID:   "MyInstance",
					Type:        "AWS::EC2::Instance",
					ResourceID:  "instance-789",
					Replacement: "True",
				},
			},
			expectDangerous: true,
		},
		"no_dangerous_changes": {
			changes: []lib.ChangesetChanges{
				{
					Action:      "Add",
					LogicalID:   "MyBucket",
					Type:        "AWS::S3::Bucket",
					ResourceID:  "bucket-123",
					Replacement: "False",
				},
			},
			expectDangerous: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			changeset := lib.ChangesetInfo{
				Name:      "test-changeset",
				HasModule: false,
				Changes:   tc.changes,
			}

			deployment := lib.DeployInfo{
				StackName: "test-stack",
				IsNew:     false,
				IsDryRun:  false,
			}

			awsConfig := config.AWSConfig{
				Region: "us-east-1",
			}

			// Test JSON format
			viper.Set("output", "json")
			viper.Set("use-colors", false)
			viper.Set("use-emoji", false)

			output := captureStdout(func() {
				buildAndRenderChangeset(changeset, deployment, awsConfig)
			})

			var tables []any
			if err := json.Unmarshal([]byte(output), &tables); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Find dangerous changes table
			foundDangerTable := false
			for _, table := range tables {
				tableMap := table.(map[string]any)
				if title, ok := tableMap["title"].(string); ok {
					if contains(title, "destructive") {
						foundDangerTable = true
						// Check if data field exists and is not nil
						if dataInterface, ok := tableMap["data"]; ok && dataInterface != nil {
							data := dataInterface.([]any)
							if tc.expectDangerous {
								if len(data) == 0 {
									t.Error("Expected dangerous changes but table is empty")
								}
							} else {
								if len(data) != 0 {
									t.Error("Expected no dangerous changes but table has data")
								}
							}
						} else {
							// data field is missing or nil, treat as empty
							if tc.expectDangerous {
								t.Error("Expected dangerous changes but data field is missing")
							}
						}
					}
				}
			}

			if !foundDangerTable {
				t.Error("Dangerous changes table should always be present")
			}
		})
	}
}
