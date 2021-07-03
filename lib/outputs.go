package lib

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/smithy-go"
)

type CfnOutput struct {
	StackName   string
	OutputKey   string
	OutputValue string
	Description string
	ExportName  string
	Imported    bool
}

// GetExports returns all the exports in the account and region. If stackname
// is provided, results will be limited to that stack. Each export will also
// be checked whether it is being imported or not.
func GetExports(stackname *string, svc *cloudformation.Client) []CfnOutput {
	exports := []CfnOutput{}
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" {
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
		for _, output := range stack.Outputs {
			if aws.ToString(output.ExportName) != "" {
				parsedOutput := CfnOutput{
					StackName:   *stack.StackName,
					OutputKey:   *output.OutputKey,
					OutputValue: *output.OutputValue,
					ExportName:  *output.ExportName,
				}
				if output.Description != nil {
					parsedOutput.Description = *output.Description
				}
				exports = append(exports, parsedOutput)
			}
		}
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
			_, err := svc.ListImports(
				context.TODO(),
				&cloudformation.ListImportsInput{ExportName: &export.ExportName})
			if err != nil {
				resexport.Imported = false
			} else {
				resexport.Imported = true
			}
			c <- resexport
		}(export)
	}
	for i := 0; i < len(results); i++ {
		results[i] = <-c
	}
	return results
}
