package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
)

// CfnResource represents a CloudFormation resource with its identifying information and status.
type CfnResource struct {
	StackName  string
	Type       string
	ResourceID string
	LogicalID  string
	Status     string
}

// GetResources returns all the resources in the account and region. If stackname
// is provided, results will be limited to that stack.
func GetResources(ctx context.Context, stackname *string, svc interface {
	CloudFormationDescribeStacksAPI
	CloudFormationListStackResourcesAPI
}) ([]CfnResource, error) {
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	paginator := cloudformation.NewDescribeStacksPaginator(svc, input)
	allstacks := make([]types.Stack, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe stacks: %w", err)
		}
		allstacks = append(allstacks, output.Stacks...)
	}
	tocheckstacks := make([]types.Stack, 0)
	for _, stack := range allstacks {
		stackLabel := aws.ToString(stack.StackName)
		if stackLabel == "" {
			continue
		}
		if strings.Contains(*stackname, "*") {
			if !GlobToRegex(*stackname).MatchString(stackLabel) {
				continue
			}
		}
		tocheckstacks = append(tocheckstacks, stack)
	}
	resourcelist := make([]CfnResource, 0)
	for _, stack := range tocheckstacks {
		stackLabel := aws.ToString(stack.StackName)
		resources, err := ListAllStackResources(ctx, stack.StackName, svc)
		if err != nil {
			var ae smithy.APIError
			if errors.As(err, &ae) {
				// If the error is because of throttling, we'll wait 5 seconds before trying the same query again
				if ae.ErrorCode() == "Throttling" && ae.ErrorMessage() == "Rate exceeded" {
					time.Sleep(5 * time.Second)
					resources, err = ListAllStackResources(ctx, stack.StackName, svc)
					// If it still fails after retry, return the error
					if err != nil {
						return nil, fmt.Errorf("failed to list stack resources for %s after throttling retry: %w", stackLabel, err)
					}
				} else {
					// If it's another type of API error, return it with the original error wrapped
					return nil, fmt.Errorf("failed to list stack resources for %s (%s): %w", stackLabel, ae.ErrorCode(), ae)
				}
			} else {
				// If it's a completely different type of error, return it
				return nil, fmt.Errorf("failed to list stack resources for %s: %w", stackLabel, err)
			}
		}
		for _, resource := range resources {
			physicalID := aws.ToString(resource.PhysicalResourceId)
			if physicalID == "" {
				continue
			}
			resitem := CfnResource{
				StackName:  stackLabel,
				Type:       aws.ToString(resource.ResourceType),
				ResourceID: physicalID,
				LogicalID:  aws.ToString(resource.LogicalResourceId),
				Status:     string(resource.ResourceStatus),
			}
			resourcelist = append(resourcelist, resitem)
		}
	}
	return resourcelist, nil
}

// ListAllStackResources returns every resource in the given stack.
// DescribeStackResources only returns the first 100 resources of a stack, so
// this follows all ListStackResources pages instead to get the full inventory.
func ListAllStackResources(ctx context.Context, stackName *string, svc CloudFormationListStackResourcesAPI) ([]types.StackResourceSummary, error) {
	paginator := cloudformation.NewListStackResourcesPaginator(svc, &cloudformation.ListStackResourcesInput{StackName: stackName})
	resources := make([]types.StackResourceSummary, 0)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		resources = append(resources, page.StackResourceSummaries...)
	}
	return resources, nil
}
