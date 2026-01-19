package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

func TestFormatInfo(t *testing.T) {
	result := formatInfo("Test message")

	if !strings.Contains(result, "ℹ️") {
		t.Error("formatInfo should include info emoji")
	}
	if !strings.Contains(result, "Test message") {
		t.Error("formatInfo should include the message text")
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Error("formatInfo should end with double newline")
	}
}

func TestFormatSuccess(t *testing.T) {
	result := formatSuccess("Operation completed")

	if !strings.Contains(result, "✅") {
		t.Error("formatSuccess should include success emoji")
	}
	if !strings.Contains(result, "Operation completed") {
		t.Error("formatSuccess should include the message text")
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Error("formatSuccess should end with double newline")
	}
	// Check for ANSI color codes (green bold)
	if !strings.Contains(result, "\x1b[32") {
		t.Error("formatSuccess should include green color code")
	}
}

func TestFormatError(t *testing.T) {
	result := formatError("Something failed")

	if !strings.Contains(result, "🚨") {
		t.Error("formatError should include warning emoji")
	}
	// Should have emoji on both sides
	if strings.Count(result, "🚨") != 2 {
		t.Error("formatError should have warning emoji on both sides")
	}
	if !strings.Contains(result, "Something failed") {
		t.Error("formatError should include the message text")
	}
	if !strings.HasSuffix(result, "\n\n") {
		t.Error("formatError should end with double newline")
	}
	if !strings.HasPrefix(result, "\n") {
		t.Error("formatError should start with newline")
	}
	// Check for ANSI color codes (red bold)
	if !strings.Contains(result, "\x1b[31") {
		t.Error("formatError should include red color code")
	}
}

func TestFormatPositive(t *testing.T) {
	result := formatPositive("All checks passed")

	if !strings.Contains(result, "All checks passed") {
		t.Error("formatPositive should include the message text")
	}
	if !strings.HasSuffix(result, "\n") {
		t.Error("formatPositive should end with single newline")
	}
	// Check for ANSI color codes (green bold)
	if !strings.Contains(result, "\x1b[32") {
		t.Error("formatPositive should include green color code")
	}
}

func TestFormatBold(t *testing.T) {
	result := formatBold("Important text")

	if !strings.Contains(result, "Important text") {
		t.Error("formatBold should include the message text")
	}
	// Check for ANSI bold code
	if !strings.Contains(result, "\x1b[") {
		t.Error("formatBold should include ANSI formatting")
	}
}

// setupViperDefaults resets viper and sets defaults for testing
func setupViperDefaults() {
	viper.Reset()
	viper.Set("output", "table")
	viper.Set("verbose", false)
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)
	viper.Set("use-emoji", false)
	viper.Set("use-colors", false)
}

func TestRenderDocument_ConsoleOnly(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state
	setupViperDefaults()

	// Create a simple document
	data := []map[string]any{
		{"Name": "test-item", "Value": "test-value"},
	}
	doc := output.New().
		Table("Test Table", data, output.WithKeys("Name", "Value")).
		Build()

	// renderDocument should succeed with console-only output
	err := renderDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("renderDocument failed for console-only output: %v", err)
	}
}

func TestRenderDocument_WithFileMatchingFormat(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state
	setupViperDefaults()

	// Create temp directory for file output
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	viper.Set("output", "json")
	viper.Set("output-file", outputFile)
	// When output-file-format is not set or matches output, same Output handles both

	data := []map[string]any{
		{"Name": "test-item", "Value": "test-value"},
	}
	doc := output.New().
		Table("Test Table", data, output.WithKeys("Name", "Value")).
		Build()

	err := renderDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("renderDocument failed with matching file format: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}

	// Verify file contains JSON (since both formats are json)
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	if !strings.Contains(string(content), "test-item") {
		t.Error("Output file should contain the test data")
	}
}

func TestRenderDocument_WithFileDifferentFormat(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state
	setupViperDefaults()

	// Create temp directory for file output
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.md")

	viper.Set("output", "table")                // Console format
	viper.Set("output-file", outputFile)        // File path
	viper.Set("output-file-format", "markdown") // Different file format

	data := []map[string]any{
		{"Name": "test-item", "Value": "test-value"},
	}
	doc := output.New().
		Table("Test Table", data, output.WithKeys("Name", "Value")).
		Build()

	err := renderDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("renderDocument failed with different file format: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Error("Expected output file to be created")
	}

	// Verify file contains markdown format (should have pipes for tables)
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, "|") {
		t.Error("Output file should contain markdown table formatting (pipes)")
	}
	if !strings.Contains(contentStr, "test-item") {
		t.Error("Output file should contain the test data")
	}
}

func TestRenderDocument_ErrorHandling(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state
	setupViperDefaults()

	// Test with nil document - should return an error
	err := renderDocument(context.Background(), nil)
	if err == nil {
		t.Error("renderDocument should return error for nil document")
	}
}

func TestRenderDocument_MultipleFormats(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		consoleFormat string
		fileFormat    string
	}{
		"table_to_json": {
			consoleFormat: "table",
			fileFormat:    "json",
		},
		"json_to_markdown": {
			consoleFormat: "json",
			fileFormat:    "markdown",
		},
		"table_to_csv": {
			consoleFormat: "table",
			fileFormat:    "csv",
		},
		"yaml_to_table": {
			consoleFormat: "yaml",
			fileFormat:    "table",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state
			setupViperDefaults()

			tmpDir := t.TempDir()
			outputFile := filepath.Join(tmpDir, "output.txt")

			viper.Set("output", tc.consoleFormat)
			viper.Set("output-file", outputFile)
			viper.Set("output-file-format", tc.fileFormat)

			data := []map[string]any{
				{"Name": "test-item", "Value": "test-value"},
			}
			doc := output.New().
				Table("Test Table", data, output.WithKeys("Name", "Value")).
				Build()

			err := renderDocument(context.Background(), doc)
			if err != nil {
				t.Fatalf("renderDocument failed for %s to %s: %v", tc.consoleFormat, tc.fileFormat, err)
			}

			// Verify file was created
			if _, err := os.Stat(outputFile); os.IsNotExist(err) {
				t.Errorf("Expected output file to be created for %s to %s", tc.consoleFormat, tc.fileFormat)
			}

			// Verify file has content
			content, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}
			if len(content) == 0 {
				t.Error("Output file should not be empty")
			}
		})
	}
}

func TestRenderDocument_NoFileWhenNotConfigured(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state
	setupViperDefaults()

	// Explicitly ensure no output file is configured
	viper.Set("output-file", "")
	viper.Set("output-file-format", "")

	data := []map[string]any{
		{"Name": "test-item", "Value": "test-value"},
	}
	doc := output.New().
		Table("Test Table", data, output.WithKeys("Name", "Value")).
		Build()

	// Should succeed without creating any file
	err := renderDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("renderDocument failed: %v", err)
	}
}
