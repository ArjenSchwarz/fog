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

// paginatingExportsMockClient supports multi-page DescribeStacks responses for exports.
// Pages are keyed by NextToken ("" for the first call).
type paginatingExportsMockClient struct {
	pages            map[string]cloudformation.DescribeStacksOutput
	ImportsByExport  map[string][]string
	ListImportsError error
}

func (m paginatingExportsMockClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	token := ""
	if params.NextToken != nil {
		token = *params.NextToken
	}
	out := m.pages[token]
	return &out, nil
}

func (m paginatingExportsMockClient) ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
	if m.ListImportsError != nil {
		return nil, m.ListImportsError
	}
	imports, ok := m.ImportsByExport[*params.ExportName]
	if !ok {
		return nil, errors.New("not found")
	}
	return &cloudformation.ListImportsOutput{Imports: imports}, nil
}

// TestGetExports_Pagination verifies that exports from stacks across multiple
// DescribeStacks pages are all collected, not just the first page.
func TestGetExports_Pagination(t *testing.T) {
	stackName := ""
	exportName := ""
	mock := paginatingExportsMockClient{
		pages: map[string]cloudformation.DescribeStacksOutput{
			"": {
				Stacks: []types.Stack{
					{
						StackName: strPtrOut("stack-page1"),
						Outputs: []types.Output{
							{OutputKey: strPtrOut("K1"), OutputValue: strPtrOut("V1"), ExportName: strPtrOut("Export1")},
						},
					},
				},
				NextToken: strPtrOut("token2"),
			},
			"token2": {
				Stacks: []types.Stack{
					{
						StackName: strPtrOut("stack-page2"),
						Outputs: []types.Output{
							{OutputKey: strPtrOut("K2"), OutputValue: strPtrOut("V2"), ExportName: strPtrOut("Export2")},
						},
					},
				},
				NextToken: strPtrOut("token3"),
			},
			"token3": {
				Stacks: []types.Stack{
					{
						StackName: strPtrOut("stack-page3"),
						Outputs: []types.Output{
							{OutputKey: strPtrOut("K3"), OutputValue: strPtrOut("V3"), ExportName: strPtrOut("Export3")},
						},
					},
				},
			},
		},
		ImportsByExport: map[string][]string{
			"Export2": {"importing-stack"},
		},
	}

	results := GetExports(&stackName, &exportName, mock)
	if len(results) != 3 {
		t.Fatalf("expected 3 exports from 3 pages, got %d", len(results))
	}

	byName := map[string]CfnOutput{}
	for _, r := range results {
		byName[r.ExportName] = r
	}

	// Verify all three pages contributed exports
	for _, name := range []string{"Export1", "Export2", "Export3"} {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing export %s from paginated results", name)
		}
	}

	// Verify import information was populated for Export2
	if !byName["Export2"].Imported || len(byName["Export2"].ImportedBy) != 1 || byName["Export2"].ImportedBy[0] != "importing-stack" {
		t.Errorf("expected Export2 imported by importing-stack: %#v", byName["Export2"])
	}

	// Verify non-imported exports are marked correctly
	if byName["Export1"].Imported {
		t.Errorf("expected Export1 not imported")
	}
}

func strPtrOut(s string) *string { return &s }
