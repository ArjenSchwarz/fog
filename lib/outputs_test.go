package lib

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// Interfaces for the cloudformation operations used
type CFNDescribeStacksAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type CFNListImportsAPI interface {
	ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error)
}

type CFNExportsAPI interface {
	CFNDescribeStacksAPI
	CFNListImportsAPI
}

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

// Wrapper for GetExports using interfaces for testing
func GetExportsTest(stackname *string, exportname *string, svc CFNExportsAPI) []CfnOutput {
	exports := []CfnOutput{}
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	resp, err := svc.DescribeStacks(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	for _, stack := range resp.Stacks {
		exports = append(exports, getOutputsForStack(stack, *stackname, *exportname, true)...)
	}
	c := make(chan CfnOutput)
	results := make([]CfnOutput, len(exports))
	for _, export := range exports {
		go func(export CfnOutput) {
			resexport := CfnOutput{
				StackName:   export.StackName,
				OutputKey:   export.OutputKey,
				OutputValue: export.OutputValue,
				ExportName:  export.ExportName,
				Description: export.Description,
			}
			imports, err := svc.ListImports(context.TODO(), &cloudformation.ListImportsInput{ExportName: &export.ExportName})
			if err != nil {
				resexport.Imported = false
			} else {
				resexport.Imported = true
				resexport.ImportedBy = imports.Imports
			}
			c <- resexport
		}(export)
	}
	for i := 0; i < len(results); i++ {
		results[i] = <-c
	}
	return results
}

// Wrapper for FillImports using interface
func (output *CfnOutput) FillImportsTest(svc CFNListImportsAPI) {
	if output.ExportName == "" {
		return
	}
	imports, err := svc.ListImports(context.TODO(), &cloudformation.ListImportsInput{ExportName: &output.ExportName})
	if err != nil {
		output.Imported = false
	} else {
		output.Imported = true
		output.ImportedBy = imports.Imports
	}
}

// Test getOutputsForStack filtering and parsing
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

// Test FillImports logic with and without error
func TestCfnOutput_FillImports(t *testing.T) {
	out := &CfnOutput{ExportName: "Export1"}
	mock := MockCFNClient{ImportsByExport: map[string][]string{"Export1": {"stackA"}}}

	out.FillImportsTest(mock)
	if !out.Imported || len(out.ImportedBy) != 1 || out.ImportedBy[0] != "stackA" {
		t.Errorf("FillImports success case failed: %#v", out)
	}

	out2 := &CfnOutput{ExportName: "Export1"}
	mockErr := MockCFNClient{ListImportsError: errors.New("fail")}

	out2.FillImportsTest(mockErr)
	if out2.Imported {
		t.Errorf("expected Imported=false on error")
	}
}

// Test GetExports wrapper combining DescribeStacks and ListImports logic
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

	results := GetExportsTest(&stackName, strPtr(""), mock)
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
