package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/texts"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// loadAWSConfig allows tests to replace the default config loader.
var loadAWSConfig = config.DefaultAwsConfig

// allow stubbing AWS API calls in tests
var getCfnClient = func(cfg config.AWSConfig) lib.CloudFormationDescribeStacksAPI {
	return cfg.CloudformationClient()
}

var createChangesetFunc = createChangeset
var fetchChangesetFunc = fetchChangeset
var showChangesetFunc = showChangeset
var deleteChangesetFunc = deleteChangeset
var deployChangesetFunc = deployChangeset
var askForConfirmationFunc = askForConfirmation
var showFailedEventsFunc = showFailedEvents
var deleteStackIfNewFunc = deleteStackIfNew

// osExitFunc allows tests to intercept os.Exit calls in command handlers.
var osExitFunc = os.Exit

var getFreshStackFunc = func(info *lib.DeployInfo, svc lib.CloudFormationDescribeStacksAPI) (types.Stack, error) {
	return info.GetFreshStack(context.Background(), svc)
}

// prepareDeployment validates flags, loads AWS configuration and
// populates a DeployInfo instance for the deployment.
func prepareDeployment() (lib.DeployInfo, config.AWSConfig, error) {
	if err := deployFlags.Validate(); err != nil {
		return lib.DeployInfo{}, config.AWSConfig{}, err
	}

	ctx := context.Background()

	// Auto-enable non-interactive mode when quiet mode is enabled
	if deployFlags.Quiet {
		deployFlags.NonInteractive = true
	}
	deployment := lib.DeployInfo{StackName: deployFlags.StackName}
	deployment.ChangesetName = deployFlags.ChangesetName
	if deployment.ChangesetName == "" {
		deployment.ChangesetName = placeholderParser(viper.GetString("changeset.name-format"), &deployment)
	}

	awsCfg, err := loadAWSConfig(ctx, *settings)
	if err != nil {
		return lib.DeployInfo{}, config.AWSConfig{}, err
	}

	cfnClient := getCfnClient(awsCfg)
	deployment.IsNew = deployment.IsNewStack(ctx, cfnClient)
	if !deployment.IsNew {
		if err := validateStackReadiness(ctx, deployFlags.StackName, cfnClient); err != nil {
			return lib.DeployInfo{}, config.AWSConfig{}, err
		}
	}
	deployment.IsDryRun = deployFlags.Dryrun

	showDeploymentInfo(deployment, awsCfg, deployFlags.Quiet)
	if !deployment.IsNew && !deployFlags.Quiet {
		deploymentName := lib.GenerateDeploymentName(awsCfg, deployment.StackName)
		if settings.GetBool("logging.enabled") && settings.GetBool("logging.show-previous") {
			log := lib.GetLatestSuccessFulLogByDeploymentName(deploymentName)
			if log.DeploymentName != "" {
				printLog(log, true, formatInfo("Previous deployment found:"))
			}
		}
	}
	if deployFlags.DeploymentFile != "" {
		if err := deployment.LoadDeploymentFile(deployFlags.DeploymentFile); err != nil {
			return lib.DeployInfo{}, config.AWSConfig{}, err
		}
	}
	// When deploying an existing changeset, the template/tags/parameters are
	// already captured on the changeset itself so we skip loading them here.
	if !deployFlags.DeployChangeset {
		setDeployTemplate(&deployment, awsCfg)
		setDeployTags(&deployment)
		setDeployParameters(&deployment)
	}

	return deployment, awsCfg, nil
}

// runPrechecks executes all prechecks for the deployment and updates
// the deployment log accordingly. It returns the collected output and
// whether the deployment should be aborted (true when prechecks fail
// and stop-on-failed-prechecks is enabled).
func runPrechecks(info *lib.DeployInfo, logObj *lib.DeploymentLog) (string, bool) {
	commands := viper.GetStringSlice("templates.prechecks")
	if len(commands) == 0 {
		return "", false
	}
	var builder strings.Builder
	precheckMessage := fmt.Sprintf(string(texts.FilePrecheckStarted), len(commands))
	builder.WriteString("\n")
	builder.WriteString(formatInfo(precheckMessage))
	results, err := lib.RunPrechecks(info)
	if err != nil {
		// Treat execution/configuration errors as failed prechecks so
		// that the stop flag is honored and the log records the failure.
		info.PrechecksFailed = true
		logObj.PreChecks = lib.DeploymentLogPreChecksFailed
		builder.WriteString(formatError(err.Error()))
		stopOnFailed := viper.GetBool("templates.stop-on-failed-prechecks")
		if stopOnFailed {
			builder.WriteString(formatError(string(texts.FilePrecheckFailureStop)))
		} else {
			builder.WriteString(formatError(string(texts.FilePrecheckFailureContinue)))
		}
		return builder.String(), stopOnFailed
	}
	if info.PrechecksFailed {
		logObj.PreChecks = lib.DeploymentLogPreChecksFailed
		stopOnFailed := viper.GetBool("templates.stop-on-failed-prechecks")
		if stopOnFailed {
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
		return builder.String(), stopOnFailed
	}
	logObj.PreChecks = lib.DeploymentLogPreChecksPassed
	builder.WriteString(formatPositive(string(texts.FilePrecheckSuccess)))
	return builder.String(), false
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

// runDeployChangesetFlow retrieves an existing changeset (named by
// --changeset), attaches it to the deployment and displays it. It is the
// counterpart to createAndShowChangeset for the --deploy-changeset flow.
// If fetchChangesetFunc returns nil (e.g. a test stub or an error path that
// returned instead of exiting), the caller receives nil and should abort.
func runDeployChangesetFlow(info *lib.DeployInfo, cfg config.AWSConfig, logObj *lib.DeploymentLog, quiet bool) *lib.ChangesetInfo {
	changeset := fetchChangesetFunc(info, cfg)
	if changeset == nil {
		return nil
	}
	logObj.AddChangeSet(changeset)

	info.CapturedChangeset = changeset
	info.Changeset = changeset

	if !quiet {
		showChangesetFunc(*changeset, *info, cfg)
	}

	return changeset
}

// confirmAndDeployChangeset asks for deployment confirmation and executes
// the deployment if approved. It returns true when the stack was actually
// deployed and an error if the deployment execution failed.
func confirmAndDeployChangeset(changeset *lib.ChangesetInfo, info *lib.DeployInfo, cfg config.AWSConfig) (bool, error) {
	var confirm bool
	if deployFlags.NonInteractive {
		confirm = true
	} else {
		confirm = askForConfirmationFunc(string(texts.DeployChangesetMessageDeployConfirm))
	}
	if confirm {
		if err := deployChangesetFunc(*info, cfg); err != nil {
			return false, err
		}
		return true, nil
	}
	deleteChangesetFunc(*info, cfg)
	return false, nil
}

// printDeploymentResults fetches the final stack state and prints the
// results. Success or failure information is written to the deployment log.
func printDeploymentResults(info *lib.DeployInfo, cfg config.AWSConfig, logObj *lib.DeploymentLog) {
	// Set deployment end timestamp before generating output
	info.DeploymentEnd = time.Now()

	svc := getCfnClient(cfg)
	resultStack, err := getFreshStackFunc(info, svc)
	if err != nil {
		info.DeploymentError = fmt.Errorf("failed to retrieve final stack state: %w", err)
		logObj.StatusDescription = info.DeploymentError.Error()

		failures := []map[string]any{{
			"CfnName": info.StackName,
			"Type":    "AWS::CloudFormation::Stack",
			"Status":  "POST_DEPLOY_LOOKUP_FAILED",
			"Reason":  err.Error(),
		}}

		printMessage(formatError(string(texts.DeployStackMessageRetrievePostFailed)))
		if err := logObj.Failed(failures); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write deployment log: %v\n", err)
		}

		eventsClient, _ := svc.(lib.CloudFormationDescribeStackEventsAPI)
		if err := outputFailureResult(info, eventsClient); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)
		}
		return
	}

	// Capture final stack state for output generation
	info.FinalStackState = &resultStack

	switch resultStack.StackStatus {
	case types.StackStatusCreateComplete, types.StackStatusUpdateComplete:
		if err := logObj.Success(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write deployment log: %v\n", err)
		}

		// Output success message to stderr (streaming output)
		if !deployFlags.Quiet {
			printMessage(formatSuccess(string(texts.DeployStackMessageSuccess)))
		}

		// Output final deployment summary to stdout
		if err := outputSuccessResult(info); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)
		}

	case types.StackStatusRollbackComplete, types.StackStatusRollbackFailed, types.StackStatusUpdateRollbackComplete, types.StackStatusUpdateRollbackFailed:
		// Capture deployment error
		info.DeploymentError = fmt.Errorf("deployment failed with status: %s", resultStack.StackStatus)

		// Show failed events to stderr (streaming output)
		failures := showFailedEventsFunc(*info, cfg, formatError(string(texts.DeployStackMessageFailed)))
		if err := logObj.Failed(failures); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write deployment log: %v\n", err)
		}

		// Output failure summary to stdout
		if err := outputFailureResult(info, cfg.CloudformationClient()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)
		}

		if info.IsNew {
			// double verify that the stack can be deleted
			deleteStackIfNewFunc(*info, cfg)
		}
	}
}

// validateStackReadiness checks if an existing stack is ready for updates.
// Returns an error if the stack is in a non-updateable state.
func validateStackReadiness(ctx context.Context, stackName string, client lib.CloudFormationDescribeStacksAPI) error {
	deployment := lib.DeployInfo{StackName: stackName}
	if ready, status := deployment.IsReadyForUpdate(ctx, client); !ready {
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
