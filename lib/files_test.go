package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestReadFile(t *testing.T) {
	// Setup test directory and files
	tempDir := t.TempDir()

	// Create test files
	testContent := "test content"
	testFilePath := filepath.Join(tempDir, "testfile.yaml")
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup viper config for tests
	viper.Set("testtype.directory", tempDir)
	viper.Set("testtype.extensions", []string{".yaml", ".json"})

	// Test cases
	tests := []struct {
		name        string
		fileName    string
		fileType    string
		wantContent string
		wantPath    string
		wantErr     bool
	}{
		{
			name:        "Existing file with full path",
			fileName:    testFilePath,
			fileType:    "testtype",
			wantContent: testContent,
			wantPath:    testFilePath,
			wantErr:     false,
		},
		{
			name:        "File name only with extension search",
			fileName:    "testfile",
			fileType:    "testtype",
			wantContent: testContent,
			wantPath:    filepath.Join(tempDir, "testfile.yaml"),
			wantErr:     false,
		},
		{
			name:        "Non-existent file",
			fileName:    "nonexistent",
			fileType:    "testtype",
			wantContent: "",
			wantPath:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotContent, gotPath, err := ReadFile(&tt.fileName, tt.fileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotContent != tt.wantContent {
				t.Errorf("ReadFile() gotContent = %v, want %v", gotContent, tt.wantContent)
			}
			if !tt.wantErr && gotPath != tt.wantPath {
				t.Errorf("ReadFile() gotPath = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}

func TestReadTemplate(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	testContent := "template content"
	testFilePath := filepath.Join(tempDir, "template.yaml")
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	viper.Set("templates.directory", tempDir)
	viper.Set("templates.extensions", []string{".yaml", ".json"})

	// Test
	templateName := "template"
	gotContent, gotPath, err := ReadTemplate(&templateName)
	if err != nil {
		t.Errorf("ReadTemplate() error = %v", err)
		return
	}
	if gotContent != testContent {
		t.Errorf("ReadTemplate() gotContent = %v, want %v", gotContent, testContent)
	}
	if gotPath != testFilePath {
		t.Errorf("ReadTemplate() gotPath = %v, want %v", gotPath, testFilePath)
	}
}

func TestReadTagsfile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	testContent := "tags content"
	testFilePath := filepath.Join(tempDir, "tags.json")
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	viper.Set("tags.directory", tempDir)
	viper.Set("tags.extensions", []string{".json"})

	// Test
	gotContent, gotPath, err := ReadTagsfile("tags")
	if err != nil {
		t.Errorf("ReadTagsfile() error = %v", err)
		return
	}
	if gotContent != testContent {
		t.Errorf("ReadTagsfile() gotContent = %v, want %v", gotContent, testContent)
	}
	if gotPath != testFilePath {
		t.Errorf("ReadTagsfile() gotPath = %v, want %v", gotPath, testFilePath)
	}
}

func TestReadParametersfile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	testContent := "parameters content"
	testFilePath := filepath.Join(tempDir, "params.json")
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	viper.Set("parameters.directory", tempDir)
	viper.Set("parameters.extensions", []string{".json"})

	// Test
	gotContent, gotPath, err := ReadParametersfile("params")
	if err != nil {
		t.Errorf("ReadParametersfile() error = %v", err)
		return
	}
	if gotContent != testContent {
		t.Errorf("ReadParametersfile() gotContent = %v, want %v", gotContent, testContent)
	}
	if gotPath != testFilePath {
		t.Errorf("ReadParametersfile() gotPath = %v, want %v", gotPath, testFilePath)
	}
}

func TestReadDeploymentFile(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	testContent := "deployment content"
	testFilePath := filepath.Join(tempDir, "deploy.yaml")
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	viper.Set("deployments.directory", tempDir)
	viper.Set("deployments.extensions", []string{".yaml"})

	// Test
	gotContent, gotPath, err := ReadDeploymentFile("deploy")
	if err != nil {
		t.Errorf("ReadDeploymentFile() error = %v", err)
		return
	}
	if gotContent != testContent {
		t.Errorf("ReadDeploymentFile() gotContent = %v, want %v", gotContent, testContent)
	}
	if gotPath != testFilePath {
		t.Errorf("ReadDeploymentFile() gotPath = %v, want %v", gotPath, testFilePath)
	}
}

func TestRunPrechecks(t *testing.T) {
	// This is a simplified test since we can't easily mock exec.Command
	// We'll just test the unsafe command detection

	// Setup
	deployment := &DeployInfo{
		TemplateRelativePath: "test/path",
	}

	// Test unsafe command detection
	viper.Set("templates.prechecks", []string{"rm -rf /"})
	results, err := RunPrechecks(deployment)

	if err == nil {
		t.Errorf("RunPrechecks() should detect unsafe command")
	}
	if len(results) > 0 {
		t.Errorf("RunPrechecks() should not return results for unsafe command")
	}
}

func TestYamlToJson(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "Simple YAML",
			input:   "key: value\nnum: 123",
			want:    `{"key":"value","num":123}`,
			wantErr: false,
		},
		{
			name:    "Nested YAML",
			input:   "parent:\n  child: value\n  number: 42",
			want:    `{"parent":{"child":"value","number":42}}`,
			wantErr: false,
		},
		{
			name:    "Array YAML",
			input:   "items:\n  - item1\n  - item2",
			want:    `{"items":["item1","item2"]}`,
			wantErr: false,
		},
		{
			name:    "Invalid YAML",
			input:   "key: value\n- invalid: [unclosed bracket",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YamlToJson([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("YamlToJson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Compare JSON by unmarshaling to ensure equivalent structure
				var gotObj, wantObj interface{}
				if err := json.Unmarshal(got, &gotObj); err != nil {
					t.Errorf("Failed to unmarshal result: %v", err)
				}
				if err := json.Unmarshal([]byte(tt.want), &wantObj); err != nil {
					t.Errorf("Failed to unmarshal expected: %v", err)
				}
				if !reflect.DeepEqual(gotObj, wantObj) {
					t.Errorf("YamlToJson() = %v, want %v", string(got), tt.want)
				}
			}
		})
	}
}

func TestConvertMapInterfaceToMapString(t *testing.T) {
	// Test with a map[interface{}]interface{}
	input := map[interface{}]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": map[interface{}]interface{}{
			"nested": "value",
			"num":    456,
		},
		"key4": []interface{}{
			"item1",
			map[interface{}]interface{}{"arrayItem": "value"},
		},
	}

	expected := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": map[string]interface{}{
			"nested": "value",
			"num":    456,
		},
		"key4": []interface{}{
			"item1",
			map[string]interface{}{"arrayItem": "value"},
		},
	}

	result := convertMapInterfaceToMapString(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("convertMapInterfaceToMapString() = %v, want %v", result, expected)
	}
}
