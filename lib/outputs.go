package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
)

// CfnOutput represents a CloudFormation stack output value
type CfnOutput struct {
	StackName   string
	OutputKey   string
	OutputValue string
	Description string
	ExportName  string
	Imported    bool
	ImportedBy  []string
}

// importResult carries the result of a concurrent ListImports call along with
// the source index, so results can be placed back in DescribeStacks order.
type importResult struct {
	index  int
	output CfnOutput
	err    error
}

// GetExports returns all the exports in the account and region. If stackname
// is provided, results will be limited to that stack. Each export will also
// be checked whether it is being imported or not.
// Returns an error if any ListImports call fails with a non-"not imported" error
// (e.g., throttling, permission denied).
func GetExports(ctx context.Context, stackname *string, exportname *string, svc CFNExportsAPI) ([]CfnOutput, error) {
	exports := []CfnOutput{}
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	paginator := cloudformation.NewDescribeStacksPaginator(svc, input)
	allstacks := make([]types.Stack, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing stacks: %w", err)
		}
		allstacks = append(allstacks, output.Stacks...)
	}
	for _, stack := range allstacks {
		exports = append(exports, getOutputsForStack(stack, *stackname, *exportname, true)...)
	}
	c := make(chan importResult)
	results := make([]CfnOutput, len(exports))
	for i, export := range exports {
		go func(index int, export CfnOutput) {
			resexport := CfnOutput{
				StackName:   export.StackName,
				OutputKey:   export.OutputKey,
				OutputValue: export.OutputValue,
				ExportName:  export.ExportName,
				Description: export.Description,
			}
			paginator := cloudformation.NewListImportsPaginator(svc, &cloudformation.ListImportsInput{ExportName: &export.ExportName})
			var allImports []string
			var paginationErr error
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(ctx)
				if err != nil {
					paginationErr = err
					break
				}
				allImports = append(allImports, page.Imports...)
			}
			if paginationErr != nil {
				if isNotImportedError(paginationErr) {
					resexport.Imported = false
					c <- importResult{index: index, output: resexport}
				} else {
					c <- importResult{index: index, output: resexport, err: fmt.Errorf("ListImports for %q: %w", export.ExportName, paginationErr)}
				}
				return
			}
			resexport.Imported = true
			resexport.ImportedBy = allImports
			c <- importResult{index: index, output: resexport}
		}(i, export)
	}
	var errs []error
	for range results {
		r := <-c
		results[r.index] = r.output
		if r.err != nil {
			errs = append(errs, r.err)
		}
	}
	if len(errs) > 0 {
		return results, errors.Join(errs...)
	}
	return results, nil
}

func getOutputsForStack(stack types.Stack, stackfilter string, exportfilter string, exportsOnly bool) []CfnOutput {
	result := []CfnOutput{}
	stackName := aws.ToString(stack.StackName)
	if stackName == "" {
		return result
	}
	if strings.Contains(stackfilter, "*") {
		if !GlobToRegex(stackfilter).MatchString(stackName) {
			return result
		}
	}
	for _, output := range stack.Outputs {
		if output.OutputKey == nil || output.OutputValue == nil {
			continue
		}
		exportName := aws.ToString(output.ExportName)
		if exportsOnly && exportName == "" {
			continue
		}
		if exportfilter != "" {
			if !GlobToRegex(exportfilter).MatchString(exportName) {
				continue
			}
		}
		parsedOutput := CfnOutput{
			StackName:   stackName,
			OutputKey:   *output.OutputKey,
			OutputValue: *output.OutputValue,
			ExportName:  exportName,
		}
		if output.Description != nil {
			parsedOutput.Description = *output.Description
		}
		result = append(result, parsedOutput)
	}
	return result
}

// FillImports populates the import information for an exported output.
// Returns an error if ListImports fails with anything other than the expected
// "is not imported by any stack" message (e.g., throttling, permission errors).
func (output *CfnOutput) FillImports(ctx context.Context, svc CFNListImportsAPI) error {
	if output.ExportName == "" {
		return nil
	}
	paginator := cloudformation.NewListImportsPaginator(svc, &cloudformation.ListImportsInput{ExportName: &output.ExportName})
	var allImports []string
	var paginationErr error
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			paginationErr = err
			break
		}
		allImports = append(allImports, page.Imports...)
	}
	if paginationErr != nil {
		if isNotImportedError(paginationErr) {
			output.Imported = false
			output.ImportedBy = nil
			return nil
		}
		return fmt.Errorf("ListImports for %q: %w", output.ExportName, paginationErr)
	}
	output.Imported = true
	output.ImportedBy = allImports
	return nil
}

// isNotImportedError returns true when the error is the specific CloudFormation
// message indicating an export has no importers: "Export '...' is not imported
// by any stack." All other errors (throttling, permissions, etc.) return false.
//
// Checks for a smithy API error first (the typed representation from the AWS SDK),
// then falls back to string matching on the error message.
func isNotImportedError(err error) bool {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return strings.Contains(apiErr.ErrorMessage(), "is not imported by any stack")
	}
	return strings.Contains(err.Error(), "is not imported by any stack")
}
