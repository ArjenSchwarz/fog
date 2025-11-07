package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/texts"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/fatih/color"
	"github.com/spf13/viper"
	"log"
)

// loadAWSConfig allows tests to replace the default config loader.
var loadAWSConfig = config.DefaultAwsConfig

// allow stubbing AWS API calls in tests
var getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
	return cfg.CloudformationClient()
}

var createChangesetFunc = createChangeset
var showChangesetFunc = showChangeset
var deleteChangesetFunc = deleteChangeset
var deployChangesetFunc = deployChangeset
var askForConfirmationFunc = askForConfirmation
var showFailedEventsFunc = showFailedEvents
var deleteStackIfNewFunc = deleteStackIfNew
var getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
	return info.GetFreshStack(svc)
}

// prepareDeployment validates flags, loads AWS configuration and
// populates a DeployInfo instance for the deployment.
func prepareDeployment() (lib.DeployInfo, config.AWSConfig, error) {
	if err := deployFlags.Validate(); err != nil {
		return lib.DeployInfo{}, config.AWSConfig{}, err
	}

	// Auto-enable non-interactive mode when quiet mode is enabled
	if deployFlags.Quiet {
		deployFlags.NonInteractive = true
	}
	deployment := lib.DeployInfo{StackName: deployFlags.StackName}
	deployment.ChangesetName = deployFlags.ChangesetName
	if deployment.ChangesetName == "" {
		deployment.ChangesetName = placeholderParser(viper.GetString("changeset.name-format"), &deployment)
	}

	awsCfg, err := loadAWSConfig(*settings)
	if err != nil {
		return lib.DeployInfo{}, config.AWSConfig{}, err
	}

	cfnClient := getCfnClient(awsCfg)
	deployment.IsNew = deployment.IsNewStack(cfnClient)
	if !deployment.IsNew {
		if err := validateStackReadiness(deployFlags.StackName, cfnClient); err != nil {
			return lib.DeployInfo{}, config.AWSConfig{}, err
		}
	}
	deployment.IsDryRun = deployFlags.Dryrun

	showDeploymentInfo(deployment, awsCfg, deployFlags.Quiet)
	if !deployment.IsNew {
		deploymentName := lib.GenerateDeploymentName(awsCfg, deployment.StackName)
		if settings.GetBool("logging.enabled") && settings.GetBool("logging.show-previous") {
			log := lib.GetLatestSuccessFulLogByDeploymentName(deploymentName)
			if log.DeploymentName != "" {
				printLog(log, formatInfo("Previous deployment found:"))
			}
		}
	}
	if deployFlags.DeploymentFile != "" {
		if err := deployment.LoadDeploymentFile(deployFlags.DeploymentFile); err != nil {
			return lib.DeployInfo{}, config.AWSConfig{}, err
		}
	}
	setDeployTemplate(&deployment, awsCfg)
	setDeployTags(&deployment)
	setDeployParameters(&deployment)

	return deployment, awsCfg, nil
}

// runPrechecks executes all prechecks for the deployment and updates
// the deployment log accordingly. The collected output is returned so
// the caller can decide how to display it.
func runPrechecks(info *lib.DeployInfo, logObj *lib.DeploymentLog) string {
	commands := viper.GetStringSlice("templates.prechecks")
	if len(commands) == 0 {
		return ""
	}
	var builder strings.Builder
	precheckMessage := fmt.Sprintf(string(texts.FilePrecheckStarted), len(commands))
	builder.WriteString("\n")
	builder.WriteString(formatInfo(precheckMessage))
	results, err := lib.RunPrechecks(info)
	if err != nil {
		builder.WriteString(formatError(err.Error()))
		return builder.String()
	}
	if info.PrechecksFailed {
		logObj.PreChecks = lib.DeploymentLogPreChecksFailed
		if viper.GetBool("templates.stop-on-failed-prechecks") {
			builder.WriteString(formatError(string(texts.FilePrecheckFailureStop)))
		} else {
			builder.WriteString(formatError(string(texts.FilePrecheckFailureContinue)))
		}
		for cmd, out := range results {
			builder.WriteString(formatBold(cmd))
			builder.WriteString("\n")
			builder.WriteString(out)
			builder.WriteString("\n")
		}
	} else {
		logObj.PreChecks = lib.DeploymentLogPreChecksPassed
		builder.WriteString(formatPositive(string(texts.FilePrecheckSuccess)))
	}
	return builder.String()
}

// createAndShowChangeset creates a change set, displays it and
// appends it to the deployment log.
func createAndShowChangeset(info *lib.DeployInfo, cfg config.AWSConfig, logObj *lib.DeploymentLog, quiet bool) *lib.ChangesetInfo {
	changeset := createChangesetFunc(info, cfg)
	logObj.AddChangeSet(changeset)

	// Capture changeset immediately for final output
	info.CapturedChangeset = changeset
	info.Changeset = changeset // Maintain existing field for backwards compatibility

	// Show changeset overview to stderr only when not in quiet mode
	if !quiet {
		showChangesetFunc(*changeset, *info, cfg)
	}

	return changeset
}

// confirmAndDeployChangeset asks for deployment confirmation and executes
// the deployment if approved. It returns true when the stack was actually
// deployed.
func confirmAndDeployChangeset(changeset *lib.ChangesetInfo, info *lib.DeployInfo, cfg config.AWSConfig) bool {
	var confirm bool
	if deployFlags.NonInteractive {
		confirm = true
	} else {
		confirm = askForConfirmationFunc(string(texts.DeployChangesetMessageDeployConfirm))
	}
	if confirm {
		deployChangesetFunc(*info, cfg)
		return true
	}
	deleteChangesetFunc(*info, cfg)
	return false
}

// printDeploymentResults fetches the final stack state and prints the
// results. Success or failure information is written to the deployment log.
func printDeploymentResults(info *lib.DeployInfo, cfg config.AWSConfig, logObj *lib.DeploymentLog) {
	svc := getCfnClient(cfg)
	resultStack, err := getFreshStackFunc(info, svc)
	if err != nil {
		printMessage(formatError(string(texts.DeployStackMessageRetrievePostFailed)))
		log.Fatalln(err.Error())
	}

	// Capture final stack state for output generation
	info.FinalStackState = &resultStack

	switch resultStack.StackStatus {
	case types.StackStatusCreateComplete, types.StackStatusUpdateComplete:
		logObj.Success()

		// Build document with success message and optional outputs table
		builder := output.New().
			Text(formatSuccess(string(texts.DeployStackMessageSuccess)))

		if len(resultStack.Outputs) > 0 {
			outputkeys := []string{"Key", "Value", "Description", "ExportName"}
			outputtitle := fmt.Sprintf("Outputs for stack %v", *resultStack.StackName)
			outputRows := make([]map[string]any, 0)
			for _, outputresult := range resultStack.Outputs {
				exportName := ""
				if outputresult.ExportName != nil {
					exportName = *outputresult.ExportName
				}
				description := ""
				if outputresult.Description != nil {
					description = *outputresult.Description
				}
				content := make(map[string]any)
				content["Key"] = *outputresult.OutputKey
				content["Value"] = *outputresult.OutputValue
				content["Description"] = description
				content["ExportName"] = exportName
				outputRows = append(outputRows, content)
			}
			// Add outputs table to the document
			builder = builder.Table(
				outputtitle,
				outputRows,
				output.WithKeys(outputkeys...),
			)
		}

		// Render everything together
		doc := builder.Build()
		out := output.NewOutput(settings.GetOutputOptions()...)
		if err := out.Render(context.Background(), doc); err != nil {
			fmt.Printf("ERROR: Failed to render outputs: %v\n", err)
		}

	case types.StackStatusRollbackComplete, types.StackStatusRollbackFailed, types.StackStatusUpdateRollbackComplete, types.StackStatusUpdateRollbackFailed:
		// Capture deployment error
		info.DeploymentError = fmt.Errorf("deployment failed with status: %s", resultStack.StackStatus)

		failures := showFailedEventsFunc(*info, cfg, formatError(string(texts.DeployStackMessageFailed)))
		logObj.Failed(failures)
		if info.IsNew {
			// double verify that the stack can be deleted
			deleteStackIfNewFunc(*info, cfg)
		}
	}
}

// validateStackReadiness checks if an existing stack is ready for updates.
// Returns an error if the stack is in a non-updateable state.
func validateStackReadiness(stackName string, client lib.CloudFormationDescribeStacksAPI) error {
	deployment := lib.DeployInfo{StackName: stackName}
	if ready, status := deployment.IsReadyForUpdate(client); !ready {
		return fmt.Errorf("the stack '%v' is currently in status %v and can't be updated", stackName, status)
	}
	return nil
}

// formatAccountDisplay formats account information with optional alias.
// Returns "alias (accountID)" if alias is present, otherwise just "accountID".
func formatAccountDisplay(accountID string, accountAlias string) string {
	if accountAlias != "" {
		return fmt.Sprintf("%v (%v)", accountAlias, accountID)
	}
	return accountID
}

// determineDeploymentMethod returns the deployment method description
// based on whether this is a new stack and if it's a dry run.
func determineDeploymentMethod(isNew bool, isDryrun bool) string {
	bold := color.New(color.Bold).SprintFunc()
	if isNew {
		if isDryrun {
			return fmt.Sprintf("Doing a %v for", bold("dry run"))
		}
		return "Deploying"
	}
	if isDryrun {
		return fmt.Sprintf("Doing a %v for updating", bold("dry run"))
	}
	return "Updating"
}
