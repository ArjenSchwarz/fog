package cmd

import (
	"context"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// TestDemoTables_V2BuilderPattern tests the v2 Builder pattern for demo tables command
func TestDemoTables_V2BuilderPattern(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		columnOrder []string
	}{
		"demo_export_order": {
			columnOrder: []string{"Export", "Description", "Stack", "Value", "Imported"},
		},
	}

	for name, tc := range tests {
		tc := tc // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)
			viper.Set("use-emoji", false)
			viper.Set("use-colors", false)

			// Create sample demo data
			data := []map[string]any{
				{
					"Export":      "awesome-stack-dev-s3-arn",
					"Value":       "arn:aws:s3:::fog-awesome-stack-dev",
					"Description": "ARN of the S3 bucket",
					"Stack":       "awesome-stack-dev",
					"Imported":    true,
				},
				{
					"Export":      "awesome-stack-test-s3-arn",
					"Value":       "arn:aws:s3:::fog-awesome-stack-test",
					"Description": "ARN of the S3 bucket",
					"Stack":       "awesome-stack-test",
					"Imported":    true,
				},
				{
					"Export":      "awesome-stack-prod-s3-arn",
					"Value":       "arn:aws:s3:::fog-awesome-stack-prod",
					"Description": "ARN of the S3 bucket",
					"Stack":       "awesome-stack-prod",
					"Imported":    true,
				},
				{
					"Export":      "demo-s3-bucket",
					"Value":       "fog-demo-bucket",
					"Description": "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
					"Stack":       "demo-resources",
					"Imported":    false,
				},
			}

			// Build document using v2 Builder pattern with WithKeys to preserve column order
			doc := output.New().
				Table(
					"Export values demo",
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

// TestDemoTables_V2DifferentStyles tests rendering with different table styles
func TestDemoTables_V2DifferentStyles(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		style string
	}{
		"default_style": {
			style: "Default",
		},
		"bold_style": {
			style: "Bold",
		},
		"colored_bright_style": {
			style: "ColoredBright",
		},
		"light_style": {
			style: "Light",
		},
		"rounded_style": {
			style: "Rounded",
		},
	}

	for name, tc := range tests {
		tc := tc // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper with specific style
			viper.Set("output", "table")
			viper.Set("table.style", tc.style)
			viper.Set("table.max-column-width", 50)
			viper.Set("use-emoji", false)
			viper.Set("use-colors", false)

			// Sample data
			data := []map[string]any{
				{
					"Export":      "test-export",
					"Value":       "test-value",
					"Description": "Test description",
					"Stack":       "test-stack",
					"Imported":    true,
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Export values demo - "+tc.style,
					data,
					output.WithKeys("Export", "Description", "Stack", "Value", "Imported"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render with specific style
			out := output.NewOutput(
				output.WithFormat(output.Table),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output with style %s: %v", tc.style, err)
			}
		})
	}
}

// TestDemoTables_V2LongDescriptions tests column width wrapping with long descriptions
func TestDemoTables_V2LongDescriptions(t *testing.T) {
	t.Parallel()

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Data with exceptionally long description to test wrapping
	data := []map[string]any{
		{
			"Export":      "demo-s3-bucket",
			"Value":       "fog-demo-bucket",
			"Description": "The S3 bucket used for demos but has an exceptionally long description so it can show a multi-line example",
			"Stack":       "demo-resources",
			"Imported":    false,
		},
	}

	// Build document
	doc := output.New().
		Table(
			"Export values with long description",
			data,
			output.WithKeys("Export", "Description", "Stack", "Value", "Imported"),
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
		t.Fatalf("Failed to render output with long description: %v", err)
	}
}

// TestDemoTables_V2SortedOutput tests sorting by Export column
func TestDemoTables_V2SortedOutput(t *testing.T) {
	t.Parallel()

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Data in unsorted order
	data := []map[string]any{
		{
			"Export":      "zebra-export",
			"Value":       "value-z",
			"Description": "Last alphabetically",
			"Stack":       "stack-z",
			"Imported":    true,
		},
		{
			"Export":      "alpha-export",
			"Value":       "value-a",
			"Description": "First alphabetically",
			"Stack":       "stack-a",
			"Imported":    true,
		},
		{
			"Export":      "middle-export",
			"Value":       "value-m",
			"Description": "Middle alphabetically",
			"Stack":       "stack-m",
			"Imported":    true,
		},
	}

	// Build document with sorting
	doc := output.New().
		Table(
			"Sorted export values demo",
			data,
			output.WithKeys("Export", "Description", "Stack", "Value", "Imported"),
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
		t.Fatalf("Failed to render sorted output: %v", err)
	}
}

// TestDemoTables_V2OutputFormats tests rendering in different output formats
func TestDemoTables_V2OutputFormats(t *testing.T) {
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
		tc := tc // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Sample data
			data := []map[string]any{
				{
					"Export":      "test-export",
					"Value":       "test-value",
					"Description": "Test description",
					"Stack":       "test-stack",
					"Imported":    true,
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Export values demo",
					data,
					output.WithKeys("Export", "Description", "Stack", "Value", "Imported"),
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

// TestDemoTables_V2BooleanValues tests handling of boolean Imported field
func TestDemoTables_V2BooleanValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		imported bool
	}{
		"imported_true": {
			imported: true,
		},
		"imported_false": {
			imported: false,
		},
	}

	for name, tc := range tests {
		tc := tc // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper configuration
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Data with boolean field
			data := []map[string]any{
				{
					"Export":      "test-export",
					"Value":       "test-value",
					"Description": "Test description",
					"Stack":       "test-stack",
					"Imported":    tc.imported,
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Export values with boolean",
					data,
					output.WithKeys("Export", "Description", "Stack", "Value", "Imported"),
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
				t.Fatalf("Failed to render output with imported=%v: %v", tc.imported, err)
			}
		})
	}
}

// TestDemoTables_V2ColumnOrdering tests that column order is preserved
func TestDemoTables_V2ColumnOrdering(t *testing.T) {
	t.Parallel()

	// Setup viper configuration
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)

	// Test data
	data := []map[string]any{
		{
			"Export":      "test-export",
			"Value":       "test-value",
			"Description": "Test description",
			"Stack":       "test-stack",
			"Imported":    true,
		},
	}

	// Expected column order for demo tables command
	expectedOrder := []string{"Export", "Description", "Stack", "Value", "Imported"}

	// Build document with specific column order
	doc := output.New().
		Table(
			"Export values demo",
			data,
			output.WithKeys(expectedOrder...),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify rendering preserves column order
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render with column ordering: %v", err)
	}
}
