package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type DeployInfo struct {
	Changeset         *ChangesetInfo
	ChangesetName     string
	IsNew             bool
	Parameters        []types.Parameter
	RawStack          *types.Stack
	StackArn          string
	StackName         string
	Tags              []types.Tag
	Template          string
	TemplateLocalPath string
	TemplateName      string
	TemplateUrl       string
}

type CfnStack struct {
	Name        string
	Description string
	RawInfo     types.Stack
	Outputs     []CfnOutput
	Resources   []CfnResource
	ImportedBy  []string
}

func (deployment *DeployInfo) ChangesetType() types.ChangeSetType {
	if deployment.IsNew {
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

func GetCfnStacks(stackname *string, svc *cloudformation.Client) (map[string]CfnStack, error) {
	result := make(map[string]CfnStack)
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	resp, err := svc.DescribeStacks(context.TODO(), input)
	if err != nil {
		return result, err
	}
	for _, stack := range resp.Stacks {
		stackobject := CfnStack{
			RawInfo: stack,
			Name:    *stack.StackName,
		}
		if stack.Description != nil {
			stackobject.Description = *stack.Description
		}
		outputs := getOutputsForStack(stack, "", "", false)
		for _, output := range outputs {
			output.FillImports(svc)
			if output.Imported {
				stackobject.ImportedBy = append(stackobject.ImportedBy, output.ImportedBy...)
			}
		}
		stackobject.Outputs = outputs
		result[*stack.StackName] = stackobject
	}
	return result, nil
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

// IsNewStack verifies if a stack is new. This can mean either that it doesn't exist yet or is in review in progress state
func (deployment DeployInfo) IsNewStack(svc *cloudformation.Client) bool {
	stackExists := StackExists(&deployment, svc)
	if !stackExists {
		return true
	}
	stack, err := deployment.GetFreshStack(svc)
	if err != nil {
		return false
	}
	availableStatuses := []string{
		string(types.StackStatusReviewInProgress),
	}
	return stringInSlice(string(stack.StackStatus), availableStatuses)
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
		ChangeSetType: deployment.ChangesetType(),
		ChangeSetName: &deployment.ChangesetName,
		Capabilities:  types.CapabilityCapabilityAutoExpand.Values(),
	}
	if deployment.TemplateUrl != "" {
		input.TemplateURL = &deployment.TemplateUrl
	} else if deployment.Template != "" {
		input.TemplateBody = &deployment.Template
	} else {
		input.UsePreviousTemplate = aws.Bool(true)
	}
	if len(deployment.Parameters) != 0 {
		input.Parameters = deployment.Parameters
	}
	if len(deployment.Tags) != 0 {
		input.Tags = deployment.Tags
	}
	resp, err := svc.CreateChangeSet(context.TODO(), input)
	if err != nil {
		return "", err
	}
	return *resp.Id, nil
}

func ParseParameterString(parameters string) ([]types.Parameter, error) {
	result := make([]types.Parameter, 0)
	err := json.Unmarshal([]byte(parameters), &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func ParseTagString(tags string) ([]types.Tag, error) {
	result := make([]types.Tag, 0)
	err := json.Unmarshal([]byte(tags), &result)
	if err != nil {
		return result, err
	}
	return result, nil
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
	changeset = deployment.AddChangeset(resp)
	return &changeset, err
}

func (deployment *DeployInfo) AddChangeset(resp cloudformation.DescribeChangeSetOutput) ChangesetInfo {
	changeset := ChangesetInfo{}
	for _, change := range resp.Changes {
		changestruct := ChangesetChanges{
			Action:      string(change.ResourceChange.Action),
			Replacement: string(change.ResourceChange.Replacement),
			ResourceID:  aws.ToString(change.ResourceChange.PhysicalResourceId),
			LogicalID:   aws.ToString(change.ResourceChange.LogicalResourceId),
			Type:        aws.ToString(change.ResourceChange.ResourceType),
		}
		if change.ResourceChange.ModuleInfo != nil {
			changestruct.Module = fmt.Sprintf("%v(%v)", aws.ToString(change.ResourceChange.ModuleInfo.LogicalIdHierarchy), aws.ToString(change.ResourceChange.ModuleInfo.TypeHierarchy))
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
	return changeset
}

func (deployment *DeployInfo) GetChangeset(svc *cloudformation.Client) (cloudformation.DescribeChangeSetOutput, error) {
	input := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: &deployment.ChangesetName,
		StackName:     &deployment.StackName,
	}
	resp, err := svc.DescribeChangeSet(context.TODO(), input)
	if err != nil {
		return cloudformation.DescribeChangeSetOutput{}, err
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
	_, err := svc.DeleteStack(context.TODO(), input)

	return err == nil
}
