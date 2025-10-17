package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
)

// TestReportBuilderPatternMultipleTables tests v2 Builder pattern with multiple tables
func TestReportBuilderPatternMultipleTables(t *testing.T) {
	t.Parallel()

	// Build a report document with multiple tables using v2 Builder pattern
	doc := output.New()

	// Add header
	doc.Header("Fog Deployment Report - Test Stack")

	// Add metadata table
	metadataData := []map[string]any{
		{
			"Stack":      "test-stack",
			"Account":    "123456789012 (test-account)",
			"Region":     "us-east-1",
			"Type":       "Create",
			"Start time": "2025-01-01T10:00:00Z",
			"Duration":   "45s",
			"Success":    true,
		},
	}
	doc.Table(
		"Stack Metadata",
		metadataData,
		output.WithKeys("Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"),
	)

	// Add events table
	eventsData := []map[string]any{
		{
			"Action":     "Add",
			"CfnName":    "MyBucket",
			"Type":       "AWS::S3::Bucket",
			"ID":         "my-test-bucket",
			"Start time": "2025-01-01T10:00:05Z",
			"Duration":   "10s",
			"Success":    true,
		},
		{
			"Action":     "Add",
			"CfnName":    "MyRole",
			"Type":       "AWS::IAM::Role",
			"ID":         "MyRole-ABC123",
			"Start time": "2025-01-01T10:00:15Z",
			"Duration":   "5s",
			"Success":    true,
		},
	}
	doc.Table(
		"Event Details",
		eventsData,
		output.WithKeys("Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success"),
	)

	// Build final document
	builtDoc := doc.Build()

	if builtDoc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Verify we can render it
	out := output.NewOutput(
		output.WithFormat(output.Table),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), builtDoc)
	if err != nil {
		t.Fatalf("Failed to render multi-table report: %v", err)
	}
}

// TestReportTableColumnOrdering tests that column ordering matches v1 behavior
func TestReportTableColumnOrdering(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		title           string
		data            []map[string]any
		expectedColumns []string
	}{
		"metadata_columns": {
			title: "Metadata",
			data: []map[string]any{
				{
					"Stack":      "test",
					"Account":    "123456789012",
					"Region":     "us-east-1",
					"Type":       "Create",
					"Start time": "2025-01-01T10:00:00Z",
					"Duration":   "30s",
					"Success":    true,
				},
			},
			expectedColumns: []string{"Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"},
		},
		"events_columns": {
			title: "Events",
			data: []map[string]any{
				{
					"Action":     "Add",
					"CfnName":    "MyBucket",
					"Type":       "AWS::S3::Bucket",
					"ID":         "bucket-1",
					"Start time": "2025-01-01T10:00:05Z",
					"Duration":   "10s",
					"Success":    true,
				},
			},
			expectedColumns: []string{"Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success"},
		},
	}

	for name, tc := range tests {
		tc := tc // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Build document with table
			doc := output.New().
				Table(tc.title, tc.data, output.WithKeys(tc.expectedColumns...)).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Verify rendering works with column ordering
			out := output.NewOutput(
				output.WithFormat(output.Table),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render table with column ordering: %v", err)
			}
		})
	}
}

// TestReportOutputFormats tests rendering in multiple output formats
func TestReportOutputFormats(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		format output.Format
	}{
		"table": {
			format: output.Table,
		},
		"csv": {
			format: output.CSV,
		},
		"json": {
			format: output.JSON,
		},
		"markdown": {
			format: output.Markdown,
		},
	}

	for name, tc := range tests {
		tc := tc // capture loop variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create report data
			metadataData := []map[string]any{
				{
					"Stack":      "test-stack",
					"Account":    "123456789012",
					"Region":     "us-east-1",
					"Type":       "Create",
					"Start time": "2025-01-01T10:00:00Z",
					"Duration":   "30s",
					"Success":    true,
				},
			}

			// Build document
			doc := output.New().
				Header("Fog Report").
				Table(
					"Metadata",
					metadataData,
					output.WithKeys("Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render in specific format
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

// TestReportHelperFunctions tests the table data builder helper functions
func TestReportHelperFunctions(t *testing.T) {
	t.Parallel()

	// Create test data
	awsConfig := config.AWSConfig{
		AccountID: "123456789012",
		Region:    "us-east-1",
	}

	stack := lib.CfnStack{
		Name: "test-stack",
	}

	now := time.Now()
	event := lib.StackEvent{
		Type:      "Create",
		Success:   true,
		StartDate: now,
		EndDate:   now.Add(30 * time.Second),
	}

	// Test createMetadataTable
	title, data := createMetadataTable(stack, event, awsConfig)

	if title == "" {
		t.Fatal("Metadata table title should not be empty")
	}

	if len(data) != 1 {
		t.Fatalf("Expected 1 data row, got %d", len(data))
	}

	row := data[0]
	expectedFields := []string{"Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"}
	for _, field := range expectedFields {
		if _, ok := row[field]; !ok {
			t.Fatalf("Expected field '%s' not found in metadata row", field)
		}
	}

	// Verify specific values
	if row["Stack"] != "test-stack" {
		t.Errorf("Expected Stack=test-stack, got %v", row["Stack"])
	}
	if row["Region"] != "us-east-1" {
		t.Errorf("Expected Region=us-east-1, got %v", row["Region"])
	}
	if row["Type"] != "Create" {
		t.Errorf("Expected Type=Create, got %v", row["Type"])
	}
	if row["Success"] != true {
		t.Errorf("Expected Success=true, got %v", row["Success"])
	}
}

// TestReportMermaidTableGeneration tests Mermaid diagram data generation
func TestReportMermaidTableGeneration(t *testing.T) {
	t.Parallel()

	stack := lib.CfnStack{
		Name: "test-stack",
	}

	now := time.Now()
	milestones := map[time.Time]string{
		now:                      "CREATE_IN_PROGRESS",
		now.Add(5 * time.Second): "CREATE_COMPLETE",
	}

	event := lib.StackEvent{
		Type:       "Create",
		Success:    true,
		StartDate:  now,
		EndDate:    now.Add(10 * time.Second),
		Milestones: milestones,
	}

	// Test createMermaidTable
	title, data := createMermaidTable(stack, event)

	if title == "" {
		t.Fatal("Mermaid table title should not be empty")
	}

	// Should have milestones in the data
	if len(data) < len(milestones) {
		t.Fatalf("Expected at least %d rows for milestones, got %d", len(milestones), len(data))
	}

	// Verify columns
	expectedColumns := []string{"Start time", "Duration", "Label"}
	for _, row := range data {
		for _, col := range expectedColumns {
			if _, ok := row[col]; !ok {
				t.Fatalf("Missing column '%s' in mermaid row", col)
			}
		}
	}
}

// TestReportEventDataBuilding tests event table data building
func TestReportEventDataBuilding(t *testing.T) {
	t.Parallel()

	stack := lib.CfnStack{
		Name: "test-stack",
	}

	now := time.Now()
	resourceEvent := lib.ResourceEvent{
		EventType: "Add",
		Resource: lib.CfnResource{
			LogicalID:  "MyBucket",
			Type:       "AWS::S3::Bucket",
			ResourceID: "my-bucket",
		},
		StartDate:         now,
		EndDate:           now.Add(10 * time.Second),
		EndStatus:         "CREATE_COMPLETE",
		ExpectedEndStatus: "CREATE_COMPLETE",
	}

	// Test with successful event
	successEvent := lib.StackEvent{
		Type:           "Create",
		Success:        true,
		StartDate:      now,
		EndDate:        now.Add(30 * time.Second),
		ResourceEvents: []lib.ResourceEvent{resourceEvent},
	}

	// Test with failed event
	failedEvent := lib.StackEvent{
		Type:           "Update",
		Success:        false,
		StartDate:      now,
		EndDate:        now.Add(30 * time.Second),
		ResourceEvents: []lib.ResourceEvent{resourceEvent},
	}

	// Test createEventsTable with successful event
	title, keys, data := createEventsTable(stack, successEvent)

	if title == "" {
		t.Fatal("Events table title should not be empty")
	}

	if len(data) != 1 {
		t.Fatalf("Expected 1 data row, got %d", len(data))
	}

	// For successful events, Reason column should not be in keys
	expectedKeys := []string{"Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success"}
	if len(keys) != len(expectedKeys) {
		t.Errorf("Expected %d keys for successful event, got %d", len(expectedKeys), len(keys))
	}

	// Test createEventsTable with failed event
	title2, keys2, data2 := createEventsTable(stack, failedEvent)

	if title2 == "" {
		t.Fatal("Events table title for failed event should not be empty")
	}

	// For failed events, Reason column should be in keys
	expectedKeys2 := []string{"Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success", "Reason"}
	if len(keys2) != len(expectedKeys2) {
		t.Errorf("Expected %d keys for failed event, got %d", len(expectedKeys2), len(keys2))
	}

	// Verify Reason column is included for failed events
	if len(data2) > 0 {
		row := data2[0]
		if _, ok := row["Reason"]; !ok {
			t.Error("Expected Reason field in data for failed event")
		}
	}
}
