package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudcontrol"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// StartDriftDetection initiates drift detection for a stack and returns the detection ID
func StartDriftDetection(ctx context.Context, stackName *string, svc CloudFormationDetectStackDriftAPI) (*string, error) {
	input := &cloudformation.DetectStackDriftInput{
		StackName: stackName,
	}
	result, err := svc.DetectStackDrift(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to start drift detection: %w", err)
	}
	return result.StackDriftDetectionId, nil
}

// WaitForDriftDetectionToFinish polls until drift detection completes and returns the final status
func WaitForDriftDetectionToFinish(ctx context.Context, driftDetectionId *string, svc CloudFormationDescribeStackDriftDetectionStatusAPI) (types.StackDriftDetectionStatus, error) {
	input := &cloudformation.DescribeStackDriftDetectionStatusInput{
		StackDriftDetectionId: driftDetectionId,
	}
	for {
		result, err := svc.DescribeStackDriftDetectionStatus(ctx, input)
		if err != nil {
			return "", fmt.Errorf("failed to check drift detection status: %w", err)
		}
		if result.DetectionStatus != types.StackDriftDetectionStatusDetectionInProgress {
			return result.DetectionStatus, nil
		}
		time.Sleep(5 * time.Second)
	}
}

// GetDefaultStackDrift retrieves all resource drift information for a stack
func GetDefaultStackDrift(ctx context.Context, stackName *string, svc CloudFormationDescribeStackResourceDriftsAPI) ([]types.StackResourceDrift, error) {
	input := &cloudformation.DescribeStackResourceDriftsInput{
		StackName: stackName,
	}

	var allDrifts []types.StackResourceDrift
	var nextToken *string

	for {
		if nextToken != nil {
			input.NextToken = nextToken
		}

		output, err := svc.DescribeStackResourceDrifts(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve stack drifts: %w", err)
		}

		allDrifts = append(allDrifts, output.StackResourceDrifts...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allDrifts, nil
}

// GetUncheckedStackResources returns stack resources that have not been checked for drift
func GetUncheckedStackResources(ctx context.Context, stackName *string, checkedResources []string, svc interface {
	CloudFormationDescribeStacksAPI
	CloudFormationDescribeStackResourcesAPI
}) ([]CfnResource, error) {
	resources, err := GetResources(ctx, stackName, svc)
	if err != nil {
		return nil, err
	}
	uncheckedresources := []CfnResource{}
	for _, resource := range resources {
		if stringInSlice(resource.LogicalID, checkedResources) {
			continue
		}
		uncheckedresources = append(uncheckedresources, resource)
	}
	return uncheckedresources, nil
}

// GetResource retrieves a specific resource using Cloud Control API
func GetResource(ctx context.Context, client *cloudcontrol.Client, typeName string, identifier string) (*cloudcontrol.GetResourceOutput, error) {
	input := &cloudcontrol.GetResourceInput{
		TypeName:   &typeName,
		Identifier: &identifier,
	}

	result, err := client.GetResource(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return result, nil
}

// ListAllResources lists all resources of a given type using Cloud Control API or service-specific APIs.
// For SSO types it delegates to service-specific functions. For all other types it uses the
// Cloud Control ListResources API with pagination.
func ListAllResources(ctx context.Context, typeName string, client CloudControlListResourcesAPI, ssoClient interface {
	SSOAdminListInstancesAPI
	SSOAdminListPermissionSetsAPI
	SSOAdminListAccountAssignmentsAPI
}, organizationsClient OrganizationsListAccountsAPI) (map[string]string, error) {
	if typeName == "AWS::SSO::PermissionSet" {
		return GetPermissionSetArns(ctx, ssoClient)
	}
	if typeName == "AWS::SSO::Assignment" {
		return GetAssignmentArns(ctx, ssoClient, organizationsClient)
	}

	resources := map[string]string{}
	input := &cloudcontrol.ListResourcesInput{
		TypeName: &typeName,
	}

	var nextToken *string
	for {
		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := client.ListResources(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources of type %s: %w", typeName, err)
		}

		for _, desc := range result.ResourceDescriptions {
			if desc.Identifier != nil {
				resources[*desc.Identifier] = typeName
			}
		}

		if result.NextToken == nil {
			break
		}
		nextToken = result.NextToken
	}

	return resources, nil
}
