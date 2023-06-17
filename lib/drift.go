package lib

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
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
		// StackResourceDriftStatusFilters: []types.StackResourceDriftStatus{types.StackResourceDriftStatusModified, types.StackResourceDriftStatusDeleted},
	}
	result, err := svc.DescribeStackResourceDrifts(context.TODO(), input)
	if err != nil {
		panic(err)
	}
	return result.StackResourceDrifts
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
