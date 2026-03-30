package lib

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
			name:        "File name with extension in configured directory",
			fileName:    "testfile.yaml",
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

func TestRunPrechecksQuotedArgs(t *testing.T) {
	// Regression test for T-378: RunPrechecks must correctly parse
	// quoted arguments so that spaces inside quotes are not treated
	// as argument separators.
	t.Cleanup(viper.Reset)

	// Use "test" (the shell built-in) to verify the argument value.
	// "test X = Y" exits non-zero when X != Y, so this fails if the
	// path is split incorrectly or quotes are kept as literal characters.
	deployment := &DeployInfo{
		TemplateRelativePath: "path with spaces/template.yaml",
	}

	// Double-quoted $TEMPLATEPATH — the quotes should be stripped and
	// the path kept as a single argument. We use test to verify the
	// exact argument value.
	viper.Set("templates.prechecks", []string{`test "$TEMPLATEPATH" = "path with spaces/template.yaml"`})
	results, err := RunPrechecks(deployment)
	if err != nil {
		t.Fatalf("RunPrechecks() unexpected error: %v", err)
	}
	if deployment.PrechecksFailed {
		t.Errorf("RunPrechecks() precheck should not have failed, results: %v", results)
	}

	// Single-quoted argument with spaces — verify exact value using test.
	deployment2 := &DeployInfo{
		TemplateRelativePath: "simple/path.yaml",
	}
	viper.Set("templates.prechecks", []string{`test 'hello world' = 'hello world'`})
	results, err = RunPrechecks(deployment2)
	if err != nil {
		t.Fatalf("RunPrechecks() unexpected error: %v", err)
	}
	if deployment2.PrechecksFailed {
		t.Errorf("RunPrechecks() single-quoted precheck should not have failed, results: %v", results)
	}
}

func TestRunPrechecksUnsafeCommandWithPath(t *testing.T) {
	// Regression test for T-611: RunPrechecks must block unsafe commands
	// even when invoked via absolute or relative paths (e.g. /bin/rm, ./rm).
	// The denylist must normalise the executable name with filepath.Base
	// before checking against the blocked list.
	t.Cleanup(viper.Reset)

	deployment := &DeployInfo{
		TemplateRelativePath: "test/path.yaml",
	}

	cases := []struct {
		name    string
		command string
	}{
		{"absolute path rm", "/bin/rm --help"},
		{"absolute path del", "/usr/bin/del --help"},
		{"absolute path kill", "/bin/kill --help"},
		{"relative path rm", "./rm --help"},
		{"relative path with dir", "../bin/rm --help"},
		{"bare command rm", "rm --help"},
		{"bare command del", "del --help"},
		{"bare command kill", "kill --help"},
		{"uppercase RM", "/bin/RM --help"},
		{"uppercase KILL", "KILL --help"},
		{"mixed case Del", "./Del --help"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Set("templates.prechecks", []string{tc.command})
			results, err := RunPrechecks(deployment)
			if err == nil {
				t.Errorf("RunPrechecks(%q) should detect unsafe command, got results: %v", tc.command, results)
			} else if !strings.Contains(err.Error(), "unsafe command") {
				t.Errorf("RunPrechecks(%q) error should mention 'unsafe command', got: %v", tc.command, err)
			}
		})
	}
}

func TestRunPrechecksEmptyCommand(t *testing.T) {
	t.Cleanup(viper.Reset)

	deployment := &DeployInfo{
		TemplateRelativePath: "test/path.yaml",
	}
	viper.Set("templates.prechecks", []string{"   "})
	_, err := RunPrechecks(deployment)
	if err == nil {
		t.Errorf("RunPrechecks() should return an error for empty/whitespace precheck commands")
	}
}

func TestRunPrechecksUnbalancedQuotes(t *testing.T) {
	t.Cleanup(viper.Reset)

	deployment := &DeployInfo{
		TemplateRelativePath: "test/path.yaml",
	}
	viper.Set("templates.prechecks", []string{`echo "unterminated`})
	_, err := RunPrechecks(deployment)
	if err == nil {
		t.Errorf("RunPrechecks() should return an error for unbalanced quotes")
	}
}

func TestSplitShellArgs(t *testing.T) {
	// Regression test for T-378: splitShellArgs must handle quoted
	// arguments correctly, keeping spaces inside quotes as part of
	// the same argument and stripping the surrounding quotes.

	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "Simple command without quotes",
			input: "cfn-lint -t template.yaml",
			want:  []string{"cfn-lint", "-t", "template.yaml"},
		},
		{
			name:  "Double-quoted argument with spaces",
			input: `cfn-lint -t "path with spaces/template.yaml"`,
			want:  []string{"cfn-lint", "-t", "path with spaces/template.yaml"},
		},
		{
			name:  "Single-quoted argument with spaces",
			input: `echo 'hello world'`,
			want:  []string{"echo", "hello world"},
		},
		{
			name:  "Multiple quoted arguments",
			input: `cmd "arg one" "arg two"`,
			want:  []string{"cmd", "arg one", "arg two"},
		},
		{
			name:  "Mixed quoted and unquoted arguments",
			input: `cmd --flag "quoted arg" plain`,
			want:  []string{"cmd", "--flag", "quoted arg", "plain"},
		},
		{
			name:  "Empty quoted argument",
			input: `cmd "" arg`,
			want:  []string{"cmd", "", "arg"},
		},
		{
			name:  "Consecutive spaces between arguments",
			input: "cmd   arg1   arg2",
			want:  []string{"cmd", "arg1", "arg2"},
		},
		{
			name:  "Quoted argument containing single quotes inside double quotes",
			input: `cmd "it's a test"`,
			want:  []string{"cmd", "it's a test"},
		},
		{
			name:  "Single argument only",
			input: "cmd",
			want:  []string{"cmd"},
		},
		{
			name:  "Escaped quote inside double quotes",
			input: `cmd "arg with \"escaped\" quotes"`,
			want:  []string{"cmd", `arg with "escaped" quotes`},
		},
		{
			name:    "Unbalanced double quote",
			input:   `cmd "unterminated`,
			wantErr: true,
		},
		{
			name:    "Unbalanced single quote",
			input:   `cmd 'unterminated`,
			wantErr: true,
		},
		// T-612: Backslash-escaped spaces outside quotes should be treated
		// as literal spaces within the same argument, matching shell behaviour.
		{
			name:  "Escaped space outside quotes",
			input: `cfn-lint -t path\ with\ spaces/template.yaml`,
			want:  []string{"cfn-lint", "-t", "path with spaces/template.yaml"},
		},
		{
			name:  "Multiple escaped spaces in different arguments",
			input: `cmd first\ arg second\ arg`,
			want:  []string{"cmd", "first arg", "second arg"},
		},
		{
			name:  "Mixed escaped spaces and quoted strings",
			input: `cmd path\ one "path two" 'path three'`,
			want:  []string{"cmd", "path one", "path two", "path three"},
		},
		{
			name:  "Escaped space at start of argument",
			input: `cmd \ leading`,
			want:  []string{"cmd", " leading"},
		},
		{
			name:  "Backslash followed by non-space outside quotes",
			input: `cmd path\\file`,
			want:  []string{"cmd", `path\file`},
		},
		{
			name:  "Trailing backslash preserved",
			input: `cmd arg\`,
			want:  []string{"cmd", `arg\`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitShellArgs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitShellArgs(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitShellArgs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
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
				var gotObj, wantObj any
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
	input := map[any]any{
		"key1": "value1",
		"key2": 123,
		"key3": map[any]any{
			"nested": "value",
			"num":    456,
		},
		"key4": []any{
			"item1",
			map[any]any{"arrayItem": "value"},
		},
	}

	expected := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": map[string]any{
			"nested": "value",
			"num":    456,
		},
		"key4": []any{
			"item1",
			map[string]any{"arrayItem": "value"},
		},
	}

	result := convertMapInterfaceToMapString(input)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("convertMapInterfaceToMapString() = %v, want %v", result, expected)
	}
}
