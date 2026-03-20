package lib

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
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

// importResult carries the result of a concurrent ListImports call.
type importResult struct {
	output CfnOutput
	err    error
}

// GetExports returns all the exports in the account and region. If stackname
// is provided, results will be limited to that stack. Each export will also
// be checked whether it is being imported or not.
// Returns an error if any ListImports call fails with a non-"not imported" error
// (e.g., throttling, permission denied).
func GetExports(stackname *string, exportname *string, svc CFNExportsAPI) ([]CfnOutput, error) {
	exports := []CfnOutput{}
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	paginator := cloudformation.NewDescribeStacksPaginator(svc, input)
	allstacks := make([]types.Stack, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			var bne *smithy.OperationError
			if errors.As(err, &bne) {
				log.Fatalln("error:", bne.Err)
			}
			log.Fatalln(err)
		}
		allstacks = append(allstacks, output.Stacks...)
	}
	for _, stack := range allstacks {
		exports = append(exports, getOutputsForStack(stack, *stackname, *exportname, true)...)
	}
	c := make(chan importResult)
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
			imports, err := svc.ListImports(
				context.TODO(),
				&cloudformation.ListImportsInput{ExportName: &export.ExportName})
			if err != nil {
				if isNotImportedError(err) {
					resexport.Imported = false
					c <- importResult{output: resexport}
				} else {
					c <- importResult{output: resexport, err: fmt.Errorf("ListImports for %q: %w", export.ExportName, err)}
				}
				return
			}
			resexport.Imported = true
			resexport.ImportedBy = imports.Imports
			c <- importResult{output: resexport}
		}(export)
	}
	var errs []error
	for i := range results {
		r := <-c
		results[i] = r.output
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
	stackRegex := "^" + strings.ReplaceAll(stackfilter, "*", ".*") + "$"
	exportRegex := "^" + strings.ReplaceAll(exportfilter, "*", ".*") + "$"
	if strings.Contains(stackfilter, "*") {
		if matched, err := regexp.MatchString(stackRegex, *stack.StackName); !matched || err != nil {
			return result
		}
	}
	for _, output := range stack.Outputs {
		if exportsOnly && aws.ToString(output.ExportName) == "" {
			continue
		}
		if exportfilter != "" {
			if matched, err := regexp.MatchString(exportRegex, *output.ExportName); !matched || err != nil {
				continue
			}
		}
		parsedOutput := CfnOutput{
			StackName:   *stack.StackName,
			OutputKey:   *output.OutputKey,
			OutputValue: *output.OutputValue,
			ExportName:  aws.ToString(output.ExportName),
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
func (output *CfnOutput) FillImports(svc CFNListImportsAPI) error {
	if output.ExportName == "" {
		return nil
	}
	imports, err := svc.ListImports(
		context.TODO(),
		&cloudformation.ListImportsInput{ExportName: &output.ExportName})
	if err != nil {
		if isNotImportedError(err) {
			output.Imported = false
			output.ImportedBy = nil
			return nil
		}
		return fmt.Errorf("ListImports for %q: %w", output.ExportName, err)
	}
	output.Imported = true
	output.ImportedBy = imports.Imports
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
