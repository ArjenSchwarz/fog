package deployment

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
)

// createChangeSet is a helper used by Service.CreateChangeset.
func (s *Service) createChangeSet(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, ferr.FogError) {
	errorCtx := ferr.GetErrorContext(ctx)

	// Determine changeset type based on whether it's a new stack
	changesetType := cfnTypes.ChangeSetTypeUpdate
	if plan.IsNewStack {
		changesetType = cfnTypes.ChangeSetTypeCreate
	}

	// Build CreateChangeSet input
	input := &cloudformation.CreateChangeSetInput{
		StackName:     aws.String(plan.StackName),
		ChangeSetName: aws.String(plan.ChangesetName),
		ChangeSetType: changesetType,
		Capabilities:  cfnTypes.CapabilityCapabilityAutoExpand.Values(),
	}

	// Set template source
	if plan.Template.S3URL != "" {
		input.TemplateURL = aws.String(plan.Template.S3URL)
	} else if plan.Template.Content != "" {
		input.TemplateBody = aws.String(plan.Template.Content)
	} else {
		return nil, ferr.ContextualError(errorCtx, ferr.ErrTemplateInvalid, "template content or S3 URL is required")
	}

	// Add parameters if provided
	if len(plan.Parameters) > 0 {
		input.Parameters = plan.Parameters
	}

	// Add tags if provided
	if len(plan.Tags) > 0 {
		input.Tags = plan.Tags
	}

	// Create the changeset
	output, err := s.cfnClient.CreateChangeSet(ctx, input)
	if err != nil {
		return nil, ferr.ContextualError(errorCtx, ferr.ErrChangesetFailed, fmt.Sprintf("failed to create changeset: %v", err))
	}

	// Wait for changeset creation to complete
	changesetResult, fogErr := s.waitForChangesetCreation(ctx, plan.StackName, plan.ChangesetName)
	if fogErr != nil {
		return nil, fogErr
	}

	// Set the changeset ID from the creation response
	changesetResult.ID = aws.ToString(output.Id)

	return changesetResult, nil
}

// waitForChangesetCreation waits for changeset creation to complete and returns the result
func (s *Service) waitForChangesetCreation(ctx context.Context, stackName, changesetName string) (*services.ChangesetResult, ferr.FogError) {
	errorCtx := ferr.GetErrorContext(ctx)

	// Wait a bit before checking status
	time.Sleep(5 * time.Second)

	maxAttempts := 60 // 5 minutes with 5-second intervals
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Describe the changeset
		input := &cloudformation.DescribeChangeSetInput{
			StackName:     aws.String(stackName),
			ChangeSetName: aws.String(changesetName),
		}

		output, err := s.cfnClient.DescribeChangeSet(ctx, input)
		if err != nil {
			return nil, ferr.ContextualError(errorCtx, ferr.ErrChangesetFailed, fmt.Sprintf("failed to describe changeset: %v", err))
		}

		status := output.Status

		// Check if changeset creation is complete
		if status == cfnTypes.ChangeSetStatusCreateComplete {
			return s.buildChangesetResult(output), nil
		}

		// Check if changeset creation failed
		if status == cfnTypes.ChangeSetStatusFailed || status == cfnTypes.ChangeSetStatusDeleteFailed {
			statusReason := ""
			if output.StatusReason != nil {
				statusReason = aws.ToString(output.StatusReason)
			}
			return nil, ferr.ContextualError(errorCtx, ferr.ErrChangesetFailed, fmt.Sprintf("changeset creation failed: %s", statusReason))
		}

		// Continue waiting if still in progress
		if status == cfnTypes.ChangeSetStatusCreateInProgress || status == cfnTypes.ChangeSetStatusCreatePending {
			time.Sleep(5 * time.Second)
			continue
		}

		// Unexpected status
		return nil, ferr.ContextualError(errorCtx, ferr.ErrChangesetFailed, fmt.Sprintf("unexpected changeset status: %s", status))
	}

	return nil, ferr.ContextualError(errorCtx, ferr.ErrNetworkTimeout, "timeout waiting for changeset creation")
}

// buildChangesetResult converts CloudFormation DescribeChangeSetOutput to services.ChangesetResult
func (s *Service) buildChangesetResult(output *cloudformation.DescribeChangeSetOutput) *services.ChangesetResult {
	result := &services.ChangesetResult{
		Name:         aws.ToString(output.ChangeSetName),
		ID:           aws.ToString(output.ChangeSetId),
		Status:       output.Status,
		StatusReason: aws.ToString(output.StatusReason),
		Changes:      output.Changes,
		CreationTime: aws.ToTime(output.CreationTime),
		StackID:      aws.ToString(output.StackId),
	}

	// Generate console URL for the changeset
	if s.config != nil {
		region := s.config.GetString("region")
		if region != "" {
			result.ConsoleURL = fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%s#/stacks/changesets/changes?stackId=%s&changeSetId=%s",
				region, result.StackID, result.ID)
		}
	}

	return result
}

// executeChangeset executes a changeset and waits for deployment completion
func (s *Service) executeChangeset(ctx context.Context, plan *services.DeploymentPlan, changeset *services.ChangesetResult) (*services.DeploymentResult, ferr.FogError) {
	errorCtx := ferr.GetErrorContext(ctx)
	startTime := time.Now()

	// Execute the changeset
	input := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: aws.String(changeset.Name),
		StackName:     aws.String(plan.StackName),
	}

	_, err := s.cfnClient.ExecuteChangeSet(ctx, input)
	if err != nil {
		return nil, ferr.ContextualError(errorCtx, ferr.ErrDeploymentFailed, fmt.Sprintf("failed to execute changeset: %v", err))
	}

	// Wait for deployment completion
	deploymentResult, fogErr := s.waitForDeploymentCompletion(ctx, plan.StackName, startTime)
	if fogErr != nil {
		return nil, fogErr
	}

	deploymentResult.ExecutionTime = time.Since(startTime)
	return deploymentResult, nil
}

// waitForDeploymentCompletion waits for deployment to complete and returns the result
func (s *Service) waitForDeploymentCompletion(ctx context.Context, stackName string, startTime time.Time) (*services.DeploymentResult, ferr.FogError) {
	errorCtx := ferr.GetErrorContext(ctx)

	// Wait before checking status
	time.Sleep(10 * time.Second)

	maxAttempts := 360 // 30 minutes with 5-second intervals
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Describe the stack to get current status
		input := &cloudformation.DescribeStacksInput{
			StackName: aws.String(stackName),
		}

		output, err := s.cfnClient.DescribeStacks(ctx, input)
		if err != nil {
			return nil, ferr.ContextualError(errorCtx, ferr.ErrDeploymentFailed, fmt.Sprintf("failed to describe stack: %v", err))
		}

		if len(output.Stacks) == 0 {
			return nil, ferr.ContextualError(errorCtx, ferr.ErrStackNotFound, "stack not found during deployment")
		}

		stack := output.Stacks[0]
		status := stack.StackStatus

		// Check if deployment completed successfully
		if s.isSuccessStatus(status) {
			return s.buildDeploymentResult(stack, true, ""), nil
		}

		// Check if deployment failed
		if s.isFailureStatus(status) {
			statusReason := aws.ToString(stack.StackStatusReason)
			return s.buildDeploymentResult(stack, false, statusReason), nil
		}

		// Continue waiting if still in progress
		if s.isInProgressStatus(status) {
			time.Sleep(5 * time.Second)
			continue
		}

		// Unexpected status
		return nil, ferr.ContextualError(errorCtx, ferr.ErrDeploymentFailed, fmt.Sprintf("unexpected stack status: %s", status))
	}

	return nil, ferr.ContextualError(errorCtx, ferr.ErrNetworkTimeout, "timeout waiting for deployment completion")
}

// buildDeploymentResult creates a DeploymentResult from CloudFormation stack information
func (s *Service) buildDeploymentResult(stack cfnTypes.Stack, success bool, errorMessage string) *services.DeploymentResult {
	result := &services.DeploymentResult{
		StackID:      aws.ToString(stack.StackId),
		Status:       stack.StackStatus,
		Success:      success,
		ErrorMessage: errorMessage,
	}

	// Add outputs if available
	if stack.Outputs != nil {
		result.Outputs = stack.Outputs
	}

	return result
}

// isSuccessStatus checks if a stack status indicates successful completion
func (s *Service) isSuccessStatus(status cfnTypes.StackStatus) bool {
	successStatuses := []cfnTypes.StackStatus{
		cfnTypes.StackStatusCreateComplete,
		cfnTypes.StackStatusUpdateComplete,
		cfnTypes.StackStatusImportComplete,
	}
	for _, successStatus := range successStatuses {
		if status == successStatus {
			return true
		}
	}
	return false
}

// isFailureStatus checks if a stack status indicates failure
func (s *Service) isFailureStatus(status cfnTypes.StackStatus) bool {
	failureStatuses := []cfnTypes.StackStatus{
		cfnTypes.StackStatusCreateFailed,
		cfnTypes.StackStatusRollbackFailed,
		cfnTypes.StackStatusRollbackComplete,
		cfnTypes.StackStatusUpdateFailed,
		cfnTypes.StackStatusUpdateRollbackFailed,
		cfnTypes.StackStatusUpdateRollbackComplete,
		cfnTypes.StackStatusDeleteFailed,
		cfnTypes.StackStatusImportRollbackFailed,
		cfnTypes.StackStatusImportRollbackComplete,
	}
	for _, failureStatus := range failureStatuses {
		if status == failureStatus {
			return true
		}
	}
	return false
}

// isInProgressStatus checks if a stack status indicates operation in progress
func (s *Service) isInProgressStatus(status cfnTypes.StackStatus) bool {
	inProgressStatuses := []cfnTypes.StackStatus{
		cfnTypes.StackStatusCreateInProgress,
		cfnTypes.StackStatusRollbackInProgress,
		cfnTypes.StackStatusDeleteInProgress,
		cfnTypes.StackStatusUpdateInProgress,
		cfnTypes.StackStatusUpdateCompleteCleanupInProgress,
		cfnTypes.StackStatusUpdateRollbackInProgress,
		cfnTypes.StackStatusUpdateRollbackCompleteCleanupInProgress,
		cfnTypes.StackStatusImportInProgress,
		cfnTypes.StackStatusImportRollbackInProgress,
	}
	for _, inProgressStatus := range inProgressStatuses {
		if status == inProgressStatus {
			return true
		}
	}
	return false
}
