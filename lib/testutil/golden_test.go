package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// mockT is a test double for testing.T to capture test failures
type mockT struct {
	testing.TB
	failed  bool
	errMsg  string
	logMsgs []string
	fatals  []string
}

func (m *mockT) Helper() {}

func (m *mockT) Errorf(format string, args ...any) {
	m.failed = true
	m.errMsg = format
}

func (m *mockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.fatals = append(m.fatals, format)
}

func (m *mockT) Logf(format string, args ...any) {
	m.logMsgs = append(m.logMsgs, format)
}

func TestGoldenFile_Assert_Matching(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create test golden directory: %v", err)
	}

	// Create a golden file with expected content
	goldenFile := filepath.Join(goldenDir, "test.golden")
	expectedContent := []byte("This is the expected content\nWith multiple lines\n")
	if err := os.WriteFile(goldenFile, expectedContent, 0644); err != nil {
		t.Fatalf("Failed to create test golden file: %v", err)
	}

	// Create a mock testing.T
	mockT := &mockT{}

	// Create GoldenFile instance with the test directory
	g := &GoldenFile{
		t:          mockT,
		updateFlag: false,
		dir:        goldenDir,
	}

	// Test with matching content
	g.Assert("test", expectedContent)

	// Check that no error was reported
	if mockT.failed {
		t.Errorf("Assert failed when content matched: %s", mockT.errMsg)
	}
}

func TestGoldenFile_Assert_NonMatching(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create test golden directory: %v", err)
	}

	// Create a golden file with expected content
	goldenFile := filepath.Join(goldenDir, "test.golden")
	expectedContent := []byte("This is the expected content\n")
	if err := os.WriteFile(goldenFile, expectedContent, 0644); err != nil {
		t.Fatalf("Failed to create test golden file: %v", err)
	}

	// Create a mock testing.T
	mockT := &mockT{}

	// Create GoldenFile instance with the test directory
	g := &GoldenFile{
		t:          mockT,
		updateFlag: false,
		dir:        goldenDir,
	}

	// Test with non-matching content
	actualContent := []byte("This is different content\n")
	g.Assert("test", actualContent)

	// Check that an error was reported
	if !mockT.failed {
		t.Error("Assert did not fail when content didn't match")
	}

	// Check that the error message contains the diff
	if mockT.errMsg == "" {
		t.Error("Error message was empty")
	}
}

func TestGoldenFile_Assert_MissingFile(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")

	// Create a mock testing.T
	mockT := &mockT{}

	// Create GoldenFile instance with the test directory
	g := &GoldenFile{
		t:          mockT,
		updateFlag: false,
		dir:        goldenDir,
	}

	// Test with a non-existent golden file
	g.Assert("nonexistent", []byte("Some content"))

	// Check that a fatal error was reported
	if !mockT.failed || len(mockT.fatals) == 0 {
		t.Error("Assert did not fail fatally when golden file was missing")
	}

	// Check that the error message mentions the -update flag
	if len(mockT.fatals) > 0 && !contains(mockT.fatals[0], "-update") {
		t.Errorf("Fatal error message should mention -update flag, got: %s", mockT.fatals[0])
	}
}

func TestGoldenFile_Assert_UpdateFlag(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")

	// Create a mock testing.T
	mockT := &mockT{}

	// Create GoldenFile instance with update flag set
	g := &GoldenFile{
		t:          mockT,
		updateFlag: true,
		dir:        goldenDir,
	}

	// Test with new content that should be written
	newContent := []byte("New golden file content\n")
	g.Assert("new", newContent)

	// Check that no error was reported
	if mockT.failed {
		t.Errorf("Assert failed when update flag was set: %v", mockT.fatals)
	}

	// Verify that the file was created with the correct content
	goldenFile := filepath.Join(goldenDir, "new.golden")
	writtenContent, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("Failed to read created golden file: %v", err)
	}

	if diff := cmp.Diff(string(newContent), string(writtenContent)); diff != "" {
		t.Errorf("Written content mismatch (-want +got):\n%s", diff)
	}

	// Check that a log message was recorded
	if len(mockT.logMsgs) == 0 {
		t.Error("No log message recorded when updating golden file")
	}
}

func TestGoldenFile_AssertString(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create test golden directory: %v", err)
	}

	// Create a golden file with expected content
	goldenFile := filepath.Join(goldenDir, "test.golden")
	expectedContent := "This is a string content"
	if err := os.WriteFile(goldenFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("Failed to create test golden file: %v", err)
	}

	// Create a mock testing.T
	mockT := &mockT{}

	// Create GoldenFile instance
	g := &GoldenFile{
		t:          mockT,
		updateFlag: false,
		dir:        goldenDir,
	}

	// Test AssertString method
	g.AssertString("test", expectedContent)

	// Check that no error was reported
	if mockT.failed {
		t.Errorf("AssertString failed when content matched: %s", mockT.errMsg)
	}
}

func TestGoldenFile_Path(t *testing.T) {
	g := &GoldenFile{
		dir: "testdata/golden",
	}

	tests := map[string]struct {
		name string
		want string
	}{
		"simple name": {
			name: "test",
			want: filepath.Join("testdata", "golden", "test.golden"),
		},
		"nested path": {
			name: "subdir/test",
			want: filepath.Join("testdata", "golden", "subdir", "test.golden"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := g.Path(tc.name)
			if got != tc.want {
				t.Errorf("Path() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGoldenFile_Exists(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create test golden directory: %v", err)
	}

	// Create a golden file
	existingFile := filepath.Join(goldenDir, "existing.golden")
	if err := os.WriteFile(existingFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test golden file: %v", err)
	}

	g := &GoldenFile{
		dir: goldenDir,
	}

	tests := map[string]struct {
		name string
		want bool
	}{
		"existing file": {
			name: "existing",
			want: true,
		},
		"non-existing file": {
			name: "nonexistent",
			want: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := g.Exists(tc.name)
			if got != tc.want {
				t.Errorf("Exists() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGoldenFile_Update(t *testing.T) {
	// Create a temporary directory for test golden files
	tmpDir := t.TempDir()
	goldenDir := filepath.Join(tmpDir, "golden")

	// Create a mock testing.T
	mockT := &mockT{}

	g := &GoldenFile{
		t:          mockT,
		updateFlag: false, // Update flag is false, but Update should still write
		dir:        goldenDir,
	}

	// Force update of a golden file
	content := []byte("Forced update content\n")
	g.Update("forced", content)

	// Check that no error was reported
	if mockT.failed {
		t.Errorf("Update failed: %v", mockT.fatals)
	}

	// Verify that the file was created
	goldenFile := filepath.Join(goldenDir, "forced.golden")
	writtenContent, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("Failed to read updated golden file: %v", err)
	}

	if diff := cmp.Diff(string(content), string(writtenContent)); diff != "" {
		t.Errorf("Updated content mismatch (-want +got):\n%s", diff)
	}
}

func TestNewGoldenFile(t *testing.T) {
	// Save original flag value if it exists
	var originalValue string
	if updateFlag := flag.Lookup("update"); updateFlag != nil {
		originalValue = updateFlag.Value.String()
		defer func() {
			_ = updateFlag.Value.Set(originalValue)
		}()
	}

	// Test without update flag
	g := NewGoldenFile(t)
	if g.t != t {
		t.Error("NewGoldenFile did not set testing.T correctly")
	}
	if g.updateFlag {
		t.Error("Update flag should be false by default")
	}
	expectedDir := filepath.Join("testdata", "golden")
	if g.dir != expectedDir {
		t.Errorf("Default directory = %v, want %v", g.dir, expectedDir)
	}
}

func TestNewGoldenFileWithDir(t *testing.T) {
	customDir := "custom/golden"
	g := NewGoldenFileWithDir(t, customDir)

	if g.t != t {
		t.Error("NewGoldenFileWithDir did not set testing.T correctly")
	}
	if g.dir != customDir {
		t.Errorf("Custom directory = %v, want %v", g.dir, customDir)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
