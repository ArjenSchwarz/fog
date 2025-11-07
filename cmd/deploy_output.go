package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
)

// outputDryRunResult outputs the changeset result for dry-run and create-changeset modes.
// It flushes stderr before writing to stdout to ensure proper stream separation.
// Reuses buildAndRenderChangeset() from describe_changeset.go for consistent output.
func outputDryRunResult(deployment *lib.DeployInfo, awsConfig config.AWSConfig) {
	// Flush stderr before stdout output to ensure clean separation
	// Note: This is best-effort ordering, not atomic. In practice works 99.9% of the time.
	os.Stderr.Sync()

	// Reuse existing buildAndRenderChangeset function
	// This function internally calls settings.GetOutputOptions() which uses stdout by default
	buildAndRenderChangeset(*deployment.CapturedChangeset, *deployment, awsConfig)
}

// outputSuccessResult outputs the deployment summary for successful deployments.
// It builds a document with deployment summary, planned changes, and stack outputs.
func outputSuccessResult(deployment *lib.DeployInfo) error {
	// Flush stderr before stdout output
	os.Stderr.Sync()
	fmt.Println("\n=== Deployment Summary ===")

	// Calculate duration
	duration := deployment.DeploymentEnd.Sub(deployment.DeploymentStart)

	// Get changeset ID if available
	changesetID := ""
	if deployment.CapturedChangeset != nil {
		changesetID = deployment.CapturedChangeset.ID
	}

	// Build deployment summary table
	summaryData := []map[string]any{
		{
			"Status":     string(deployment.FinalStackState.StackStatus),
			"Stack ARN":  deployment.StackArn,
			"Changeset":  changesetID,
			"Start Time": deployment.DeploymentStart.Format(time.RFC3339),
			"End Time":   deployment.DeploymentEnd.Format(time.RFC3339),
			"Duration":   duration.String(),
		},
	}

	builder := output.New().
		Table(
			"Deployment Summary",
			summaryData,
			output.WithKeys("Status", "Stack ARN", "Changeset", "Start Time", "End Time", "Duration"),
		)

	// Build planned changes table
	if deployment.CapturedChangeset != nil && len(deployment.CapturedChangeset.Changes) > 0 {
		changesData := make([]map[string]any, 0, len(deployment.CapturedChangeset.Changes))
		for _, change := range deployment.CapturedChangeset.Changes {
			changesData = append(changesData, map[string]any{
				"Action":      change.Action,
				"LogicalID":   change.LogicalID,
				"Type":        change.Type,
				"ResourceID":  change.ResourceID,
				"Replacement": change.Replacement,
			})
		}
		builder = builder.Table(
			"Planned Changes",
			changesData,
			output.WithKeys("Action", "LogicalID", "Type", "ResourceID", "Replacement"),
		)
	}

	// Build stack outputs table
	if len(deployment.FinalStackState.Outputs) > 0 {
		outputsData := make([]map[string]any, 0, len(deployment.FinalStackState.Outputs))
		for _, outputItem := range deployment.FinalStackState.Outputs {
			description := ""
			if outputItem.Description != nil {
				description = *outputItem.Description
			}
			outputsData = append(outputsData, map[string]any{
				"OutputKey":   aws.ToString(outputItem.OutputKey),
				"OutputValue": aws.ToString(outputItem.OutputValue),
				"Description": description,
			})
		}
		builder = builder.Table(
			"Stack Outputs",
			outputsData,
			output.WithKeys("OutputKey", "OutputValue", "Description"),
		)
	}

	// Render to stdout
	doc := builder.Build()
	out := output.NewOutput(settings.GetOutputOptions()...)
	return out.Render(context.Background(), doc)
}

// outputNoChangesResult outputs the no-changes message when CloudFormation determines
// there are no changes to apply.
func outputNoChangesResult(deployment *lib.DeployInfo) error {
	// Flush stderr before stdout output
	os.Stderr.Sync()
	fmt.Println("\n=== Deployment Summary ===")

	// Extract last updated time from RawStack
	var lastUpdated time.Time
	if deployment.RawStack != nil && deployment.RawStack.LastUpdatedTime != nil {
		lastUpdated = *deployment.RawStack.LastUpdatedTime
	} else if deployment.RawStack != nil && deployment.RawStack.CreationTime != nil {
		lastUpdated = *deployment.RawStack.CreationTime
	}

	message := "No changes to deploy - stack is already up to date"

	stackStatus := ""
	if deployment.RawStack != nil {
		stackStatus = string(deployment.RawStack.StackStatus)
	}

	stackData := []map[string]any{
		{
			"Stack Name":   deployment.StackName,
			"Status":       stackStatus,
			"ARN":          deployment.StackArn,
			"Last Updated": lastUpdated.Format(time.RFC3339),
		},
	}

	builder := output.New().
		Text(message).
		Table(
			"Stack Information",
			stackData,
			output.WithKeys("Stack Name", "Status", "ARN", "Last Updated"),
		)

	// Render to stdout
	doc := builder.Build()
	out := output.NewOutput(settings.GetOutputOptions()...)
	return out.Render(context.Background(), doc)
}

// FailedResource represents a resource that failed during deployment
type FailedResource struct {
	LogicalID      string
	ResourceStatus string
	StatusReason   string
	ResourceType   string
}

// extractFailedResources queries stack events to find resources with failed statuses
// and returns them as a slice of FailedResource structs.
func extractFailedResources(deployment *lib.DeployInfo, awsConfig config.AWSConfig) []FailedResource {
	events, err := deployment.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		// Return empty slice if events are unavailable
		return []FailedResource{}
	}

	failedResources := make([]FailedResource, 0)

	// Only look at events after changeset creation
	for _, event := range events {
		if deployment.CapturedChangeset != nil && event.Timestamp.After(deployment.CapturedChangeset.CreationTime) {
			// Check if this is a failed status
			switch event.ResourceStatus {
			case "CREATE_FAILED", "UPDATE_FAILED", "DELETE_FAILED", "IMPORT_FAILED":
				failedResource := FailedResource{
					LogicalID:      aws.ToString(event.LogicalResourceId),
					ResourceStatus: string(event.ResourceStatus),
					StatusReason:   aws.ToString(event.ResourceStatusReason),
					ResourceType:   aws.ToString(event.ResourceType),
				}
				failedResources = append(failedResources, failedResource)
			}
		}
	}

	return failedResources
}

// outputFailureResult outputs deployment failure details when a deployment fails.
// It includes error messages, stack status, and information about failed resources.
func outputFailureResult(deployment *lib.DeployInfo, awsConfig config.AWSConfig) error {
	// Flush stderr before stdout output
	os.Stderr.Sync()
	fmt.Println("\n=== Deployment Summary ===")

	errorMessage := "Deployment failed"
	if deployment.DeploymentError != nil {
		errorMessage = deployment.DeploymentError.Error()
	}

	stackStatus := ""
	statusReason := ""
	if deployment.FinalStackState != nil {
		stackStatus = string(deployment.FinalStackState.StackStatus)
		statusReason = aws.ToString(deployment.FinalStackState.StackStatusReason)
	}

	// Use DeploymentEnd if available, otherwise current time
	timestamp := time.Now()
	if !deployment.DeploymentEnd.IsZero() {
		timestamp = deployment.DeploymentEnd
	}

	// Build stack status table
	statusData := []map[string]any{
		{
			"Stack ARN":     deployment.StackArn,
			"Status":        stackStatus,
			"Status Reason": statusReason,
			"Timestamp":     timestamp.Format(time.RFC3339),
		},
	}

	builder := output.New().
		Text(fmt.Sprintf("Deployment failed: %s", errorMessage)).
		Table(
			"Stack Status",
			statusData,
			output.WithKeys("Stack ARN", "Status", "Status Reason", "Timestamp"),
		)

	// Extract and add failed resources table
	failedResources := extractFailedResources(deployment, awsConfig)
	if len(failedResources) > 0 {
		failedData := make([]map[string]any, 0, len(failedResources))
		for _, resource := range failedResources {
			failedData = append(failedData, map[string]any{
				"Logical ID": resource.LogicalID,
				"Type":       resource.ResourceType,
				"Status":     resource.ResourceStatus,
				"Reason":     resource.StatusReason,
			})
		}
		builder = builder.Table(
			"Failed Resources",
			failedData,
			output.WithKeys("Logical ID", "Type", "Status", "Reason"),
		)
	}

	// Render to stdout
	doc := builder.Build()
	out := output.NewOutput(settings.GetOutputOptions()...)
	return out.Render(context.Background(), doc)
}
