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

func StartDriftDetection(stackName *string, svc *cloudformation.Client) *string {
	input := &cloudformation.DetectStackDriftInput{
		StackName: stackName,
	}
	result, err := svc.DetectStackDrift(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	return result.StackDriftDetectionId
}

func WaitForDriftDetectionToFinish(driftDetectionId *string, svc *cloudformation.Client) types.StackDriftDetectionStatus {
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

func GetDefaultStackDrift(stackName *string, svc *cloudformation.Client) []types.StackResourceDrift {
	input := &cloudformation.DescribeStackResourceDriftsInput{
		StackName: stackName,
	}

	var allDrifts []types.StackResourceDrift
	paginator := cloudformation.NewDescribeStackResourceDriftsPaginator(svc, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			panic(err)
		}

		allDrifts = append(allDrifts, output.StackResourceDrifts...)
	}

	return allDrifts
}

func GetUncheckedStackResources(stackName *string, checkedResources []string, svc *cloudformation.Client) []CfnResource {
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
