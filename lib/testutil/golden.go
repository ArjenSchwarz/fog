package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// GoldenFile manages golden file testing for validating complex output
type GoldenFile struct {
	t          testing.TB
	updateFlag bool
	dir        string
}

// NewGoldenFile creates a new GoldenFile instance with automatic update flag detection
func NewGoldenFile(t *testing.T) *GoldenFile {
	t.Helper()

	// Check if update flag is set
	var updateFlag bool
	if flag.Lookup("update") != nil {
		updateFlag = flag.Lookup("update").Value.String() == "true"
	}

	return &GoldenFile{
		t:          t,
		updateFlag: updateFlag,
		dir:        filepath.Join("testdata", "golden"),
	}
}

// NewGoldenFileWithDir creates a new GoldenFile instance with a custom directory
func NewGoldenFileWithDir(t *testing.T, dir string) *GoldenFile {
	t.Helper()

	// Check if update flag is set
	var updateFlag bool
	if flag.Lookup("update") != nil {
		updateFlag = flag.Lookup("update").Value.String() == "true"
	}

	return &GoldenFile{
		t:          t,
		updateFlag: updateFlag,
		dir:        dir,
	}
}

// Assert compares actual content with golden file content
// If update flag is set, it updates the golden file instead
func (g *GoldenFile) Assert(name string, actual []byte) {
	g.t.Helper()

	goldenPath := filepath.Join(g.dir, name+".golden")

	// If update flag is set, write the actual content to the golden file
	if g.updateFlag {
		// Create directory if it doesn't exist
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			g.t.Fatalf("Failed to create golden file directory: %v", err)
		}

		if err := os.WriteFile(goldenPath, actual, 0644); err != nil {
			g.t.Fatalf("Failed to update golden file %s: %v", goldenPath, err)
		}
		g.t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read the expected content from the golden file
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			g.t.Fatalf("Golden file %s does not exist. Run with -update flag to create it.", goldenPath)
		}
		g.t.Fatalf("Failed to read golden file %s: %v", goldenPath, err)
	}

	// Compare actual with expected
	if diff := cmp.Diff(string(expected), string(actual)); diff != "" {
		g.t.Errorf("Golden file mismatch for %s (-want +got):\n%s", name, diff)
	}
}

// AssertBytes is an alias for Assert for backward compatibility
func (g *GoldenFile) AssertBytes(name string, actual []byte) {
	g.t.Helper()
	g.Assert(name, actual)
}

// AssertString converts string to bytes and calls Assert
func (g *GoldenFile) AssertString(name string, actual string) {
	g.t.Helper()
	g.Assert(name, []byte(actual))
}

// Path returns the full path to a golden file
func (g *GoldenFile) Path(name string) string {
	return filepath.Join(g.dir, name+".golden")
}

// Exists checks if a golden file exists
func (g *GoldenFile) Exists(name string) bool {
	_, err := os.Stat(g.Path(name))
	return err == nil
}

// Update forces an update of the golden file regardless of the update flag
func (g *GoldenFile) Update(name string, content []byte) {
	g.t.Helper()

	goldenPath := g.Path(name)

	// Create directory if it doesn't exist
	dir := filepath.Dir(goldenPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		g.t.Fatalf("Failed to create golden file directory: %v", err)
	}

	if err := os.WriteFile(goldenPath, content, 0644); err != nil {
		g.t.Fatalf("Failed to update golden file %s: %v", goldenPath, err)
	}
}

// StripAnsi removes ANSI escape codes from a string
// This is useful for testing content without color/formatting codes
func StripAnsi(s string) string {
	// Regex pattern for ANSI escape codes
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

// AssertStringWithoutAnsi strips ANSI codes from actual content before comparing with golden file
// This is useful for testing content structure while allowing colored output in production
func (g *GoldenFile) AssertStringWithoutAnsi(name string, actual string) {
	g.t.Helper()
	stripped := StripAnsi(actual)
	g.Assert(name, []byte(stripped))
}
