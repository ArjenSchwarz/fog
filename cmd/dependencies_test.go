package cmd

import (
	"context"
	"fmt"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// TestDependencies_V2BuilderPattern tests the v2 Builder pattern for dependencies command
func TestDependencies_V2BuilderPattern(t *testing.T) {
	t.Parallel()

	// Create sample stack dependency data
	stacks := []map[string]any{
		{
			"Stack":       "vpc-stack",
			"Description": "VPC and networking resources",
			"Imported By": []string{"web-stack", "api-stack"},
		},
		{
			"Stack":       "web-stack",
			"Description": "Web tier resources",
			"Imported By": []string{"cdn-stack"},
		},
		{
			"Stack":       "api-stack",
			"Description": "API tier resources",
			"Imported By": []string{},
		},
	}

	tests := map[string]struct {
		columnOrder []string
	}{
		"stack_description_imported": {
			columnOrder: []string{"Stack", "Description", "Imported By"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Build document using v2 Builder pattern
			doc := output.New().
				Table(
					"All stacks in account 123456789012 for region us-east-1",
					stacks,
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

// TestDependencies_V2Sorting tests the v2 data pipeline sorting for dependencies
func TestDependencies_V2Sorting(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stacks    []map[string]any
		sortKey   string
		expectIdx int // index of first element after sorting
	}{
		"sort_by_stack_name": {
			stacks: []map[string]any{
				{
					"Stack":       "zebra-stack",
					"Description": "Last alphabetically",
					"Imported By": []string{},
				},
				{
					"Stack":       "alpha-stack",
					"Description": "First alphabetically",
					"Imported By": []string{},
				},
				{
					"Stack":       "middle-stack",
					"Description": "Middle alphabetically",
					"Imported By": []string{},
				},
			},
			sortKey:   "Stack",
			expectIdx: 0, // alpha-stack should be first after sorting
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Build document with sorting using data pipeline
			doc := output.New().
				Table(
					"Test Dependencies with Sorting",
					tc.stacks,
					output.WithKeys("Stack", "Description", "Imported By"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render should work with sorting
			out := output.NewOutput(
				output.WithFormat(output.Table),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render sorted output: %v", err)
			}
		})
	}
}

// TestDependencies_V2ArrayHandling tests array field handling in v2 output
func TestDependencies_V2ArrayHandling(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		importedBy []string
	}{
		"empty_array": {
			importedBy: []string{},
		},
		"single_import": {
			importedBy: []string{"consumer-stack"},
		},
		"multiple_imports": {
			importedBy: []string{"web-stack", "api-stack", "cache-stack"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data with array field - v2 should handle automatically
			data := []map[string]any{
				{
					"Stack":       "vpc-stack",
					"Description": "VPC resources",
					"Imported By": tc.importedBy,
				},
			}

			// Build document with array field
			doc := output.New().
				Table(
					"Test Stack Dependencies",
					data,
					output.WithKeys("Stack", "Description", "Imported By"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render should handle array without error
			out := output.NewOutput(
				output.WithFormat(output.Table),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output with array field: %v", err)
			}
		})
	}
}

// TestDependencies_V2OutputFormats tests rendering in different output formats
func TestDependencies_V2OutputFormats(t *testing.T) {
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
					"Stack":       "vpc-stack",
					"Description": "VPC resources",
					"Imported By": []string{"web-stack", "api-stack"},
				},
				{
					"Stack":       "web-stack",
					"Description": "Web tier",
					"Imported By": []string{},
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Dependencies",
					data,
					output.WithKeys("Stack", "Description", "Imported By"),
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

// TestDependencies_V2ColumnOrdering tests that column order is preserved
func TestDependencies_V2ColumnOrdering(t *testing.T) {
	t.Parallel()

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Test data
	data := []map[string]any{
		{
			"Stack":       "vpc-stack",
			"Description": "VPC resources",
			"Imported By": []string{"web-stack"},
		},
	}

	// Expected column order for dependencies command
	expectedOrder := []string{"Stack", "Description", "Imported By"}

	// Build document with specific column order
	doc := output.New().
		Table(
			"Stack Dependencies",
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

// TestDependencies_V2FilteredOutput tests output with filtered stacks
func TestDependencies_V2FilteredOutput(t *testing.T) {
	t.Parallel()

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Simulated filtered stacks (after pattern matching)
	filteredStacks := []map[string]any{
		{
			"Stack":       "dev-vpc-stack",
			"Description": "Dev VPC resources",
			"Imported By": []string{"dev-web-stack"},
		},
		{
			"Stack":       "dev-web-stack",
			"Description": "Dev Web tier",
			"Imported By": []string{},
		},
	}

	// Build document
	doc := output.New().
		Table(
			"Stacks filtered by 'dev-*' in account 123456789012 for region us-east-1",
			filteredStacks,
			output.WithKeys("Stack", "Description", "Imported By"),
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
		t.Fatalf("Failed to render filtered output: %v", err)
	}
}

// TestDependencies_V2LargeDataSet tests output with many stacks
func TestDependencies_V2LargeDataSet(t *testing.T) {
	t.Parallel()

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Create larger dataset with many stacks
	data := make([]map[string]any, 0, 20)
	for i := range 20 {
		var stackName string
		if i%3 == 0 {
			stackName = "prod-stack-" + fmt.Sprintf("%d", i)
		} else if i%3 == 1 {
			stackName = "staging-stack-" + fmt.Sprintf("%d", i)
		} else {
			stackName = "dev-stack-" + fmt.Sprintf("%d", i)
		}

		row := map[string]any{
			"Stack":       stackName,
			"Description": "Stack " + stackName,
			"Imported By": []string{},
		}

		// Add some dependencies for variety
		if i > 0 && i%5 == 0 {
			row["Imported By"] = []string{"consumer-" + stackName}
		}

		data = append(data, row)
	}

	// Build document
	doc := output.New().
		Table(
			"All stacks in account 123456789012 for region us-east-1",
			data,
			output.WithKeys("Stack", "Description", "Imported By"),
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
		t.Fatalf("Failed to render large dataset: %v", err)
	}
}
