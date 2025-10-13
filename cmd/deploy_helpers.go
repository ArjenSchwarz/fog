package cmd

import (
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/texts"
	format "github.com/ArjenSchwarz/go-output"
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

	showDeploymentInfo(deployment, awsCfg)
	if !deployment.IsNew {
		deploymentName := lib.GenerateDeploymentName(awsCfg, deployment.StackName)
		if settings.GetBool("logging.enabled") && settings.GetBool("logging.show-previous") {
			log := lib.GetLatestSuccessFulLogByDeploymentName(deploymentName)
			if log.DeploymentName != "" {
				fmt.Print(outputsettings.StringInfo("Previous deployment found:"))
				printLog(log)
				// Hack to print the buffer in printLog. Need to get a better solution.
				output := format.OutputArray{Keys: []string{}, Settings: settings.NewOutputSettings()}
				output.Write()
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
	builder.WriteString(outputsettings.StringInfo(precheckMessage))
	results, err := lib.RunPrechecks(info)
	if err != nil {
		builder.WriteString(outputsettings.StringFailure(err.Error()))
		return builder.String()
	}
	if info.PrechecksFailed {
		logObj.PreChecks = lib.DeploymentLogPreChecksFailed
		if viper.GetBool("templates.stop-on-failed-prechecks") {
			builder.WriteString(outputsettings.StringFailure(texts.FilePrecheckFailureStop))
		} else {
			builder.WriteString(outputsettings.StringFailure(texts.FilePrecheckFailureContinue))
		}
		for cmd, out := range results {
			builder.WriteString(outputsettings.StringBold(cmd))
			builder.WriteString("\n")
			builder.WriteString(out)
			builder.WriteString("\n")
		}
	} else {
		logObj.PreChecks = lib.DeploymentLogPreChecksPassed
		builder.WriteString(outputsettings.StringPositive(string(texts.FilePrecheckSuccess)))
	}
	return builder.String()
}

// createAndShowChangeset creates a change set, displays it and
// appends it to the deployment log. When running in dry-run mode the
// change set is deleted again.
func createAndShowChangeset(info *lib.DeployInfo, cfg config.AWSConfig, logObj *lib.DeploymentLog) *lib.ChangesetInfo {
	changeset := createChangesetFunc(info, cfg)
	logObj.AddChangeSet(changeset)
	showChangesetFunc(*changeset, *info, cfg)
	if info.IsDryRun {
		fmt.Print(outputsettings.StringSuccess(texts.DeployChangesetMessageDryrunSuccess))
		deleteChangesetFunc(*info, cfg)
	}
	return changeset
}

// confirmAndDeployChangeset asks for deployment confirmation and executes
// the deployment if approved. It returns true when the stack was actually
// deployed.
func confirmAndDeployChangeset(changeset *lib.ChangesetInfo, info *lib.DeployInfo, cfg config.AWSConfig) bool {
	if deployFlags.CreateChangeset {
		fmt.Print(outputsettings.StringSuccess(texts.DeployChangesetMessageSuccess))
		fmt.Print(outputsettings.StringInfo("Only created the change set, will now terminate"))
		return false
	}
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
		fmt.Print(outputsettings.StringFailure(texts.DeployStackMessageRetrievePostFailed))
		log.Fatalln(err.Error())
	}
	switch resultStack.StackStatus {
	case types.StackStatusCreateComplete, types.StackStatusUpdateComplete:
		logObj.Success()
		fmt.Print(outputsettings.StringSuccess(texts.DeployStackMessageSuccess))
		if len(resultStack.Outputs) > 0 {
			outputkeys := []string{"Key", "Value", "Description", "ExportName"}
			outputtitle := fmt.Sprintf("Outputs for stack %v", *resultStack.StackName)
			output := format.OutputArray{Keys: outputkeys, Settings: outputsettings}
			output.Settings.Title = outputtitle
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
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
			output.Write()
		}
	case types.StackStatusRollbackComplete, types.StackStatusRollbackFailed, types.StackStatusUpdateRollbackComplete, types.StackStatusUpdateRollbackFailed:
		fmt.Print(outputsettings.StringFailure(texts.DeployStackMessageFailed))
		failures := showFailedEventsFunc(*info, cfg)
		logObj.Failed(failures)
		if info.IsNew {
			//double verify that the stack can be deleted
			deleteStackIfNewFunc(*info, cfg)
		}
	}
}

// validateStackReadiness checks if an existing stack is ready for updates.
// Returns an error if the stack is in a non-updateable state.
func validateStackReadiness(stackName string, client lib.CloudFormationDescribeStacksAPI) error {
	deployment := lib.DeployInfo{StackName: stackName}
	if ready, status := deployment.IsReadyForUpdate(client); !ready {
		return fmt.Errorf("The stack '%v' is currently in status %v and can't be updated", stackName, status)
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
