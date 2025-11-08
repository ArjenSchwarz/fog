// Package lib provides core functionality for managing AWS CloudFormation stacks.
//
// This package contains the business logic for interacting with AWS CloudFormation,
// including stack operations, changeset management, drift detection, resource tracking,
// and template processing. It serves as the foundation for the fog CLI tool.
//
// # Core Components
//
// Stack Operations:
//   - Stack creation, updates, and deletion
//   - Stack status monitoring and event tracking
//   - Stack dependency analysis
//   - Stack export management
//
// Changeset Management:
//   - Creating and executing changesets
//   - Analyzing changeset changes and impact
//   - Changeset validation and status tracking
//
// Drift Detection:
//   - Detecting configuration drift in stacks
//   - Analyzing drift status for resources
//   - Reporting drift details
//
// Resource Management:
//   - Listing and describing stack resources
//   - Resource status and property tracking
//   - Resource type filtering
//
// Template Processing:
//   - Reading and validating CloudFormation templates
//   - Template preprocessing and placeholder replacement
//   - S3 upload for large templates
//   - Support for YAML, JSON, and deployment file formats
//
// AWS Service Integration:
//
// The package integrates with multiple AWS services:
//   - CloudFormation: Core stack and changeset operations
//   - S3: Template storage and retrieval
//   - EC2: VPC, subnet, and transit gateway operations
//   - Identity Center: Permission set management
//
// # Data Types
//
// Key types defined in this package:
//   - DeployInfo: Complete deployment configuration and state
//   - CfnStack: CloudFormation stack representation with metadata
//   - ChangesetInfo: Changeset details and changes
//   - StackEvent/ResourceEvent: Event tracking for deployments
//   - CfnResource: CloudFormation resource with status and properties
//   - StackDeploymentFile: Deployment file configuration
//
// # Error Handling
//
// Functions return standard Go errors. AWS API errors are propagated with
// appropriate context. Callers should check errors and handle them appropriately.
//
// # Examples
//
// Get a stack:
//
//	stack, err := lib.GetStack(ctx, client, "my-stack")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Create a changeset:
//
//	changesetID, err := lib.CreateChangeSet(ctx, client, deployInfo)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Detect drift:
//
//	driftInfo, err := lib.GetDrift(ctx, client, "my-stack")
//	if err != nil {
//	    log.Fatal(err)
//	}
package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/gosimple/slug"
)

// DeployInfo represents all the information required to deploy a CloudFormation stack.
type DeployInfo struct {
	// Changeset contains the ChangesetInfo object with the change set information
	Changeset *ChangesetInfo
	// ChangesetName contains the name of the change set
	ChangesetName string
	// IsDryRun shows whether this is a dry run or not
	IsDryRun bool
	// IsNew shows whether this is a new stack or if it will update one
	IsNew bool
	// Parameters holds a slice of parameter objects
	Parameters []types.Parameter
	// PrechecksFailed shows whether the deployment failed the prechecks
	PrechecksFailed bool
	// RawStack holds the raw version of the stack as returned by AWS
	RawStack *types.Stack
	// StackArn holds the ARN of the stack
	StackArn string
	// StackDeploymentFile holds the contents of the stack deployment file
	StackDeploymentFile *StackDeploymentFile
	// StackName holds the name of the stack
	StackName string
	// Tags holds a slice of tag objects
	Tags []types.Tag
	// Template holds the contents of the template that will be deployed
	Template string
	// TemplateLocalPath is the path relative to the root as defined by the config file
	TemplateLocalPath string
	// TemplateName is the name of the template
	TemplateName string
	// TemplateRelativePath is the path relative to where the command is run
	TemplateRelativePath string
	// TemplateUrl holds the S3 URL where the template has been uploaded to
	TemplateUrl string
}

// CfnStack represents a CloudFormation stack with its associated resources and metadata
type CfnStack struct {
	Name        string
	Id          string
	Description string
	RawInfo     types.Stack
	Outputs     []CfnOutput
	Resources   []CfnResource
	ImportedBy  []string
	Events      []StackEvent
}

// StackEvent represents a CloudFormation stack-level event (create, update, delete)
type StackEvent struct {
	EndDate        time.Time
	ResourceEvents []ResourceEvent
	StartDate      time.Time
	Type           string
	Success        bool
	Milestones     map[time.Time]string
}

// ResourceEvent represents events for a single resource within a stack event
type ResourceEvent struct {
	Resource          CfnResource
	RawInfo           []types.StackEvent
	EventType         string
	StartDate         time.Time
	EndDate           time.Time
	StartStatus       string
	EndStatus         string
	EndStatusReason   string
	ExpectedEndStatus string
}

// ChangesetType returns the appropriate changeset type (CREATE or UPDATE) based on whether this is a new stack
func (deployment *DeployInfo) ChangesetType() types.ChangeSetType {
	if deployment.IsNew {
		return types.ChangeSetTypeCreate
	}
	return types.ChangeSetTypeUpdate
}

// GetStack retrieves a single stack by name or ARN
func GetStack(stackname *string, svc CloudFormationDescribeStacksAPI) (types.Stack, error) {
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

// GetCfnStacks retrieves stacks matching the given name pattern with their outputs and import information
func GetCfnStacks(stackname *string, svc *cloudformation.Client) (map[string]CfnStack, error) {
	result := make(map[string]CfnStack)
	input := &cloudformation.DescribeStacksInput{}
	if *stackname != "" && !strings.Contains(*stackname, "*") {
		input.StackName = stackname
	}
	paginator := cloudformation.NewDescribeStacksPaginator(svc, input)
	allstacks := make([]types.Stack, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		allstacks = append(allstacks, output.Stacks...)
	}
	stackRegex := "^" + strings.ReplaceAll(*stackname, "*", ".*") + "$"
	tocheckstacks := make([]types.Stack, 0)
	for _, stack := range allstacks {
		if strings.Contains(*stackname, "*") {
			if matched, _ := regexp.MatchString(stackRegex, *stack.StackName); !matched {
				continue
			}
		}
		tocheckstacks = append(tocheckstacks, stack)
	}
	for _, stack := range tocheckstacks {
		stackobject := CfnStack{
			RawInfo: stack,
			Name:    *stack.StackName,
			Id:      *stack.StackId,
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
		result[*stack.StackId] = stackobject
	}
	return result, nil
}

// StackExists checks whether the stack in the deployment exists
func StackExists(deployment *DeployInfo, svc CloudFormationDescribeStacksAPI) bool {
	stack, err := GetStack(&deployment.StackName, svc)
	if err != nil {
		deployment.RawStack = &stack
	}
	return err == nil
}

// IsReadyForUpdate checks if the stack is in a state that allows updates
func (deployment DeployInfo) IsReadyForUpdate(svc CloudFormationDescribeStacksAPI) (bool, string) {
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

// IsOngoing checks if there is an ongoing operation on the stack
func (deployment DeployInfo) IsOngoing(svc CloudFormationDescribeStacksAPI) bool {
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
func (deployment DeployInfo) IsNewStack(svc CloudFormationDescribeStacksAPI) bool {
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

// LoadDeploymentFile loads a deployment file and sets the StackDeploymentFile field
func (deployment *DeployInfo) LoadDeploymentFile(filelocation string) error {
	deploymentFile, _, err := ReadDeploymentFile(filelocation)
	if err != nil {
		return err
	}
	deploymentFileObject, err := ParseDeploymentFile(deploymentFile)
	if err != nil {
		return err
	}
	deployment.StackDeploymentFile = &deploymentFileObject
	return nil
}

// stringInSlice checks if a string exists in a slice
func stringInSlice(a string, list []string) bool {
	return slices.Contains(list, a)
}

// CreateChangeSet creates a changeset for the deployment and returns its ID
func (deployment *DeployInfo) CreateChangeSet(svc CloudFormationCreateChangeSetAPI) (string, error) {
	input := &cloudformation.CreateChangeSetInput{
		StackName:     &deployment.StackName,
		ChangeSetType: deployment.ChangesetType(),
		ChangeSetName: &deployment.ChangesetName,
		Capabilities:  types.CapabilityCapabilityAutoExpand.Values(),
	}
	switch {
	case deployment.TemplateUrl != "":
		input.TemplateURL = &deployment.TemplateUrl
	case deployment.Template != "":
		input.TemplateBody = &deployment.Template
	default:
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

// ParseParameterString parses a JSON string into CloudFormation parameters
func ParseParameterString(parameters string) ([]types.Parameter, error) {
	result := make([]types.Parameter, 0)
	err := json.Unmarshal([]byte(parameters), &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// ParseDeploymentFile parses a deployment file and returns a StackDeploymentFile object
func ParseDeploymentFile(deploymentFile string) (StackDeploymentFile, error) {
	// If the deploymentfile is yaml, convert it to json
	if deploymentFile[0] != '{' {
		deploymentFileBytes, err := YamlToJson([]byte(deploymentFile))
		if err != nil {
			return StackDeploymentFile{}, err
		}
		deploymentFile = string(deploymentFileBytes)
	}

	result := StackDeploymentFile{}
	err := json.Unmarshal([]byte(deploymentFile), &result)
	if err != nil {
		fmt.Print(err.Error())
		return result, err
	}

	return result, nil
}

// ParseTagString parses a JSON string into CloudFormation tags
func ParseTagString(tags string) ([]types.Tag, error) {
	result := make([]types.Tag, 0)
	err := json.Unmarshal([]byte(tags), &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

// WaitUntilChangesetDone polls until the changeset creation completes and returns the changeset info
func (deployment *DeployInfo) WaitUntilChangesetDone(svc CloudFormationDescribeChangeSetAPI) (*ChangesetInfo, error) {
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

	for !stringInSlice(string(resp[0].Status), availableStatuses) {
		time.Sleep(5 * time.Second)
		resp, err = deployment.GetChangeset(svc)
		if err != nil {
			return &changeset, err
		}
	}
	changeset = deployment.AddChangeset(resp)
	return &changeset, err
}

// AddChangeset processes the changeset response and adds it to the deployment
func (deployment *DeployInfo) AddChangeset(resp []cloudformation.DescribeChangeSetOutput) ChangesetInfo {
	changeset := ChangesetInfo{}
	for _, changesets := range resp {
		for _, change := range changesets.Changes {
			changestruct := ChangesetChanges{
				Action:      string(change.ResourceChange.Action),
				Replacement: string(change.ResourceChange.Replacement),
				ResourceID:  aws.ToString(change.ResourceChange.PhysicalResourceId),
				LogicalID:   aws.ToString(change.ResourceChange.LogicalResourceId),
				Type:        aws.ToString(change.ResourceChange.ResourceType),
				Details:     change.ResourceChange.Details,
			}
			if change.ResourceChange.ModuleInfo != nil {
				changestruct.Module = fmt.Sprintf("%v(%v)", aws.ToString(change.ResourceChange.ModuleInfo.LogicalIdHierarchy), aws.ToString(change.ResourceChange.ModuleInfo.TypeHierarchy))
			}
			changeset.AddChange(changestruct)
		}
	}
	changeset.StackID = *resp[0].StackId
	changeset.StackName = *resp[0].StackName
	changeset.Status = string(resp[0].Status)
	statusreason := ""
	if resp[0].StatusReason != nil {
		statusreason = *resp[0].StatusReason
	}
	changeset.StatusReason = statusreason
	changeset.ID = *resp[0].ChangeSetId
	changeset.Name = *resp[0].ChangeSetName
	changeset.CreationTime = *resp[0].CreationTime
	deployment.StackArn = changeset.StackID
	deployment.Changeset = &changeset
	return changeset
}

// GetChangeset retrieves the changeset details for the deployment
func (deployment *DeployInfo) GetChangeset(svc CloudFormationDescribeChangeSetAPI) ([]cloudformation.DescribeChangeSetOutput, error) {
	results := []cloudformation.DescribeChangeSetOutput{}
	input := &cloudformation.DescribeChangeSetInput{
		ChangeSetName: &deployment.ChangesetName,
		NextToken:     nil,
		StackName:     &deployment.StackName,
	}
	resp, err := svc.DescribeChangeSet(context.TODO(), input)
	if err != nil {
		return results, err
	}
	results = append(results, *resp)
	// write a for loop to get all the changesets
	for resp.NextToken != nil {
		input = &cloudformation.DescribeChangeSetInput{
			ChangeSetName: &deployment.ChangesetName,
			NextToken:     resp.NextToken,
			StackName:     &deployment.StackName,
		}
		resp, err = svc.DescribeChangeSet(context.TODO(), input)
		if err != nil {
			return results, err
		}
		results = append(results, *resp)
	}
	return results, nil
}

// GetFreshStack retrieves the latest stack information from AWS
func (deployment *DeployInfo) GetFreshStack(svc CloudFormationDescribeStacksAPI) (types.Stack, error) {
	return GetStack(&deployment.StackArn, svc)
}

// GetStack retrieves the stack information, using cached data if available
func (deployment *DeployInfo) GetStack(svc CloudFormationDescribeStacksAPI) (types.Stack, error) {
	if deployment.RawStack == nil {
		stack, err := GetStack(&deployment.StackName, svc)
		if err != nil {
			return stack, err
		}
		deployment.RawStack = &stack
	}
	return *deployment.RawStack, nil
}

// GetEvents retrieves all events for the deployment's stack
func (deployment *DeployInfo) GetEvents(svc CloudFormationDescribeStackEventsAPI) ([]types.StackEvent, error) {
	input := &cloudformation.DescribeStackEventsInput{
		StackName: &deployment.StackName,
	}
	resp, err := svc.DescribeStackEvents(context.TODO(), input)
	if err != nil {
		return nil, err
	}
	return resp.StackEvents, nil
}

// GetCleanedStackName extracts the stack name from an ARN or returns the name as-is
func (deployment *DeployInfo) GetCleanedStackName() string {
	// if deployment.StackName starts with arn, get the name otherwise return deployment.StackName
	if strings.HasPrefix(deployment.StackName, "arn:") {
		filtered := strings.Split(deployment.StackName, "/")
		return filtered[1]
	}
	return deployment.StackName
}

// GetEvents retrieves and processes all events for the stack, organizing them by stack-level events
func (stack *CfnStack) GetEvents(svc *cloudformation.Client) ([]StackEvent, error) {
	if len(stack.Events) != 0 {
		return stack.Events, nil
	}
	input := &cloudformation.DescribeStackEventsInput{
		StackName: &stack.Id,
	}
	paginator := cloudformation.NewDescribeStackEventsPaginator(svc, input)
	allevents := make([]types.StackEvent, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, err
		}
		allevents = append(allevents, output.StackEvents...)
	}
	sort.Sort(ReverseEvents(allevents))
	var resources map[string]ResourceEvent
	var stackEvent StackEvent
	eventName := ""
	finishedEvents := make([]string, 0)
	failedEvents := make([]string, 0)
	for _, event := range allevents {
		if aws.ToString(event.LogicalResourceId) == stack.Name && aws.ToString(event.ResourceType) == "AWS::CloudFormation::Stack" {
			if eventName == "" || strings.HasSuffix(eventName, "COMPLETE") || strings.HasSuffix(eventName, "FAILED") {
				stackEvent = StackEvent{
					StartDate:  *event.Timestamp,
					Milestones: map[time.Time]string{},
				}
				switch string(event.ResourceStatus) {
				case "REVIEW_IN_PROGRESS":
					fallthrough
				case "CREATE_IN_PROGRESS":
					stackEvent.Type = "Create"
				case "UPDATE_IN_PROGRESS":
					stackEvent.Type = "Update"
				case "DELETE_IN_PROGRESS":
					stackEvent.Type = "Delete"
				case "IMPORT_IN_PROGRESS":
					stackEvent.Type = "Import"
				}
				resources = make(map[string]ResourceEvent)
				eventName = string(event.ResourceStatus)
			} else {
				stackEvent.EndDate = *event.Timestamp
				resourceSlice := make([]ResourceEvent, 0)
				for _, revent := range resources {
					resourceSlice = append(resourceSlice, revent)
				}
				stackEvent.ResourceEvents = resourceSlice
				if !strings.Contains(string(event.ResourceStatus), "IN_PROGRESS") {
					if stringInSlice(string(event.ResourceStatus), GetSuccessStates()) {
						stackEvent.Success = true
					} else {
						stackEvent.Success = false
					}
					stack.Events = append(stack.Events, stackEvent)
				}
				eventName = string(event.ResourceStatus)
			}
			stackEvent.Milestones[*event.Timestamp] = string(event.ResourceStatus)
		} else {
			name := fmt.Sprintf("%s-%s-%s", slug.Make(*event.ResourceType), *event.LogicalResourceId, stackEvent.StartDate.Format(time.RFC3339))
			if stringInSlice(name, finishedEvents) {
				name += "-replacement"
			}
			if stringInSlice(name, failedEvents) {
				name += "-cleanup"
			}
			var resource ResourceEvent
			if _, ok := resources[name]; !ok {
				resitem := CfnResource{
					StackName:  stack.Name,
					Type:       aws.ToString(event.ResourceType),
					ResourceID: aws.ToString(event.PhysicalResourceId),
					LogicalID:  aws.ToString(event.LogicalResourceId),
				}
				resource = ResourceEvent{
					Resource:    resitem,
					StartDate:   *event.Timestamp,
					StartStatus: string(event.ResourceStatus),
					EndDate:     *event.Timestamp,
					EndStatus:   string(event.ResourceStatus),
					RawInfo:     []types.StackEvent{event},
				}
				switch {
				case strings.Contains(string(event.ResourceStatus), "CREATE"):
					resource.EventType = "Add"
					resource.ExpectedEndStatus = string(types.ResourceStatusCreateComplete)
				case strings.Contains(string(event.ResourceStatus), "UPDATE"):
					resource.EventType = "Modify"
					resource.ExpectedEndStatus = string(types.ResourceStatusUpdateComplete)
				case strings.Contains(string(event.ResourceStatus), "DELETE"):
					if strings.HasSuffix(name, "-replacement") || strings.HasSuffix(name, "-cleanup") {
						resource.EventType = "Cleanup"
					} else {
						resource.EventType = "Remove"
					}
					resource.ExpectedEndStatus = string(types.ResourceStatusDeleteComplete)
				}
			} else {
				resource = resources[name]
				resource.EndDate = *event.Timestamp
				resource.EndStatus = string(event.ResourceStatus)
				if strings.Contains(string(event.ResourceStatus), "COMPLETE") {
					finishedEvents = append(finishedEvents, name)
				}
				if strings.Contains(string(event.ResourceStatus), "FAILED") {
					failedEvents = append(failedEvents, name)
					resource.EndStatusReason = *event.ResourceStatusReason
				}
				if resource.Resource.ResourceID == "" && *event.PhysicalResourceId != "" {
					resource.Resource.ResourceID = *event.PhysicalResourceId
				}
				if resource.Resource.ResourceID != "" && *event.PhysicalResourceId != "" && !strings.Contains(resource.Resource.ResourceID, *event.PhysicalResourceId) {
					resource.Resource.ResourceID = fmt.Sprintf("%s => %s", resource.Resource.ResourceID, *event.PhysicalResourceId)
				}
			}
			resources[name] = resource
		}
	}
	return stack.Events, nil
}

// GetSuccessStates returns a list of CloudFormation stack statuses that indicate successful completion
func GetSuccessStates() []string {
	return []string{
		string(types.StackStatusCreateComplete),
		string(types.StackStatusImportComplete),
		string(types.StackStatusUpdateComplete),
		string(types.StackStatusDeleteComplete),
	}
}

// GetDuration returns how long the resource took to finish its event
func (event *ResourceEvent) GetDuration() time.Duration {
	return event.EndDate.Sub(event.StartDate)
}

// GetDuration returns how long it took for the stack to finish its event
func (event *StackEvent) GetDuration() time.Duration {
	return event.EndDate.Sub(event.StartDate)
}

// GetEventSummaries retrieves basic event information for the stack
func (stack *CfnStack) GetEventSummaries(svc *cloudformation.Client) ([]types.StackEvent, error) {
	input := &cloudformation.DescribeStackEventsInput{
		StackName: &stack.Id,
	}
	resp, err := svc.DescribeStackEvents(context.TODO(), input)
	return resp.StackEvents, err

}

// DeleteStack deletes the stack and returns true if successful
func (deployment *DeployInfo) DeleteStack(svc CloudFormationDeleteStackAPI) bool {
	input := &cloudformation.DeleteStackInput{
		StackName: &deployment.StackName,
	}
	_, err := svc.DeleteStack(context.TODO(), input)

	return err == nil
}

// GetExecutionTimes retrieves timing information for each resource in the deployment
func (deployment *DeployInfo) GetExecutionTimes(svc CloudFormationDescribeStackEventsAPI) (map[string]map[string]time.Time, error) {
	result := make(map[string]map[string]time.Time)
	events, err := deployment.GetEvents(svc)
	if err != nil {
		return result, err
	}
	for _, event := range events {
		if event.Timestamp.After(deployment.Changeset.CreationTime) {
			name := fmt.Sprintf("%s (%s)", strings.ReplaceAll(*event.ResourceType, ":", " "), *event.LogicalResourceId)
			if _, ok := result[name]; !ok {
				result[name] = make(map[string]time.Time, 0)
			}
			result[name][string(event.ResourceStatus)] = *event.Timestamp
		}
	}
	return result, nil
}

// GetParametersMap converts a slice of parameters into a map of key-value pairs
func GetParametersMap(params []types.Parameter) *map[string]any {
	result := make(map[string]any)
	for _, param := range params {
		result[*param.ParameterKey] = *param.ParameterValue
	}
	return &result
}

// ReverseEvents implements sort.Interface for sorting stack events in chronological order
type ReverseEvents []types.StackEvent

// Len returns the length of the slice
func (a ReverseEvents) Len() int { return len(a) }

// Less compares two events by timestamp
func (a ReverseEvents) Less(i, j int) bool { return a[i].Timestamp.Before(*a[j].Timestamp) }

// Swap swaps two elements in the slice
func (a ReverseEvents) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// SortStacks implements sort.Interface for sorting stacks by name
type SortStacks []CfnStack

// Len returns the length of the slice
func (a SortStacks) Len() int { return len(a) }

// Less compares two stacks by name
func (a SortStacks) Less(i, j int) bool { return strings.Compare(a[i].Name, a[j].Name) == -1 }

// Swap swaps two elements in the slice
func (a SortStacks) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
