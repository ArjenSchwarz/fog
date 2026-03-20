package lib

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
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

// GetExports returns all the exports in the account and region. If stackname
// is provided, results will be limited to that stack. Each export will also
// be checked whether it is being imported or not.
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
			return nil, fmt.Errorf("describing stacks: %w", err)
		}
		allstacks = append(allstacks, output.Stacks...)
	}
	for _, stack := range allstacks {
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
			imports, err := svc.ListImports(
				context.TODO(),
				&cloudformation.ListImportsInput{ExportName: &export.ExportName})
			if err != nil {
				// TODO limit this to only not found errors: "Export 'stackname' is not imported by any stack."
				resexport.Imported = false
			} else {
				resexport.Imported = true
				resexport.ImportedBy = imports.Imports
			}
			c <- resexport
		}(export)
	}
	for i := range results {
		results[i] = <-c
	}
	return results, nil
}

func getOutputsForStack(stack types.Stack, stackfilter string, exportfilter string, exportsOnly bool) []CfnOutput {
	result := []CfnOutput{}
	stackRegex := "^" + strings.ReplaceAll(regexp.QuoteMeta(stackfilter), "\\*", ".*") + "$"
	exportRegex := "^" + strings.ReplaceAll(regexp.QuoteMeta(exportfilter), "\\*", ".*") + "$"
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

// FillImports populates the import information for an exported output
func (output *CfnOutput) FillImports(svc CFNListImportsAPI) {
	if output.ExportName == "" {
		return
	}
	imports, err := svc.ListImports(
		context.TODO(),
		&cloudformation.ListImportsInput{ExportName: &output.ExportName})
	if err != nil {
		// TODO limit this to only not found errors: "Export 'stackname' is not imported by any stack."
		output.Imported = false
	} else {
		output.Imported = true
		output.ImportedBy = imports.Imports
	}
}
