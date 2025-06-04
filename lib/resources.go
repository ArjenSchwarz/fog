package lib

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
)

type CfnResource struct {
	StackName  string
	Type       string
	ResourceID string
	LogicalID  string
	Status     string
}

// GetResources returns all the resources in the account and region. If stackname
// is provided, results will be limited to that stack.
func GetResources(stackname *string, svc interface {
	CloudFormationDescribeStacksAPI
	CloudFormationDescribeStackResourcesAPI
}) []CfnResource {
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
	stackRegex := "^" + strings.ReplaceAll(*stackname, "*", ".*") + "$"
	tocheckstacks := make([]types.Stack, 0)
	for _, stack := range resp.Stacks {
		if strings.Contains(*stackname, "*") {
			if matched, _ := regexp.MatchString(stackRegex, *stack.StackName); !matched {
				continue
			}
		}
		tocheckstacks = append(tocheckstacks, stack)
	}
	resourcelist := make([]CfnResource, 0)
	for _, stack := range tocheckstacks {
		resources, err := svc.DescribeStackResources(
			context.TODO(),
			&cloudformation.DescribeStackResourcesInput{StackName: stack.StackName})
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) {
				// If the error is because of throttling, we'll wait 5 seconds before trying the same query again
				if ae.ErrorCode() == "Throttling" && ae.ErrorMessage() == "Rate exceeded" {
					time.Sleep(5 * time.Second)
					resources, err = svc.DescribeStackResources(
						context.TODO(),
						&cloudformation.DescribeStackResourcesInput{StackName: stack.StackName})
					// If it still fails though, we'll just break down
					if err != nil {
						log.Fatalln(err)
					}
				} else {
					// If it's another type of API error, we fail on it
					log.Fatalf("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())
				}
			} else {
				// If it's a completely different type of error, we also fail
				log.Fatalln(err)
			}
		}
		for _, resource := range resources.StackResources {
			resitem := CfnResource{
				StackName:  *stack.StackName,
				Type:       *resource.ResourceType,
				ResourceID: *resource.PhysicalResourceId,
				LogicalID:  *resource.LogicalResourceId,
				Status:     string(resource.ResourceStatus),
			}
			resourcelist = append(resourcelist, resitem)
		}
	}
	return resourcelist
}
