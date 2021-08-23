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
func GetExports(stackname *string, exportname *string, svc *cloudformation.Client) []CfnOutput {
	exports := []CfnOutput{}
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	resp, err := svc.DescribeStacks(context.TODO(), input)
	if err != nil {
		var bne *smithy.OperationError
		if errors.As(err, &bne) {
			log.Fatalln("error:", bne.Err)
		}
		log.Fatalln(err)
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
			imports, err := svc.ListImports(
				context.TODO(),
				&cloudformation.ListImportsInput{ExportName: &export.ExportName})
			if err != nil {
				//TODO limit this to only not found errors: "Export 'stackname' is not imported by any stack."
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

func getOutputsForStack(stack types.Stack, stackfilter string, exportfilter string, exportsOnly bool) []CfnOutput {
	result := []CfnOutput{}
	stackRegex := "^" + strings.Replace(stackfilter, "*", ".*", -1) + "$"
	exportRegex := "^" + strings.Replace(exportfilter, "*", ".*", -1) + "$"
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

func (output *CfnOutput) FillImports(svc *cloudformation.Client) {
	if output.ExportName == "" {
		return
	}
	imports, err := svc.ListImports(
		context.TODO(),
		&cloudformation.ListImportsInput{ExportName: &output.ExportName})
	if err != nil {
		//TODO limit this to only not found errors: "Export 'stackname' is not imported by any stack."
		output.Imported = false
	} else {
		output.Imported = true
		output.ImportedBy = imports.Imports
	}
}
