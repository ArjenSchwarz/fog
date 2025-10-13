package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

// StartDriftDetection initiates drift detection for a stack and returns the detection ID
func StartDriftDetection(stackName *string, svc CloudFormationDetectStackDriftAPI) *string {
	input := &cloudformation.DetectStackDriftInput{
		StackName: stackName,
	}
	result, err := svc.DetectStackDrift(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	return result.StackDriftDetectionId
}

// WaitForDriftDetectionToFinish polls until drift detection completes and returns the final status
func WaitForDriftDetectionToFinish(driftDetectionId *string, svc CloudFormationDescribeStackDriftDetectionStatusAPI) types.StackDriftDetectionStatus {
	input := &cloudformation.DescribeStackDriftDetectionStatusInput{
		StackDriftDetectionId: driftDetectionId,
	}
	result, err := svc.DescribeStackDriftDetectionStatus(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	if result.DetectionStatus == types.StackDriftDetectionStatusDetectionInProgress {
		time.Sleep(5 * time.Second)
		return WaitForDriftDetectionToFinish(driftDetectionId, svc)
	}
	return result.DetectionStatus
}

// GetDefaultStackDrift retrieves all resource drift information for a stack
func GetDefaultStackDrift(stackName *string, svc CloudFormationDescribeStackResourceDriftsAPI) []types.StackResourceDrift {
	input := &cloudformation.DescribeStackResourceDriftsInput{
		StackName: stackName,
	}

	var allDrifts []types.StackResourceDrift
	var nextToken *string

	for {
		if nextToken != nil {
			input.NextToken = nextToken
		}

		output, err := svc.DescribeStackResourceDrifts(context.TODO(), input)
		if err != nil {
			panic(err)
		}

		allDrifts = append(allDrifts, output.StackResourceDrifts...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allDrifts
}

// GetUncheckedStackResources returns stack resources that have not been checked for drift
func GetUncheckedStackResources(stackName *string, checkedResources []string, svc interface {
	CloudFormationDescribeStacksAPI
	CloudFormationDescribeStackResourcesAPI
}) []CfnResource {
	resources := GetResources(stackName, svc)
	uncheckedresources := []CfnResource{}
	for _, resource := range resources {
		if stringInSlice(resource.LogicalID, checkedResources) {
			continue
		}
		uncheckedresources = append(uncheckedresources, resource)
	}
	return uncheckedresources
}

// GetResource retrieves a specific resource using Cloud Control API
func GetResource(client *cloudcontrol.Client, typeName string, identifier string) (*cloudcontrol.GetResourceOutput, error) {
	input := &cloudcontrol.GetResourceInput{
		TypeName:   &typeName,
		Identifier: &identifier,
	}

	result, err := client.GetResource(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return result, nil
}

// ListAllResources lists all resources of a given type using Cloud Control API or service-specific APIs
func ListAllResources(typeName string, client *cloudcontrol.Client, ssoClient *ssoadmin.Client, organizationsClient *organizations.Client) (map[string]string, error) {
	if typeName == "AWS::SSO::PermissionSet" {
		return GetPermissionSetArns(ssoClient)
	}
	if typeName == "AWS::SSO::Assignment" {
		return GetAssignmentArns(ssoClient, organizationsClient)
	}
	// input := &cloudcontrol.ListResourcesInput{
	// 	TypeName: &typeName,
	// }

	// result, err := client.ListResources(context.TODO(), input)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to list resources: %w", err)
	// }

	return map[string]string{}, nil
}
