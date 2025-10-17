package cmd

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
)

// update flag for golden file generation
var updateGolden = flag.Bool("update", false, "update golden files")

// TestExportsOutput_GoldenFiles tests exports command output with golden files
// This establishes the v1 output baseline before migrating to v2
func TestExportsOutput_GoldenFiles(t *testing.T) {
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/exports")

	// Create sample export data
	exports := []lib.CfnOutput{
		{
			StackName:   "vpc-stack",
			OutputKey:   "VpcId",
			OutputValue: "vpc-12345678",
			ExportName:  "my-vpc-id",
			Description: "The VPC ID for the network",
			Imported:    true,
			ImportedBy:  []string{"web-stack", "api-stack"},
		},
		{
			StackName:   "vpc-stack",
			OutputKey:   "SubnetId",
			OutputValue: "subnet-87654321",
			ExportName:  "my-subnet-id",
			Description: "Public subnet ID",
			Imported:    false,
			ImportedBy:  []string{},
		},
		{
			StackName:   "database-stack",
			OutputKey:   "DBEndpoint",
			OutputValue: "db.example.com:5432",
			ExportName:  "my-db-endpoint",
			Description: "Database connection endpoint",
			Imported:    true,
			ImportedBy:  []string{"api-stack"},
		},
	}

	title := "All exports in account 123456789012 for region us-east-1"

	tests := map[string]struct {
		outputFormat string
		verbose      bool
	}{
		"table_basic": {
			outputFormat: "table",
			verbose:      false,
		},
		"table_verbose": {
			outputFormat: "table",
			verbose:      true,
		},
		"csv_basic": {
			outputFormat: "csv",
			verbose:      false,
		},
		"csv_verbose": {
			outputFormat: "csv",
			verbose:      true,
		},
		"json_basic": {
			outputFormat: "json",
			verbose:      false,
		},
		"json_verbose": {
			outputFormat: "json",
			verbose:      true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Create temp file for output
			tmpFile := filepath.Join(t.TempDir(), name+".out")

			// Setup viper with appropriate settings
			viper.Set("output", "table") // Always use table for console (not used in this test)
			viper.Set("verbose", tc.verbose)
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Build keys based on verbose flag
			keys := []string{"Export", "Description", "Stack", "Value", "Imported"}
			if tc.verbose {
				keys = append(keys, "Imported By")
			}

			// Create output using v1 API
			outputSettings := settings.NewOutputSettings()
			outputSettings.Title = title
			outputSettings.SortKey = "Export"
			outputSettings.UseEmoji = false
			outputSettings.UseColors = false
			outputSettings.OutputFile = tmpFile
			outputSettings.OutputFileFormat = tc.outputFormat

			output := format.OutputArray{
				Keys:     keys,
				Settings: outputSettings,
			}

			// Add rows
			for _, export := range exports {
				content := make(map[string]any)
				content["Export"] = export.ExportName
				content["Value"] = export.OutputValue
				content["Description"] = export.Description
				content["Stack"] = export.StackName
				if export.Imported {
					content["Imported"] = "Yes"
				} else {
					content["Imported"] = "No"
				}
				if tc.verbose {
					// Handle array output the same way the actual command does
					content["Imported By"] = strings.Join(export.ImportedBy, settings.GetSeparator())
				}
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}

			output.Write()

			// Read the generated file
			fileContent, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			// Assert against golden file
			golden.Assert(name, fileContent)
		})
	}
}

// TestExportsOutput_EmptyResults tests exports output with no results
func TestExportsOutput_EmptyResults(t *testing.T) {
	golden := testutil.NewGoldenFileWithDir(t, "testdata/golden/exports")

	// Create temp file for output
	tmpFile := filepath.Join(t.TempDir(), "empty.out")

	viper.Set("output", "table")
	viper.Set("verbose", false)
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	keys := []string{"Export", "Description", "Stack", "Value", "Imported"}
	title := "All exports in account 123456789012 for region us-east-1"

	outputSettings := settings.NewOutputSettings()
	outputSettings.Title = title
	outputSettings.SortKey = "Export"
	outputSettings.UseEmoji = false
	outputSettings.UseColors = false
	outputSettings.OutputFile = tmpFile
	outputSettings.OutputFileFormat = "table"

	output := format.OutputArray{
		Keys:     keys,
		Settings: outputSettings,
	}

	// No rows added - empty result
	output.Write()

	// Read the generated file
	fileContent, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	golden.Assert("empty_results", fileContent)
}
