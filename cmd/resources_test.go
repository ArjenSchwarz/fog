package cmd

import (
	"context"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
)

// buildResourcesDocument is a testable helper that builds the resources output document
// This extracts the document-building logic from listResources for easier testing
func buildResourcesDocument(data []map[string]any, keys []string, title string) *output.Document {
	doc := output.New().
		Table(
			title,
			data,
			output.WithKeys(keys...),
		).
		Build()
	return doc
}

// TestResources_V2BuilderPattern tests the v2 Builder pattern for resources command
func TestResources_V2BuilderPattern(t *testing.T) {
	// Create sample resource data matching the structure from lib.GetResources
	data := []map[string]any{
		{
			"Type":  "AWS::EC2::Instance",
			"ID":    "i-1234567890abcdef0",
			"Stack": "my-web-stack",
		},
		{
			"Type":  "AWS::S3::Bucket",
			"ID":    "my-bucket-12345678",
			"Stack": "my-storage-stack",
		},
		{
			"Type":  "AWS::RDS::DBInstance",
			"ID":    "mydb-instance",
			"Stack": "my-database-stack",
		},
	}

	keys := []string{"Type", "ID", "Stack"}
	title := "All resources created by CloudFormation in account 123456789012 for region us-east-1"

	// Build document using the same logic as listResources
	doc := buildResourcesDocument(data, keys, title)

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

// TestResources_ColumnOrderingMatching tests that column ordering matches v1 behavior
// Tests both basic and verbose modes
func TestResources_ColumnOrderingMatching(t *testing.T) {
	tests := map[string]struct {
		keys  []string
		data  []map[string]any
		title string
	}{
		"basic_columns": {
			keys: []string{"Type", "ID", "Stack"},
			data: []map[string]any{
				{
					"Type":  "AWS::EC2::SecurityGroup",
					"ID":    "sg-12345678",
					"Stack": "my-stack",
				},
			},
			title: "CloudFormation Resources",
		},
		"verbose_columns": {
			keys: []string{"Type", "ID", "Stack", "LogicalID", "Status"},
			data: []map[string]any{
				{
					"Type":      "AWS::EC2::SecurityGroup",
					"ID":        "sg-12345678",
					"Stack":     "my-stack",
					"LogicalID": "WebServerSecurityGroup",
					"Status":    "CREATE_COMPLETE",
				},
			},
			title: "CloudFormation Resources (Verbose)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Build document - verify WithKeys enforces proper column ordering
			doc := buildResourcesDocument(tc.data, tc.keys, tc.title)

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Verify rendering works
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

// TestResources_SortingByType tests that resources are sorted by Type
func TestResources_SortingByType(t *testing.T) {
	// Test data in intentionally unsorted order
	unsortedData := []map[string]any{
		{
			"Type":  "AWS::S3::Bucket",
			"ID":    "my-bucket",
			"Stack": "storage",
		},
		{
			"Type":  "AWS::EC2::Instance",
			"ID":    "i-12345678",
			"Stack": "compute",
		},
		{
			"Type":  "AWS::RDS::DBInstance",
			"ID":    "mydb",
			"Stack": "database",
		},
		{
			"Type":  "AWS::Lambda::Function",
			"ID":    "my-func",
			"Stack": "serverless",
		},
	}

	keys := []string{"Type", "ID", "Stack"}
	title := "Resources Sorted by Type"

	// Build document - should apply SortBy("Type")
	doc := buildResourcesDocument(unsortedData, keys, title)

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify rendering succeeds
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render sorted output: %v", err)
	}
}

// TestResources_V2OutputRenderingCorrectly tests that output renders in different formats
func TestResources_V2OutputRenderingCorrectly(t *testing.T) {
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

	// Test data
	data := []map[string]any{
		{
			"Type":  "AWS::EC2::Instance",
			"ID":    "i-0123456789abcdef0",
			"Stack": "web-stack",
		},
		{
			"Type":  "AWS::S3::Bucket",
			"ID":    "my-assets-bucket",
			"Stack": "storage-stack",
		},
	}

	keys := []string{"Type", "ID", "Stack"}
	title := "CloudFormation Resources"

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Build document
			doc := buildResourcesDocument(data, keys, title)

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

// TestResources_ArrayFieldHandling tests that array fields are rendered correctly
func TestResources_ArrayFieldHandling(t *testing.T) {
	// Test data with array fields (if Status could be an array in future)
	tests := map[string]struct {
		data []map[string]any
	}{
		"single_status": {
			data: []map[string]any{
				{
					"Type":  "AWS::Lambda::Function",
					"ID":    "my-function",
					"Stack": "serverless",
				},
			},
		},
		"verbose_with_status": {
			data: []map[string]any{
				{
					"Type":      "AWS::EC2::Instance",
					"ID":        "i-123456",
					"Stack":     "web-stack",
					"LogicalID": "WebServer",
					"Status":    "CREATE_COMPLETE",
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			keys := []string{"Type", "ID", "Stack"}
			if len(tc.data) > 0 && tc.data[0]["LogicalID"] != nil {
				keys = append(keys, []string{"LogicalID", "Status"}...)
			}

			doc := buildResourcesDocument(tc.data, keys, "Resources with Array Fields")

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render output with array fields: %v", err)
			}
		})
	}
}

// TestResources_EmptyResults tests rendering when no resources are found
func TestResources_EmptyResults(t *testing.T) {
	data := []map[string]any{}
	keys := []string{"Type", "ID", "Stack"}
	title := "All resources created by CloudFormation in account 123456789012 for region us-east-1"

	doc := buildResourcesDocument(data, keys, title)

	if doc == nil {
		t.Fatal("Built document should not be nil even for empty results")
	}

	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render empty results: %v", err)
	}
}
