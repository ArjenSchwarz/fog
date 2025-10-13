package testutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModernTestPatterns_MapBasedTables verifies that test files use map-based table tests
func TestModernTestPatterns_MapBasedTables(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile        string
		shouldUseMap    bool
		skipCheck       bool
		functionToCheck string
	}{
		"stacks_refactored_test uses map-based tables": {
			testFile:        "../stacks_refactored_test.go",
			shouldUseMap:    true,
			functionToCheck: "TestGetStack_WithDependencyInjection",
		},
		"changesets_refactored_test uses map-based tables": {
			testFile:        "../changesets_refactored_test.go",
			shouldUseMap:    true,
			functionToCheck: "TestChangesetInfo_DeleteChangesetRefactored",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.skipCheck {
				t.Skip("Skipping check for this file")
			}

			// Parse the test file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.testFile, nil, parser.ParseComments)
			require.NoError(t, err, "Failed to parse test file")

			// Find the test function
			var foundFunction bool
			var usesMapBasedTable bool

			ast.Inspect(node, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if tc.functionToCheck != "" && fn.Name.Name != tc.functionToCheck {
						return true
					}

					if strings.HasPrefix(fn.Name.Name, "Test") {
						foundFunction = true

						// Look for map[string]struct pattern in variable declarations
						ast.Inspect(fn.Body, func(n ast.Node) bool {
							if assign, ok := n.(*ast.AssignStmt); ok {
								for _, rhs := range assign.Rhs {
									if compLit, ok := rhs.(*ast.CompositeLit); ok {
										if mapType, ok := compLit.Type.(*ast.MapType); ok {
											// Check if key is string
											if ident, ok := mapType.Key.(*ast.Ident); ok && ident.Name == "string" {
												usesMapBasedTable = true
												return false
											}
										}
									}
								}
							}
							return true
						})
					}
				}
				return true
			})

			if tc.functionToCheck != "" {
				require.True(t, foundFunction, "Function %s not found in %s", tc.functionToCheck, tc.testFile)
			}

			if tc.shouldUseMap {
				assert.True(t, usesMapBasedTable, "Test file %s should use map-based table tests", tc.testFile)
			}
		})
	}
}

// TestModernTestPatterns_THelperUsage verifies that helper functions call t.Helper()
func TestModernTestPatterns_THelperUsage(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		file             string
		helperFunction   string
		shouldCallHelper bool
	}{
		"TestContext.CreateTempFile calls t.Helper": {
			file:             "helpers.go",
			helperFunction:   "CreateTempFile",
			shouldCallHelper: true,
		},
		"TestContext.ReadTempFile calls t.Helper": {
			file:             "helpers.go",
			helperFunction:   "ReadTempFile",
			shouldCallHelper: true,
		},
		"LoadFixture calls t.Helper": {
			file:             "helpers.go",
			helperFunction:   "LoadFixture",
			shouldCallHelper: true,
		},
		"LoadFixtureString calls t.Helper": {
			file:             "helpers.go",
			helperFunction:   "LoadFixtureString",
			shouldCallHelper: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Parse the file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.file, nil, parser.ParseComments)
			require.NoError(t, err, "Failed to parse file")

			// Find the helper function
			var foundFunction bool
			var callsTHelper bool

			ast.Inspect(node, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					// Check method name
					methodName := fn.Name.Name
					if fn.Recv != nil && len(fn.Recv.List) > 0 {
						// This is a method, check if it matches
						if methodName == tc.helperFunction {
							foundFunction = true

							// Check if the first statement is t.Helper()
							if fn.Body != nil && len(fn.Body.List) > 0 {
								if exprStmt, ok := fn.Body.List[0].(*ast.ExprStmt); ok {
									if call, ok := exprStmt.X.(*ast.CallExpr); ok {
										if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
											if sel.Sel.Name == "Helper" {
												callsTHelper = true
											}
										}
									}
								}
							}
						}
					} else if methodName == tc.helperFunction {
						// Regular function
						foundFunction = true

						// Check if it has a *testing.T parameter
						hasTestingParam := false
						for _, param := range fn.Type.Params.List {
							if star, ok := param.Type.(*ast.StarExpr); ok {
								if sel, ok := star.X.(*ast.SelectorExpr); ok {
									if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "testing" {
										if sel.Sel.Name == "T" {
											hasTestingParam = true
										}
									}
								}
							}
						}

						if hasTestingParam && fn.Body != nil && len(fn.Body.List) > 0 {
							// Check if the first statement is t.Helper()
							if exprStmt, ok := fn.Body.List[0].(*ast.ExprStmt); ok {
								if call, ok := exprStmt.X.(*ast.CallExpr); ok {
									if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
										if sel.Sel.Name == "Helper" {
											callsTHelper = true
										}
									}
								}
							}
						}
					}
				}
				return true
			})

			require.True(t, foundFunction, "Helper function %s not found in %s", tc.helperFunction, tc.file)

			if tc.shouldCallHelper {
				assert.True(t, callsTHelper, "Helper function %s should call t.Helper() as first statement", tc.helperFunction)
			}
		})
	}
}

// TestModernTestPatterns_GotWantNaming verifies that tests use got/want naming convention
func TestModernTestPatterns_GotWantNaming(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile         string
		functionToCheck  string
		shouldUseGotWant bool
	}{
		"stacks_refactored_test uses got/want naming": {
			testFile:         "../stacks_refactored_test.go",
			functionToCheck:  "TestGetStack_WithDependencyInjection",
			shouldUseGotWant: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Parse the test file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.testFile, nil, parser.ParseComments)
			require.NoError(t, err, "Failed to parse test file")

			// Find variables named 'got' and 'want'
			var foundGot, foundWant bool

			ast.Inspect(node, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if tc.functionToCheck != "" && fn.Name.Name != tc.functionToCheck {
						return true
					}

					if strings.HasPrefix(fn.Name.Name, "Test") {
						// Look for variable declarations with names 'got' or 'want'
						ast.Inspect(fn.Body, func(n ast.Node) bool {
							if assign, ok := n.(*ast.AssignStmt); ok {
								for _, lhs := range assign.Lhs {
									if ident, ok := lhs.(*ast.Ident); ok {
										if ident.Name == "got" {
											foundGot = true
										}
										if ident.Name == "want" {
											foundWant = true
										}
									}
								}
							}

							// Also check in struct fields
							if field, ok := n.(*ast.Field); ok {
								for _, name := range field.Names {
									if name.Name == "want" {
										foundWant = true
									}
								}
							}
							return true
						})
					}
				}
				return true
			})

			if tc.shouldUseGotWant {
				assert.True(t, foundGot, "Test should use 'got' variable naming")
				assert.True(t, foundWant, "Test should use 'want' variable naming")
			}
		})
	}
}

// TestModernTestPatterns_CmpDiffUsage verifies that tests use cmp.Diff for comparisons
func TestModernTestPatterns_CmpDiffUsage(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile         string
		functionToCheck  string
		shouldUseCmpDiff bool
	}{
		"stacks_refactored_test uses cmp.Diff": {
			testFile:         "../stacks_refactored_test.go",
			functionToCheck:  "TestGetStack_WithDependencyInjection",
			shouldUseCmpDiff: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Read the file content
			content, err := os.ReadFile(tc.testFile)
			require.NoError(t, err, "Failed to read test file")

			fileContent := string(content)

			// Check for cmp.Diff import
			hasCmpImport := strings.Contains(fileContent, `"github.com/google/go-cmp/cmp"`)
			if tc.shouldUseCmpDiff {
				assert.True(t, hasCmpImport, "Test file should import github.com/google/go-cmp/cmp")
			}

			// Check for cmp.Diff usage
			usesCmpDiff := strings.Contains(fileContent, "cmp.Diff(")
			if tc.shouldUseCmpDiff {
				assert.True(t, usesCmpDiff, "Test file should use cmp.Diff for comparisons")
			}

			// Check that reflect.DeepEqual is not used
			usesReflectDeepEqual := strings.Contains(fileContent, "reflect.DeepEqual")
			assert.False(t, usesReflectDeepEqual, "Test file should not use reflect.DeepEqual (use cmp.Diff instead)")
		})
	}
}

// TestModernTestPatterns_TRunUsage verifies that table tests use t.Run for subtests
func TestModernTestPatterns_TRunUsage(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile        string
		functionToCheck string
		shouldUseTRun   bool
	}{
		"stacks_refactored_test uses t.Run": {
			testFile:        "../stacks_refactored_test.go",
			functionToCheck: "TestGetStack_WithDependencyInjection",
			shouldUseTRun:   true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Parse the test file
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, tc.testFile, nil, parser.ParseComments)
			require.NoError(t, err, "Failed to parse test file")

			// Find t.Run calls
			var foundTRun bool

			ast.Inspect(node, func(n ast.Node) bool {
				if fn, ok := n.(*ast.FuncDecl); ok {
					if tc.functionToCheck != "" && fn.Name.Name != tc.functionToCheck {
						return true
					}

					if strings.HasPrefix(fn.Name.Name, "Test") {
						// Look for t.Run calls
						ast.Inspect(fn.Body, func(n ast.Node) bool {
							if call, ok := n.(*ast.CallExpr); ok {
								if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
									if sel.Sel.Name == "Run" {
										if ident, ok := sel.X.(*ast.Ident); ok {
											if ident.Name == "t" {
												foundTRun = true
											}
										}
									}
								}
							}
							return true
						})
					}
				}
				return true
			})

			if tc.shouldUseTRun {
				assert.True(t, foundTRun, "Test should use t.Run for subtests")
			}
		})
	}
}

// TestModernTestPatterns_TParallelUsage verifies that parallel tests call t.Parallel()
func TestModernTestPatterns_TParallelUsage(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile           string
		functionToCheck    string
		shouldUseTParallel bool
	}{
		"stacks_refactored_test subtests use t.Parallel": {
			testFile:           "../stacks_refactored_test.go",
			functionToCheck:    "TestGetStack_WithDependencyInjection",
			shouldUseTParallel: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Read the file content
			content, err := os.ReadFile(tc.testFile)
			require.NoError(t, err, "Failed to read test file")

			fileContent := string(content)

			// Check for t.Parallel() usage
			usesTParallel := strings.Contains(fileContent, "t.Parallel()")
			if tc.shouldUseTParallel {
				assert.True(t, usesTParallel, "Test file should use t.Parallel() for parallel subtests")
			}
		})
	}
}

// TestModernTestPatterns_VariableCapture verifies proper variable capture in parallel tests
func TestModernTestPatterns_VariableCapture(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile              string
		shouldCaptureVariable bool
	}{
		"stacks_refactored_test captures range variable": {
			testFile:              "../stacks_refactored_test.go",
			shouldCaptureVariable: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Read the file content
			content, err := os.ReadFile(tc.testFile)
			require.NoError(t, err, "Failed to read test file")

			fileContent := string(content)

			// Check for variable capture pattern (tc := tc or similar)
			capturesVariable := strings.Contains(fileContent, "tc := tc") ||
				strings.Contains(fileContent, "tt := tt")

			if tc.shouldCaptureVariable && strings.Contains(fileContent, "t.Parallel()") {
				assert.True(t, capturesVariable, "Test file with t.Parallel() should capture range variable")
			}
		})
	}
}

// TestModernTestPatterns_TestStructure validates overall test structure
func TestModernTestPatterns_TestStructure(t *testing.T) {
	t.Helper()

	// Test that refactored test files exist and follow naming convention
	tests := map[string]struct {
		testFile    string
		shouldExist bool
	}{
		"stacks_refactored_test.go exists": {
			testFile:    "../stacks_refactored_test.go",
			shouldExist: true,
		},
		"changesets_refactored_test.go exists": {
			testFile:    "../changesets_refactored_test.go",
			shouldExist: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := os.Stat(tc.testFile)
			if tc.shouldExist {
				require.NoError(t, err, "Test file %s should exist", tc.testFile)
			}
		})
	}
}

// TestModernTestPatterns_NoSliceBasedTables verifies that new tests don't use slice-based tables
func TestModernTestPatterns_NoSliceBasedTables(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		testFile                string
		shouldNotUseSliceTables bool
	}{
		"stacks_refactored_test should not use slice-based tables": {
			testFile:                "../stacks_refactored_test.go",
			shouldNotUseSliceTables: true,
		},
		"changesets_refactored_test should not use slice-based tables": {
			testFile:                "../changesets_refactored_test.go",
			shouldNotUseSliceTables: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Read the file content
			content, err := os.ReadFile(tc.testFile)
			if os.IsNotExist(err) {
				t.Skip("Test file does not exist yet")
			}
			require.NoError(t, err, "Failed to read test file")

			fileContent := string(content)

			// Check for slice-based table pattern ([]struct{)
			usesSliceTable := strings.Contains(fileContent, "tests := []struct{")
			if tc.shouldNotUseSliceTables {
				assert.False(t, usesSliceTable, "Test file should not use slice-based table tests (use map[string]struct instead)")
			}
		})
	}
}

// TestModernTestPatterns_AllLibTestFiles validates all lib test files
func TestModernTestPatterns_AllLibTestFiles(t *testing.T) {
	t.Helper()

	// Find all test files in lib directory
	testFiles, err := filepath.Glob("../*_test.go")
	require.NoError(t, err, "Failed to find test files")

	require.NotEmpty(t, testFiles, "Should find test files in lib directory")

	// This test serves as documentation of which files have been uplifted
	t.Logf("Found %d test files in lib directory", len(testFiles))
	for _, file := range testFiles {
		t.Logf("  - %s", filepath.Base(file))
	}
}
