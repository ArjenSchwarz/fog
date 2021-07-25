package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type DeployInfo struct {
	Changeset      *ChangesetInfo
	ChangesetName  string
	IsNew          bool
	ParametersFile string
	RawStack       *types.Stack
	StackArn       string
	StackName      string
	TagsFile       string
	Template       string
	TemplateName   string
}

func (deploy *DeployInfo) ChangesetType() types.ChangeSetType {
	if deploy.IsNew {
		return types.ChangeSetTypeCreate
	}
	return types.ChangeSetTypeUpdate
}

func GetStack(stackname *string, svc *cloudformation.Client) (types.Stack, error) {
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	resp, err := svc.DescribeStacks(context.TODO(), input)
	if err != nil {
		return types.Stack{}, err
	}
	return resp.Stacks[0], err

}

func StackExists(deployment *DeployInfo, svc *cloudformation.Client) bool {
	stack, err := GetStack(&deployment.StackName, svc)
	if err != nil {
		deployment.RawStack = &stack
	}
	return err == nil
}

func (deployment DeployInfo) IsReadyForUpdate(svc *cloudformation.Client) (bool, string) {
	stack, err := deployment.GetStack(svc)
	if err != nil {
		return false, ""
	}
	availableStatuses := []string{
		string(types.StackStatusCreateComplete),
		string(types.StackStatusImportComplete),
		string(types.StackStatusUpdateComplete),
		string(types.StackStatusRollbackComplete),
		string(types.StackStatusUpdateRollbackComplete),
	}
	return stringInSlice(string(stack.StackStatus), availableStatuses), string(stack.StackStatus)
}

func (deployment DeployInfo) IsOngoing(svc *cloudformation.Client) bool {
	stack, err := deployment.GetFreshStack(svc)
	if err != nil {
		return false
	}
	availableStatuses := []string{
		string(types.StackStatusCreateComplete),
		string(types.StackStatusImportComplete),
		string(types.StackStatusUpdateComplete),
		string(types.StackStatusRollbackComplete),
		string(types.StackStatusUpdateRollbackComplete),
	}
	return !stringInSlice(string(stack.StackStatus), availableStatuses)
}

// stringInSlice checks if a string exists in a slice
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (deployment *DeployInfo) CreateChangeSet(svc *cloudformation.Client) (string, error) {
	input := &cloudformation.CreateChangeSetInput{
		StackName:     &deployment.StackName,
		TemplateBody:  &deployment.Template,
		ChangeSetType: deployment.ChangesetType(),
		ChangeSetName: &deployment.ChangesetName,
		Capabilities:  types.CapabilityCapabilityAutoExpand.Values(),
	}
	if deployment.ParametersFile != "" {
		input.Parameters = deployment.GetParameterSlice()
	}
	if deployment.TagsFile != "" {
		input.Tags = deployment.GetTagSlice()
	}
	resp, err := svc.CreateChangeSet(context.TODO(), input)
	if err != nil {
		return "", err
	}
	return *resp.Id, nil
}

func (deployment *DeployInfo) GetParameterSlice() []types.Parameter {
	result := make([]types.Parameter, 0)
	err := json.Unmarshal([]byte(deployment.ParametersFile), &result)
	if err != nil {
		panic(err)
	}
	return result
}

func (deployment *DeployInfo) GetTagSlice() []types.Tag {
	result := make([]types.Tag, 0)
	err := json.Unmarshal([]byte(deployment.TagsFile), &result)
	if err != nil {
		panic(err)
	}
	return result
}

func (deployment *DeployInfo) WaitUntilChangesetDone(svc *cloudformation.Client) (*ChangesetInfo, error) {
	time.Sleep(5 * time.Second)
	changeset := ChangesetInfo{}
	availableStatuses := []string{
		string(types.ChangeSetStatusCreateComplete),
		string(types.ChangeSetStatusFailed),
		string(types.ChangeSetStatusDeleteFailed),
	}
	resp, err := deployment.GetChangeset(svc)
	if err != nil {
		return &changeset, err
	}

	for !stringInSlice(string(resp.Status), availableStatuses) {
		time.Sleep(5 * time.Second)
		resp, err = deployment.GetChangeset(svc)
		if err != nil {
			return &changeset, err
		}
	}
	for _, change := range resp.Changes {
		changestruct := ChangesetChanges{
			Action:      string(change.ResourceChange.Action),
			Replacement: string(change.ResourceChange.Replacement),
			ResourceID:  "",
			LogicalID:   *change.ResourceChange.LogicalResourceId,
			Type:        string(*change.ResourceChange.ResourceType),
		}
		if change.ResourceChange.PhysicalResourceId != nil {
			changestruct.ResourceID = *change.ResourceChange.PhysicalResourceId
		}
		changeset.AddChange(changestruct)
	}
	changeset.StackID = *resp.StackId
	changeset.StackName = *resp.StackName
	changeset.Status = string(resp.Status)
	statusreason := ""
	if resp.StatusReason != nil {
		statusreason = *resp.StatusReason
	}
	changeset.StatusReason = statusreason
	changeset.ID = *resp.ChangeSetId
	changeset.Name = *resp.ChangeSetName
	changeset.CreationTime = *resp.CreationTime
	deployment.StackArn = changeset.StackID
	deployment.Changeset = &changeset
	return &changeset, err
}

func (deployment *DeployInfo) GetChangeset(svc *cloudformation.Client) (cloudformation.DescribeChangeSetOutput, error) {
	input := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: &deployment.ChangesetName,
		StackName:     &deployment.StackName,
	}
	resp, err := svc.DescribeChangeSet(context.TODO(), input)
	if err != nil {
		return *resp, err
	}
	return *resp, nil
}

func (deployment *DeployInfo) GetFreshStack(svc *cloudformation.Client) (types.Stack, error) {
	return GetStack(&deployment.StackArn, svc)
}

func (deployment *DeployInfo) GetStack(svc *cloudformation.Client) (types.Stack, error) {
	if deployment.RawStack == nil {
		stack, err := GetStack(&deployment.StackName, svc)
		if err != nil {
			return stack, err
		}
		deployment.RawStack = &stack
	}
	return *deployment.RawStack, nil
}

func (deployment *DeployInfo) GetEvents(svc *cloudformation.Client) ([]types.StackEvent, error) {
	input := &cloudformation.DescribeStackEventsInput{
		StackName: &deployment.StackName,
	}
	resp, err := svc.DescribeStackEvents(context.TODO(), input)
	return resp.StackEvents, err

}

func (deployment *DeployInfo) DeleteStack(svc *cloudformation.Client) bool {
	input := &cloudformation.DeleteStackInput{
		StackName: &deployment.StackName,
	}
	fmt.Print(deployment.StackArn)
	_, err := svc.DeleteStack(context.TODO(), input)

	return err == nil
}
