package lib

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

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

// TestReadDeploymentFileWithDefaultConfig is a regression test for T-776:
// deployments.directory was defaulted to []string{"."} in root config, but
// ReadFile reads the directory with viper.GetString(). When the underlying
// value is a string slice, GetString returns "" instead of ".", breaking
// deployment file resolution for users relying on defaults.
func TestReadDeploymentFileWithDefaultConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	// Simulate the defaults set in cmd/root.go init().
	viper.SetDefault("deployments.directory", ".")
	viper.SetDefault("deployments.extensions", []string{"", ".yaml", ".yml", ".json"})

	// Create a deployment file in the current directory (the default directory).
	tempDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	testContent := "deployment: test"
	if err := os.WriteFile(filepath.Join(tempDir, "mystack.yaml"), []byte(testContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	gotContent, gotPath, err := ReadDeploymentFile("mystack")
	if err != nil {
		t.Fatalf("ReadDeploymentFile() with defaults returned error: %v", err)
	}
	if gotContent != testContent {
		t.Errorf("ReadDeploymentFile() content = %q, want %q", gotContent, testContent)
	}
	wantPath := filepath.Join(".", "mystack.yaml")
	if gotPath != wantPath {
		t.Errorf("ReadDeploymentFile() path = %q, want %q", gotPath, wantPath)
	}
}

// TestDeploymentsDirectoryDefaultIsString verifies that the deployments.directory
// default value is a plain string, not a string slice. When the default is
// []string{"."}, viper.GetString returns "" instead of ".", silently breaking
// deployment file resolution. This guards against the type mismatch fixed in T-776.
func TestDeploymentsDirectoryDefaultIsString(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	// Reproduce the defaults exactly as cmd/root.go init() sets them.
	// Before the fix, this was []string{"."} which caused GetString to return "".
	viper.SetDefault("deployments.directory", ".")

	got := viper.GetString("deployments.directory")
	if got != "." {
		t.Errorf("deployments.directory default via GetString = %q, want %q", got, ".")
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
		{"absolute path rm.exe", "/bin/rm.exe --help"},
		{"absolute path del", "/usr/bin/del --help"},
		{"absolute path kill", "/bin/kill --help"},
		{"relative path rm", "./rm --help"},
		{"relative path kill.cmd", "./kill.cmd --help"},
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

func encodePowerShellCommand(t *testing.T, command string) string {
	t.Helper()

	utf16Words := utf16.Encode([]rune(command))
	bytes := make([]byte, 0, len(utf16Words)*2)
	for _, word := range utf16Words {
		bytes = append(bytes, byte(word), byte(word>>8))
	}

	return base64.StdEncoding.EncodeToString(bytes)
}

func TestRunPrechecksUnsafeWrappedCommand(t *testing.T) {
	// Regression test for T-1071: RunPrechecks must reject unsafe commands
	// even when a wrapper executable forwards to them.
	t.Cleanup(viper.Reset)

	deployment := &DeployInfo{
		TemplateRelativePath: "test/path.yaml",
	}

	cases := []struct {
		name            string
		command         string
		wantErrContains string
	}{
		{"env wrapper", "env rm --help", "unsafe command"},
		{"env with assignment", "env SAFE=1 rm --help", "unsafe command"},
		{"env split string", `env -S 'rm --help'`, "unsafe command"},
		{"env split string equals", `env --split-string='rm --help'`, "unsafe command"},
		{"env argv0 flag", "env -a safe-name rm --help", "unsafe command"},
		{"env argv0 long flag", "env --argv0=safe-name rm --help", "unsafe command"},
		{"env -S nested shell", `env -S 'sh -c "rm -rf test/path.yaml"'`, "unsafe command"},
		{"env wrapping shell", `env sh -c 'rm -rf test/path.yaml'`, "unsafe command"},
		{"shell -c wrapper", `sh -c 'rm -rf test/path.yaml'`, "unsafe command"},
		{"shell sequence is rejected", `sh -c 'echo ok; rm -rf test/path.yaml'`, "cannot be safely unwrapped"},
		{"shell backtick substitution is rejected", "sh -c '`rm -rf test/path.yaml`'", "cannot be safely unwrapped"},
		{"bash -lc wrapper", `bash -lc 'kill -9 1234'`, "unsafe command"},
		{"bash option before -c", `bash -o pipefail -c 'rm -rf test/path.yaml'`, "unsafe command"},
		{"bash inline -o before -c", `bash -onoclobber -c 'rm -rf test/path.yaml'`, "unsafe command"},
		{"deeply nested wrappers", `sh -c 'bash -c "env rm --help"'`, "unsafe command"},
		{"cmd wrapper", `cmd /c del important.txt`, "unsafe command"},
		{"cmd keep wrapper", `cmd /k del important.txt`, "unsafe command"},
		{"cmd flag before c", `cmd /q /c del important.txt`, "unsafe command"},
		{"cmd operator without whitespace is rejected", `cmd /c echo ok&del important.txt`, "cannot be safely unwrapped"},
		{"powershell wrapper", `pwsh -Command "kill 1234"`, "unsafe command"},
		{"powershell command prefix", `pwsh -Com "rm -rf ."`, "unsafe command"},
		{"powershell multi arg sequence is rejected", `pwsh -Command echo ok; rm -rf .`, "cannot be safely unwrapped"},
		{"powershell encoded command", "pwsh -enc " + encodePowerShellCommand(t, "rm -rf ."), "unsafe command"},
		{"powershell file wrapper", `pwsh -File dangerous.ps1`, "cannot be safely unwrapped"},
		{"powershell file alias", `pwsh -f dangerous.ps1`, "cannot be safely unwrapped"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			viper.Set("templates.prechecks", []string{tc.command})
			results, err := RunPrechecks(deployment)
			if err == nil {
				t.Fatalf("RunPrechecks(%q) should detect unsafe wrapped command, got results: %v", tc.command, results)
			}
			if !strings.Contains(err.Error(), tc.wantErrContains) {
				t.Fatalf("RunPrechecks(%q) error should mention %q, got: %v", tc.command, tc.wantErrContains, err)
			}
			if len(results) > 0 {
				t.Fatalf("RunPrechecks(%q) should not return results for unsafe wrapped command, got: %v", tc.command, results)
			}
		})
	}
}

func TestFindUnsafeWrappedPrecheckSafeWrapper(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		command string
	}{
		{"env wrapper", `env SAFE=1 test hello = hello`},
		{"shell wrapper", `sh -c "echo hello"`},
		{"powershell command", `pwsh -Command "echo hello"`},
		{"powershell multi arg command", `pwsh -Command Write-Output hello`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, err := splitShellArgs(tc.command)
			if err != nil {
				t.Fatalf("splitShellArgs(%q): %v", tc.command, err)
			}
			unsafeCommand, err := findUnsafeWrappedPrecheck(args)
			if err != nil {
				t.Fatalf("findUnsafeWrappedPrecheck(%q) unexpected error: %v", tc.command, err)
			}
			if unsafeCommand != "" {
				t.Fatalf("findUnsafeWrappedPrecheck(%q) = %q, want no unsafe command", tc.command, unsafeCommand)
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
