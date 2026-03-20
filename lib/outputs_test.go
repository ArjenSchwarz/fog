package lib

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Mock client implementing the interfaces
type MockCFNClient struct {
	DescribeStacksOutput cloudformation.DescribeStacksOutput
	DescribeStacksError  error

	// map of export name to list of importing stack names
	ImportsByExport  map[string][]string
	ListImportsError error
}

func (m MockCFNClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return &m.DescribeStacksOutput, m.DescribeStacksError
}

func (m MockCFNClient) ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
	if m.ListImportsError != nil {
		return nil, m.ListImportsError
	}
	imports, ok := m.ImportsByExport[*params.ExportName]
	if !ok {
		return nil, errors.New("not found")
	}
	return &cloudformation.ListImportsOutput{Imports: imports}, nil
}

// Test_getOutputsForStack verifies export filtering and parsing logic.
func Test_getOutputsForStack(t *testing.T) {
	stack := types.Stack{
		StackName: strPtrOut("test-stack"),
		Outputs: []types.Output{
			{
				OutputKey:   strPtrOut("Key1"),
				OutputValue: strPtrOut("Val1"),
				ExportName:  strPtrOut("Export1"),
				Description: strPtrOut("desc1"),
			},
			{
				OutputKey:   strPtrOut("Key2"),
				OutputValue: strPtrOut("Val2"),
				ExportName:  nil, // not an export
			},
			{
				OutputKey:   strPtrOut("Key3"),
				OutputValue: strPtrOut("Val3"),
				ExportName:  strPtrOut("OtherExport"),
			},
		},
	}

	res := getOutputsForStack(stack, "test-stack", "", true)
	if len(res) != 2 {
		t.Fatalf("expected 2 exports, got %d", len(res))
	}
	if res[0].Description != "desc1" {
		t.Errorf("expected description copied")
	}

	// export filter
	res2 := getOutputsForStack(stack, "test-stack", "Export1", true)
	if len(res2) != 1 || res2[0].ExportName != "Export1" {
		t.Errorf("export filter not applied: %v", res2)
	}

	// stack filter not match
	res3 := getOutputsForStack(stack, "other*", "", true)
	if len(res3) != 0 {
		t.Errorf("expected no results for unmatched stack filter")
	}
}

// Test_getOutputsForStack_regexMetacharacters verifies that regex metacharacters
// in export and stack filters are treated as literal characters, not regex operators.
// Regression test for T-511.
func Test_getOutputsForStack_regexMetacharacters(t *testing.T) {
	stack := types.Stack{
		StackName: strPtrOut("my.stack.name"),
		Outputs: []types.Output{
			{
				OutputKey:   strPtrOut("Key1"),
				OutputValue: strPtrOut("Val1"),
				ExportName:  strPtrOut("my.export.name"),
			},
			{
				OutputKey:   strPtrOut("Key2"),
				OutputValue: strPtrOut("Val2"),
				ExportName:  strPtrOut("myXexportXname"),
			},
		},
	}

	// Export filter with dot should match only the literal dot, not any character.
	// Before the fix, "my.export.name" would also match "myXexportXname" because
	// '.' in regex means "any character".
	res := getOutputsForStack(stack, "my.stack.name", "my.export.name", true)
	if len(res) != 1 {
		t.Fatalf("expected 1 export matching literal dot filter, got %d", len(res))
	}
	if res[0].ExportName != "my.export.name" {
		t.Errorf("expected 'my.export.name', got %q", res[0].ExportName)
	}

	// Stack filter with dot should match only the literal dot.
	// A stack named "myXstackXname" should not match filter "my.stack.name".
	stackOther := types.Stack{
		StackName: strPtrOut("myXstackXname"),
		Outputs: []types.Output{
			{
				OutputKey:   strPtrOut("Key1"),
				OutputValue: strPtrOut("Val1"),
				ExportName:  strPtrOut("Export1"),
			},
		},
	}
	res2 := getOutputsForStack(stackOther, "my.stack.*", "", true)
	if len(res2) != 0 {
		t.Errorf("expected 0 results for stack 'myXstackXname' with filter 'my.stack.*', got %d", len(res2))
	}

	// Verify wildcard still works with metacharacters present.
	res3 := getOutputsForStack(stack, "my.stack.*", "my.export.*", true)
	if len(res3) != 1 {
		t.Fatalf("expected 1 export with wildcard+dot filter, got %d", len(res3))
	}
	if res3[0].ExportName != "my.export.name" {
		t.Errorf("expected 'my.export.name', got %q", res3[0].ExportName)
	}

	// Filter with other metacharacters like '+' should be literal.
	stackPlus := types.Stack{
		StackName: strPtrOut("test-stack"),
		Outputs: []types.Output{
			{
				OutputKey:   strPtrOut("Key1"),
				OutputValue: strPtrOut("Val1"),
				ExportName:  strPtrOut("foo+bar"),
			},
			{
				OutputKey:   strPtrOut("Key2"),
				OutputValue: strPtrOut("Val2"),
				ExportName:  strPtrOut("foobar"),
			},
		},
	}
	res4 := getOutputsForStack(stackPlus, "test-stack", "foo+bar", true)
	if len(res4) != 1 {
		t.Fatalf("expected 1 export matching literal '+' filter, got %d", len(res4))
	}
	if res4[0].ExportName != "foo+bar" {
		t.Errorf("expected 'foo+bar', got %q", res4[0].ExportName)
	}
}

// TestCfnOutput_FillImports checks success and error cases when populating import information.
func TestCfnOutput_FillImports(t *testing.T) {
	out := &CfnOutput{ExportName: "Export1"}
	mock := MockCFNClient{ImportsByExport: map[string][]string{"Export1": {"stackA"}}}

	out.FillImports(mock)
	if !out.Imported || len(out.ImportedBy) != 1 || out.ImportedBy[0] != "stackA" {
		t.Errorf("FillImports success case failed: %#v", out)
	}

	out2 := &CfnOutput{ExportName: "Export1"}
	mockErr := MockCFNClient{ListImportsError: errors.New("fail")}

	out2.FillImports(mockErr)
	if out2.Imported {
		t.Errorf("expected Imported=false on error")
	}
}

// TestGetExports validates that exports are returned with import information populated.
func TestGetExports(t *testing.T) {
	stackName := "test-stack"
	export1 := types.Output{
		OutputKey:   strPtrOut("Key1"),
		OutputValue: strPtrOut("Val1"),
		ExportName:  strPtrOut("Export1"),
	}
	export2 := types.Output{
		OutputKey:   strPtrOut("Key2"),
		OutputValue: strPtrOut("Val2"),
		ExportName:  strPtrOut("Export2"),
	}
	stacksOutput := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName: strPtrOut(stackName),
				Outputs:   []types.Output{export1, export2},
			},
		},
	}
	mock := MockCFNClient{
		DescribeStacksOutput: stacksOutput,
		ImportsByExport: map[string][]string{
			"Export1": {"stackA"},
		},
	}

	results := GetExports(&stackName, strPtrOut(""), mock)
	if len(results) != 2 {
		t.Fatalf("expected two results, got %d", len(results))
	}

	var byName = map[string]CfnOutput{}
	for _, r := range results {
		byName[r.ExportName] = r
	}

	if !byName["Export1"].Imported || byName["Export1"].ImportedBy[0] != "stackA" {
		t.Errorf("expected Export1 imported by stackA: %#v", byName["Export1"])
	}
	if byName["Export2"].Imported {
		t.Errorf("expected Export2 not imported")
	}
}

// paginatingMockCFNClient supports multi-page DescribeStacks responses for exports.
// Pages are keyed by NextToken ("" for the first call).
type paginatingMockCFNClient struct {
	pages            map[string]cloudformation.DescribeStacksOutput
	ImportsByExport  map[string][]string
	ListImportsError error
}

func (m paginatingMockCFNClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	token := ""
	if params.NextToken != nil {
		token = *params.NextToken
	}
	out := m.pages[token]
	return &out, nil
}

func (m paginatingMockCFNClient) ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
	if m.ListImportsError != nil {
		return nil, m.ListImportsError
	}
	imports, ok := m.ImportsByExport[*params.ExportName]
	if !ok {
		return nil, errors.New("not found")
	}
	return &cloudformation.ListImportsOutput{Imports: imports}, nil
}

// TestGetExportsPagination verifies that exports from stacks across multiple
// DescribeStacks pages are all returned.
func TestGetExportsPagination(t *testing.T) {
	stackName := ""
	mock := paginatingMockCFNClient{
		pages: map[string]cloudformation.DescribeStacksOutput{
			"": {
				Stacks: []types.Stack{{
					StackName: strPtrOut("stack-page1"),
					Outputs: []types.Output{
						{OutputKey: strPtrOut("K1"), OutputValue: strPtrOut("V1"), ExportName: strPtrOut("Export1")},
					},
				}},
				NextToken: strPtrOut("token2"),
			},
			"token2": {
				Stacks: []types.Stack{{
					StackName: strPtrOut("stack-page2"),
					Outputs: []types.Output{
						{OutputKey: strPtrOut("K2"), OutputValue: strPtrOut("V2"), ExportName: strPtrOut("Export2")},
					},
				}},
			},
		},
		// Neither export is imported; ListImports will return "not found"
		ImportsByExport: map[string][]string{},
	}

	results := GetExports(&stackName, strPtrOut(""), mock)
	if len(results) != 2 {
		t.Fatalf("expected 2 exports from 2 pages, got %d", len(results))
	}

	byName := map[string]CfnOutput{}
	for _, r := range results {
		byName[r.ExportName] = r
	}
	if _, ok := byName["Export1"]; !ok {
		t.Error("missing Export1 from first page")
	}
	if _, ok := byName["Export2"]; !ok {
		t.Error("missing Export2 from second page")
	}
}

func strPtrOut(s string) *string { return &s }
