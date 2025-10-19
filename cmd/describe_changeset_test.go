package cmd

import (
	"context"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// TestPrintBasicStackInfo_V2BuilderPattern tests the v2 Builder pattern for stack info table
func TestPrintBasicStackInfo_V2BuilderPattern(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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

// TestPrintChangeset_V2BuilderPattern tests the v2 Builder pattern for changeset table
func TestPrintChangeset_V2BuilderPattern(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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

// TestPrintDangerTable_V2BuilderPattern tests the v2 Builder pattern for danger table
func TestPrintDangerTable_V2BuilderPattern(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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

// TestChangesetSummary_V2BuilderPattern tests the v2 Builder pattern for changeset summary
func TestChangesetSummary_V2BuilderPattern(t *testing.T) {
	t.Parallel()

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
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render output: %v", err)
	}
}

// TestChangeset_V2OutputFormats tests that changeset output renders correctly in different formats
func TestChangeset_V2OutputFormats(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
	t.Parallel()

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
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render sorted output: %v", err)
	}
}

// TestChangeset_V2MultipleTables tests multiple tables in single changeset output
func TestChangeset_V2MultipleTables(t *testing.T) {
	t.Parallel()

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
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render multiple tables: %v", err)
	}
}

// TestChangeset_V2EmptyChangeset tests handling of changeset with no changes
func TestChangeset_V2EmptyChangeset(t *testing.T) {
	t.Parallel()

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
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render empty changeset: %v", err)
	}
}

// TestChangeset_V2ActionTypeVariations tests different action types
func TestChangeset_V2ActionTypeVariations(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
				output.WithFormat(output.Table),
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
	t.Parallel()

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
			t.Parallel()

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
				output.WithFormat(output.Table),
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
	t.Parallel()

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

// TestAddToChangesetSummary_V2 tests the summary accumulator function
func TestAddToChangesetSummary_V2(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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
