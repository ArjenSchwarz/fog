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
		StackName: strPtr("test-stack"),
		Outputs: []types.Output{
			{
				OutputKey:   strPtr("Key1"),
				OutputValue: strPtr("Val1"),
				ExportName:  strPtr("Export1"),
				Description: strPtr("desc1"),
			},
			{
				OutputKey:   strPtr("Key2"),
				OutputValue: strPtr("Val2"),
				ExportName:  nil, // not an export
			},
			{
				OutputKey:   strPtr("Key3"),
				OutputValue: strPtr("Val3"),
				ExportName:  strPtr("OtherExport"),
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
		OutputKey:   strPtr("Key1"),
		OutputValue: strPtr("Val1"),
		ExportName:  strPtr("Export1"),
	}
	export2 := types.Output{
		OutputKey:   strPtr("Key2"),
		OutputValue: strPtr("Val2"),
		ExportName:  strPtr("Export2"),
	}
	stacksOutput := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{
			{
				StackName: strPtr(stackName),
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

	results := GetExports(&stackName, strPtr(""), mock)
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

func strPtr(s string) *string { return &s }
