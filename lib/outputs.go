package lib

import (
	"context"
	"errors"
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

// GetExports returns all the exports in the account and region. If stackname
// is provided, results will be limited to that stack. Each export will also
// be checked whether it is being imported or not.
func GetExports(stackname *string, exportname *string, svc CFNExportsAPI) []CfnOutput {
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
			paginator := cloudformation.NewListImportsPaginator(svc, &cloudformation.ListImportsInput{ExportName: &export.ExportName})
			var allImports []string
			var paginationErr error
			for paginator.HasMorePages() {
				page, err := paginator.NextPage(context.TODO())
				if err != nil {
					paginationErr = err
					break
				}
				allImports = append(allImports, page.Imports...)
			}
			if paginationErr != nil {
				// TODO limit this to only not found errors: "Export 'stackname' is not imported by any stack."
				resexport.Imported = false
			} else {
				resexport.Imported = true
				resexport.ImportedBy = allImports
			}
			c <- resexport
		}(export)
	}
	for i := range results {
		results[i] = <-c
	}
	return results
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

// FillImports populates the import information for an exported output
func (output *CfnOutput) FillImports(svc CFNListImportsAPI) {
	if output.ExportName == "" {
		return
	}
	paginator := cloudformation.NewListImportsPaginator(svc, &cloudformation.ListImportsInput{ExportName: &output.ExportName})
	var allImports []string
	var paginationErr error
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			paginationErr = err
			break
		}
		allImports = append(allImports, page.Imports...)
	}
	if paginationErr != nil {
		// TODO limit this to only not found errors: "Export 'stackname' is not imported by any stack."
		output.Imported = false
	} else {
		output.Imported = true
		output.ImportedBy = allImports
	}
}
