package cmd

import (
	"context"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// TestExports_V2BuilderPattern tests the v2 Builder pattern for exports command
func TestExports_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Create sample export data
	exports := []map[string]any{
		{
			"Export": "my-vpc-id",
			"Value":  "vpc-12345678",
		},
		{
			"Export": "my-subnet-id",
			"Value":  "subnet-87654321",
		},
	}

	tests := map[string]struct {
		columnOrder []string
	}{
		"export_value_order": {
			columnOrder: []string{"Export", "Value"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("verbose", false)
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)
			viper.Set("use-emoji", false)
			viper.Set("use-colors", false)

			// Build document using v2 Builder pattern with WithKeys to preserve column order
			doc := output.New().
				Table(
					"All exports in account 123456789012 for region us-east-1",
					exports,
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

// TestExports_V2ArrayHandling tests array field handling in v2 output
func TestExports_V2ArrayHandling(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		importedBy []string
	}{
		"empty_array": {
			importedBy: []string{},
		},
		"single_item": {
			importedBy: []string{"consumer-stack"},
		},
		"multiple_items": {
			importedBy: []string{"web-stack", "api-stack", "cache-stack"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper
			viper.Set("output", "table")
			viper.Set("verbose", true)
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)
			viper.Set("use-emoji", false)
			viper.Set("use-colors", false)

			// Test data with array field
			data := []map[string]any{
				{
					"Export":      "test-export",
					"Imported By": tc.importedBy,
				},
			}

			// Build document with array field - v2 should handle array automatically
			doc := output.New().
				Table(
					"Test Export Arrays",
					data,
					output.WithKeys("Export", "Imported By"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render should handle array without error
			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output with array field: %v", err)
			}
		})
	}
}

// TestExports_V2OutputFormats tests that output renders correctly in different formats
func TestExports_V2OutputFormats(t *testing.T) {
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
			viper.Set("verbose", false)
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data
			data := []map[string]any{
				{
					"Export": "my-export",
					"Value":  "exported-value",
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Test Exports",
					data,
					output.WithKeys("Export", "Value"),
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

// TestExports_V2VerboseMode tests exports with verbose flag for ImportedBy column
func TestExports_V2VerboseMode(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper for verbose mode
	viper.Set("output", "table")
	viper.Set("verbose", true)
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Test data with ImportedBy field
	data := []map[string]any{
		{
			"Export":      "my-vpc-id",
			"Value":       "vpc-12345678",
			"Imported By": []string{"web-stack", "api-stack"},
		},
		{
			"Export":      "my-subnet-id",
			"Value":       "subnet-87654321",
			"Imported By": []string{},
		},
	}

	// Build with verbose column ordering (Export, Value, Imported By)
	doc := output.New().
		Table(
			"Exports with Imported By",
			data,
			output.WithKeys("Export", "Value", "Imported By"),
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
		t.Fatalf("Failed to render verbose output: %v", err)
	}
}
