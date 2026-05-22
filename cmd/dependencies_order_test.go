package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// renderDependencyTableForOrderTest builds the dependencies table from the given
// rows and returns the rendered table output as a string. The unique name avoids
// collisions with other test helpers in package cmd.
func renderDependencyTableForOrderTest(t *testing.T, rows []map[string]any) string {
	t.Helper()
	doc := output.New().
		Table(
			"Dependency Ordering",
			rows,
			output.WithKeys(colStack, colDescription, colImportedBy),
		).
		Build()

	var buf bytes.Buffer
	writer := output.NewStdoutWriter()
	writer.SetWriter(&buf)
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(writer),
	)
	if err := out.Render(context.Background(), doc); err != nil {
		t.Fatalf("failed to render dependency table: %v", err)
	}
	return buf.String()
}

// TestBuildDependencyRows_DeterministicOrder verifies the regression for T-1199:
// rows are ordered by stack name and each row's "Imported By" list is sorted,
// regardless of Go map-iteration order.
func TestBuildDependencyRows_DeterministicOrder(t *testing.T) {
	stacks := map[string]lib.CfnStack{
		"zebra-stack":  {Name: "zebra-stack", Description: "Z", ImportedBy: []string{"web-stack", "api-stack", "cdn-stack"}},
		"alpha-stack":  {Name: "alpha-stack", Description: "A", ImportedBy: []string{"queue-stack", "auth-stack"}},
		"middle-stack": {Name: "middle-stack", Description: "M", ImportedBy: []string{}},
	}

	wantNames := []string{"alpha-stack", "middle-stack", "zebra-stack"}
	wantImports := map[string][]string{
		"alpha-stack":  {"auth-stack", "queue-stack"},
		"middle-stack": {},
		"zebra-stack":  {"api-stack", "cdn-stack", "web-stack"},
	}

	// Run many times: map-iteration order varies between iterations, so a stable
	// result here proves the ordering no longer depends on map iteration.
	for i := 0; i < 50; i++ {
		rows := buildDependencyRows(stacks, nil, false)
		if len(rows) != len(wantNames) {
			t.Fatalf("iteration %d: expected %d rows, got %d", i, len(wantNames), len(rows))
		}
		for idx, row := range rows {
			gotName, _ := row[colStack].(string)
			if gotName != wantNames[idx] {
				t.Fatalf("iteration %d: row %d expected stack %q, got %q", i, idx, wantNames[idx], gotName)
			}
			gotImports, _ := row[colImportedBy].([]string)
			want := wantImports[gotName]
			if len(gotImports) != len(want) {
				t.Fatalf("iteration %d: stack %q expected %v, got %v", i, gotName, want, gotImports)
			}
			for j := range want {
				if gotImports[j] != want[j] {
					t.Fatalf("iteration %d: stack %q expected %v, got %v", i, gotName, want, gotImports)
				}
			}
		}
	}
}

// TestBuildDependencyRows_RenderedOutputStable verifies the full rendered table
// output is byte-identical across runs, covering ordering through the renderer.
func TestBuildDependencyRows_RenderedOutputStable(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	stacks := map[string]lib.CfnStack{
		"vpc-stack": {Name: "vpc-stack", Description: "VPC", ImportedBy: []string{"web-stack", "api-stack"}},
		"web-stack": {Name: "web-stack", Description: "Web", ImportedBy: []string{"cdn-stack"}},
		"api-stack": {Name: "api-stack", Description: "API", ImportedBy: []string{}},
		"db-stack":  {Name: "db-stack", Description: "DB", ImportedBy: []string{"api-stack", "web-stack"}},
	}

	first := renderDependencyTableForOrderTest(t, buildDependencyRows(stacks, nil, false))
	for i := 0; i < 25; i++ {
		got := renderDependencyTableForOrderTest(t, buildDependencyRows(stacks, nil, false))
		if got != first {
			t.Fatalf("iteration %d: rendered output differs from first run\nfirst:\n%s\ngot:\n%s", i, first, got)
		}
	}
}
