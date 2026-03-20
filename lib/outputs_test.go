package lib

import (
	"context"
	"fmt"
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
		return nil, fmt.Errorf("Export '%s' is not imported by any stack.", *params.ExportName)
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

// TestCfnOutput_FillImports checks success and "not imported" cases.
func TestCfnOutput_FillImports(t *testing.T) {
	out := &CfnOutput{ExportName: "Export1"}
	mock := MockCFNClient{ImportsByExport: map[string][]string{"Export1": {"stackA"}}}

	err := out.FillImports(mock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !out.Imported || len(out.ImportedBy) != 1 || out.ImportedBy[0] != "stackA" {
		t.Errorf("FillImports success case failed: %#v", out)
	}

	// "not imported" error should set Imported=false without returning an error
	out2 := &CfnOutput{ExportName: "Export1"}
	mockNotImported := MockCFNClient{
		ListImportsError: fmt.Errorf("Export 'Export1' is not imported by any stack."),
	}

	err = out2.FillImports(mockNotImported)
	if err != nil {
		t.Errorf("unexpected error for not-imported case: %v", err)
	}
	if out2.Imported {
		t.Errorf("expected Imported=false for not-imported error")
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

	results, err := GetExports(&stackName, strPtrOut(""), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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
		return nil, fmt.Errorf("Export '%s' is not imported by any stack.", *params.ExportName)
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

	results, err := GetExports(&stackName, strPtrOut(""), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

// TestFillImports_NotImportedError verifies that the specific "is not imported
// by any stack" error sets Imported=false without returning an error.
func TestFillImports_NotImportedError(t *testing.T) {
	out := &CfnOutput{ExportName: "MyExport"}
	notImportedErr := fmt.Errorf("Export 'MyExport' is not imported by any stack.")
	mock := MockCFNClient{ListImportsError: notImportedErr}

	err := out.FillImports(mock)
	if err != nil {
		t.Errorf("expected no error for 'not imported' message, got: %v", err)
	}
	if out.Imported {
		t.Errorf("expected Imported=false for 'not imported' error")
	}
}

// TestFillImports_PropagatesRealErrors verifies that non-"not imported" errors
// (e.g., throttling, permissions) are returned to the caller instead of being
// silently treated as "not imported".
func TestFillImports_PropagatesRealErrors(t *testing.T) {
	tests := map[string]error{
		"throttling":    fmt.Errorf("Rate exceeded"),
		"access denied": fmt.Errorf("Access Denied"),
		"generic":       fmt.Errorf("something went wrong"),
	}
	for name, testErr := range tests {
		t.Run(name, func(t *testing.T) {
			out := &CfnOutput{ExportName: "Export1"}
			mock := MockCFNClient{ListImportsError: testErr}

			err := out.FillImports(mock)
			if err == nil {
				t.Errorf("expected error to be propagated for %q, got nil", name)
			}
		})
	}
}

// TestGetExports_PropagatesListImportsError verifies that GetExports returns an
// error when ListImports fails with a non-"not imported" error instead of
// silently setting Imported=false.
func TestGetExports_PropagatesListImportsError(t *testing.T) {
	stackName := "test-stack"
	stacksOutput := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName: strPtrOut(stackName),
				Outputs: []types.Output{
					{
						OutputKey:   strPtrOut("Key1"),
						OutputValue: strPtrOut("Val1"),
						ExportName:  strPtrOut("Export1"),
					},
				},
			},
		},
	}
	mock := MockCFNClient{
		DescribeStacksOutput: stacksOutput,
		ListImportsError:     fmt.Errorf("Rate exceeded"),
	}

	_, err := GetExports(&stackName, strPtrOut(""), mock)
	if err == nil {
		t.Error("expected GetExports to return an error when ListImports fails with a real error")
	}
}

// TestGetExports_NotImportedErrorSetsImportedFalse verifies that GetExports
// handles the specific "not imported" error correctly by setting Imported=false
// without returning an error.
func TestGetExports_NotImportedErrorSetsImportedFalse(t *testing.T) {
	stackName := "test-stack"
	stacksOutput := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName: strPtrOut(stackName),
				Outputs: []types.Output{
					{
						OutputKey:   strPtrOut("Key1"),
						OutputValue: strPtrOut("Val1"),
						ExportName:  strPtrOut("Export1"),
					},
				},
			},
		},
	}

	// perExportMockCFNClient returns the "not imported" error for specific exports
	mock := perExportMockCFNClient{
		DescribeStacksOutput: stacksOutput,
		errorByExport: map[string]error{
			"Export1": fmt.Errorf("Export 'Export1' is not imported by any stack."),
		},
	}

	results, err := GetExports(&stackName, strPtrOut(""), mock)
	if err != nil {
		t.Errorf("expected no error for 'not imported' case, got: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Imported {
		t.Errorf("expected Imported=false for export with 'not imported' error")
	}
}

// perExportMockCFNClient allows different ListImports errors per export name.
type perExportMockCFNClient struct {
	DescribeStacksOutput cloudformation.DescribeStacksOutput
	errorByExport        map[string]error
	importsByExport      map[string][]string
}

func (m perExportMockCFNClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return &m.DescribeStacksOutput, nil
}

func (m perExportMockCFNClient) ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
	if err, ok := m.errorByExport[*params.ExportName]; ok {
		return nil, err
	}
	if imports, ok := m.importsByExport[*params.ExportName]; ok {
		return &cloudformation.ListImportsOutput{Imports: imports}, nil
	}
	return nil, fmt.Errorf("Export '%s' is not imported by any stack.", *params.ExportName)
}

func strPtrOut(s string) *string { return &s }
