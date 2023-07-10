/*
Copyright © 2021 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/viper"

	"path/filepath"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/texts"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a CloudFormation stack",
	Long: `deploy allows you to deploy a CloudFormation stack

It does so by creating a ChangeSet and then asking you for approval before continuing. You can automatically approve or only create or deploy a changeset by using flags.

A name for the changeset will automatically be generated based on your preferred name, but can be overwritten as well.

When providing tag and/or parameter files, you can add multiple files for each. These are parsed in the order provided and later values will override earlier ones.

Examples:

  fog deploy --stackname testvpc --template basicvpc --parameters vpc-private-only --tags "../globaltags/project,dev"
  fog deploy --stackname fails3 --template fails3 --non-interactive
  fog deploy --stackname myvpc --template basicvpc --parameters vpc-public --tags "../globaltags/project,dev" --config testconf/fog.yaml
`,
	Run: deployTemplate,
}

var deploy_StackName *string
var deploy_Template *string
var deploy_Parameters *string
var deploy_Bucket *string
var deploy_Tags *string
var deploy_ChangesetName *string
var deploy_Dryrun *bool
var deploy_NonInteractive *bool
var deploy_CreateChangeset *bool
var deploy_DeployChangeset *bool
var deploy_DefaultTags *bool
var deployment lib.DeployInfo

func init() {
	rootCmd.AddCommand(deployCmd)
	deploy_StackName = deployCmd.Flags().StringP("stackname", "n", "", "The name for the stack")
	deploy_Template = deployCmd.Flags().StringP("template", "f", "", "The filename for the template")
	deploy_Parameters = deployCmd.Flags().StringP("parameters", "p", "", "The file(s) containing the parameter values, comma-separated for multiple")
	deploy_Tags = deployCmd.Flags().StringP("tags", "t", "", "The file(s) containing the tags, comma-separated for multiple")
	deploy_Bucket = deployCmd.Flags().StringP("bucket", "b", "", "The S3 bucket where the template should be uploaded to (optional)")
	deploy_ChangesetName = deployCmd.Flags().StringP("changeset", "c", "", "The name of the changeset, when not provided it will be autogenerated")
	deploy_Dryrun = deployCmd.Flags().Bool("dry-run", false, "Do a dry run: create the changeset and immediately delete")
	deploy_NonInteractive = deployCmd.Flags().Bool("non-interactive", false, "Run in non-interactive mode: automatically approve the changeset and deploy")
	deploy_CreateChangeset = deployCmd.Flags().Bool("create-changeset", false, "Only create a change set")
	deploy_DeployChangeset = deployCmd.Flags().Bool("deploy-changeset", false, "Deploy a specific change set")
	deploy_DefaultTags = deployCmd.Flags().Bool("default-tags", true, "Add any default tags that are specified in your config file")
}

func deployTemplate(cmd *cobra.Command, args []string) {
	viper.Set("output", "table") //Enforce table output for deployments
	outputsettings = settings.NewOutputSettings()
	outputsettings.SeparateTables = true //Make table output stand out more
	deployment.StackName = *deploy_StackName
	// Set the changeset name to what's provided, otherwise fall back on the generated value
	deployment.ChangesetName = *deploy_ChangesetName
	if deployment.ChangesetName == "" {
		deployment.ChangesetName = placeholderParser(viper.GetString("changeset.name-format"), &deployment)
	}
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	deployment.IsNew = deployment.IsNewStack(awsConfig.CloudformationClient())
	if !deployment.IsNew {
		if ready, status := deployment.IsReadyForUpdate(awsConfig.CloudformationClient()); !ready {
			message := fmt.Sprintf("The stack '%v' is currently in status %v and can't be updated", *deploy_StackName, status)
			fmt.Print(outputsettings.StringFailure(message))
			os.Exit(1)
		}
	}
	deployment.IsDryRun = *deploy_Dryrun
	showDeploymentInfo(deployment, awsConfig)
	if !deployment.IsNew {
		deploymentName := lib.GenerateDeploymentName(awsConfig, deployment.StackName)
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
	deploymentLog := lib.NewDeploymentLog(awsConfig, deployment)
	if *deploy_DeployChangeset {
		rawchangeset, err := deployment.GetChangeset(awsConfig.CloudformationClient())
		if err != nil {
			message := fmt.Sprintf(string(texts.DeployChangesetMessageRetrieveFailed), deployment.ChangesetName)
			fmt.Print(outputsettings.StringFailure(message))
			os.Exit(1)
		}
		changeset := deployment.AddChangeset(rawchangeset)
		deploymentLog.AddChangeSet(&changeset)
		showChangeset(changeset, deployment, awsConfig)
	} else {
		setDeployTemplate(&deployment, awsConfig)
		setDeployTags(&deployment)
		setDeployParameters(&deployment)
		if viper.GetStringSlice("templates.prechecks") != nil {
			precheckmessage := fmt.Sprintf(string(texts.FilePrecheckStarted), len(viper.GetStringSlice("templates.prechecks")))
			fmt.Print(outputsettings.StringInfo(precheckmessage))
			precheckresults, err := lib.RunPrechecks(&deployment)
			if err != nil {
				fmt.Print(outputsettings.StringFailure(err))
			}
			if deployment.PrechecksFailed {
				if viper.GetBool("templates.stop-on-failed-prechecks") {
					fmt.Print(outputsettings.StringFailure(texts.FilePrecheckFailureStop))
					for command, output := range precheckresults {
						fmt.Print(outputsettings.StringBold(command))
						fmt.Println(output)
					}
					os.Exit(1)
				}
				for command, output := range precheckresults {
					fmt.Print(outputsettings.StringBold(command))
					fmt.Println(output)
				}
				deploymentLog.PreChecks = lib.DeploymentLogPreChecksFailed
				fmt.Print(outputsettings.StringFailure(texts.FilePrecheckFailureContinue))
			} else {
				deploymentLog.PreChecks = lib.DeploymentLogPreChecksPassed
				fmt.Print(outputsettings.StringPositive(string(texts.FilePrecheckSuccess)))
			}
		}
		changeset := createChangeset(&deployment, awsConfig)
		deploymentLog.AddChangeSet(changeset)
		showChangeset(*changeset, deployment, awsConfig)
		if *deploy_Dryrun {
			fmt.Print(outputsettings.StringSuccess(texts.DeployChangesetMessageDryrunSuccess))
			deleteChangeset(deployment, awsConfig)
			os.Exit(0)
		}
		if *deploy_CreateChangeset {
			fmt.Print(outputsettings.StringSuccess(texts.DeployChangesetMessageSuccess))
			fmt.Print(outputsettings.StringInfo("Only created the change set, will now terminate"))
			os.Exit(0)
		}
	}
	var deployChangesetConfirmation bool
	if *deploy_NonInteractive {
		deployChangesetConfirmation = true
	} else {
		deployChangesetConfirmation = askForConfirmation(string(texts.DeployChangesetMessageDeployConfirm))
	}
	if deployChangesetConfirmation {
		deployChangeset(deployment, awsConfig)
	} else {
		deleteChangeset(deployment, awsConfig)
		os.Exit(0)
	}
	resultStack, err := deployment.GetFreshStack(awsConfig.CloudformationClient())
	if err != nil {
		fmt.Print(outputsettings.StringFailure(texts.DeployStackMessageRetrievePostFailed))
		log.Fatalln(err.Error())
	}
	switch resultStack.StackStatus {
	case types.StackStatusCreateComplete, types.StackStatusUpdateComplete:
		deploymentLog.Success()
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
				content := make(map[string]interface{})
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
		failures := showFailedEvents(deployment, awsConfig)
		deploymentLog.Failed(failures)
		if deployment.IsNew {
			//double verify that the stack can be deleted
			deleteStackIfNew(deployment, awsConfig)
		}
	}
}

// showDeploymentInfo shows what kind of deployment this (New/Update) and where it's happening
func showDeploymentInfo(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	bold := color.New(color.Bold).SprintFunc()
	if deployment.IsNew {
		method := "Deploying"
		if *deploy_Dryrun {
			method = fmt.Sprintf("Doing a %v for", bold("dry run"))
		}
		account := awsConfig.AccountID
		if awsConfig.AccountAlias != "" {
			account = fmt.Sprintf("%v (%v)", awsConfig.AccountAlias, awsConfig.AccountID)
		}
		fmt.Printf("%v new stack '%v' to region %v of account %v\n", method, bold(*deploy_StackName), awsConfig.Region, account)
	} else {
		method := "Updating"
		if *deploy_Dryrun {
			method = fmt.Sprintf("Doing a %v for updating", bold("dry run"))
		}
		fmt.Printf("%v stack '%v' in region %v of account %v\n", method, bold(*deploy_StackName), awsConfig.Region, awsConfig.AccountID)
	}
	printBasicStackInfo(deployment, true, awsConfig)
}

func setDeployTemplate(deployment *lib.DeployInfo, awsConfig config.AWSConfig) {
	template, path, err := lib.ReadTemplate(deploy_Template)
	deployment.TemplateRelativePath = path
	if err != nil {
		fmt.Print(outputsettings.StringFailure(texts.FileTemplateReadFailure))
		log.Fatalln(err)
	}
	if *deploy_Bucket != "" {
		objectname, err := lib.UploadTemplate(deploy_Template, template, deploy_Bucket, awsConfig.S3Client())
		if err != nil {
			fmt.Print(outputsettings.StringFailure("this failed"))
			log.Fatalln(err)
		}
		url := fmt.Sprintf("https://%v.s3-%v.amazonaws.com/%v", *deploy_Bucket, awsConfig.Region, objectname)
		deployment.TemplateUrl = url
	}
	// Use the root path to correctly get the relative path of the templates
	if cfgFile != "" {
		confdir := filepath.Dir(cfgFile)
		confpath, _ := filepath.Abs(fmt.Sprintf("%s%s%s", confdir, string(os.PathSeparator), viper.GetString("rootdir")))
		localpath, _ := filepath.Abs(path)
		path, _ = filepath.Rel(confpath, localpath)
	} else {
		confpath, _ := filepath.Abs(viper.GetString("rootdir"))
		localpath, _ := filepath.Abs(path)
		path, _ = filepath.Rel(confpath, localpath)
	}
	deployment.TemplateLocalPath = path
	deployment.Template = template
}

func setDeployTags(deployment *lib.DeployInfo) {
	tagresult := make([]types.Tag, 0)
	if *deploy_DefaultTags {
		for key, value := range viper.GetStringMapString("tags.default") {
			tag := types.Tag{
				Key:   aws.String(key),
				Value: aws.String(placeholderParser(value, deployment)),
			}
			tagresult = append(tagresult, tag)
		}
	}

	if *deploy_Tags != "" {
		for _, tagfile := range strings.Split(*deploy_Tags, ",") {
			tags, _, err := lib.ReadTagsfile(tagfile)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileTagsReadFailure, tagfile)
				fmt.Print(outputsettings.StringFailure(message))
				log.Fatalln(err)
			}
			parsedtags, err := lib.ParseTagString(tags)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileTagsParseFailure, tagfile)
				fmt.Print(outputsettings.StringFailure(message))
				log.Fatalln(err)
			}
			tagresult = append(tagresult, parsedtags...)
		}
	}
	deployment.Tags = tagresult
}

func placeholderParser(value string, deployment *lib.DeployInfo) string {
	if deployment != nil {
		value = strings.Replace(value, "$TEMPLATEPATH", deployment.TemplateLocalPath, -1)
	}
	//value = strings.Replace(value, "$CURRENTDIR", os.Di)
	value = strings.Replace(value, "$TIMESTAMP", time.Now().In(settings.GetTimezoneLocation()).Format("2006-01-02T15-04-05"), -1)
	return value
}

func setDeployParameters(deployment *lib.DeployInfo) {
	parameterresult := make([]types.Parameter, 0)
	if *deploy_Parameters != "" {
		for _, parameterfile := range strings.Split(*deploy_Parameters, ",") {
			parameters, _, err := lib.ReadParametersfile(parameterfile)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileParametersReadFailure, parameterfile)
				fmt.Print(outputsettings.StringFailure(message))
				log.Fatalln(err)
			}
			parsedparameters, err := lib.ParseParameterString(parameters)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileParametersParseFailure, parameterfile)
				fmt.Print(outputsettings.StringFailure(message))
				log.Fatalln(err)
			}
			parameterresult = append(parameterresult, parsedparameters...)
		}
	}
	deployment.Parameters = parameterresult
}

func createChangeset(deployment *lib.DeployInfo, awsConfig config.AWSConfig) *lib.ChangesetInfo {
	if deployment.TemplateUrl != "" {
		text := fmt.Sprintf("Using template uploaded as %v", deployment.TemplateUrl)
		fmt.Print(outputsettings.StringInfo(text))
	}
	_, err := deployment.CreateChangeSet(awsConfig.CloudformationClient())
	if err != nil {
		fmt.Print(outputsettings.StringFailure(texts.DeployChangesetMessageCreationFailed))
		log.Fatalln(err)
	}
	changeset, err := deployment.WaitUntilChangesetDone(awsConfig.CloudformationClient())
	if err != nil {
		fmt.Print(outputsettings.StringFailure(texts.DeployChangesetMessageCreationFailed))
		log.Fatalln(err)
	}
	if changeset.Status != string(types.ChangeSetStatusCreateComplete) {
		// When the creation fails because there are no changes, say so and complete successfully
		if changeset.StatusReason == string(texts.DeployReceivedErrorMessagesNoChanges) || changeset.StatusReason == string(texts.DeployReceivedErrorMessagesNoUpdates) {
			message := fmt.Sprintf(string(texts.DeployChangesetMessageNoChanges), deployment.StackName)
			fmt.Print(outputsettings.StringSuccess(message))
			os.Exit(0)
		}
		// Otherwise, show the error and clean up
		fmt.Print(outputsettings.StringFailure(texts.DeployChangesetMessageCreationFailed))
		fmt.Println(changeset.StatusReason)
		fmt.Printf("\r\n%v %v \r\n", texts.DeployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
		var deleteChangesetConfirmation bool
		if *deploy_NonInteractive {
			deleteChangesetConfirmation = true
		} else {
			askForConfirmation(string(texts.DeployChangesetMessageDeleteConfirm))
		}
		if deleteChangesetConfirmation {
			deleteChangeset(*deployment, awsConfig)
		}
		os.Exit(1)
	}
	return changeset
}

func deleteChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	if *deploy_Dryrun {
		fmt.Print(outputsettings.StringInfo(texts.DeployChangesetMessageDryrunDelete))
	} else if *deploy_NonInteractive {
		fmt.Print(outputsettings.StringInfo(texts.DeployChangesetMessageAutoDelete))
	} else {
		fmt.Print(outputsettings.StringSuccess(texts.DeployChangesetMessageWillDelete))
	}
	deleteAttempt := deployment.Changeset.DeleteChangeset(awsConfig.CloudformationClient())
	if !deleteAttempt {
		fmt.Print(outputsettings.StringFailure(texts.DeployChangesetMessageDeleteFailed))
	}
	// Likely a new deployment. Check if the stack is in status REVIEW_IN_PROGRESS and offer to delete
	if deployment.IsNew {
		stack, err := deployment.GetFreshStack(awsConfig.CloudformationClient())
		if err != nil {
			log.Fatalln(err)
		}
		if stack.StackStatus == types.StackStatusReviewInProgress {
			deleteStackIfNew(deployment, awsConfig)
		}
	}
}

func deleteStackIfNew(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	fmt.Println(texts.DeployStackMessageNewStackDeleteInfo)
	var deleteStackConfirmation bool
	if *deploy_Dryrun || *deploy_NonInteractive {
		deleteStackConfirmation = true
	} else {
		deleteStackConfirmation = askForConfirmation("Do you want me to delete this empty stack for you?")
	}
	if deleteStackConfirmation {
		if !deployment.DeleteStack(awsConfig.CloudformationClient()) {
			fmt.Print(outputsettings.StringFailure("Something went wrong while trying to delete the stack. Please check manually."))
		} else {
			if *deploy_Dryrun {
				fmt.Print(outputsettings.StringInfo(texts.DeployStackMessageNewStackDryrunDelete))
			} else if *deploy_NonInteractive {
				fmt.Print(outputsettings.StringInfo(texts.DeployStackMessageNewStackAutoDelete))
			} else {
				fmt.Print(outputsettings.StringSuccess(texts.DeployStackMessageNewStackDeleteSuccess))
			}
		}
	} else {
		fmt.Println("No problem. I have left the stack intact, please delete it manually once you're done.")
	}
}

func deployChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	if *deploy_NonInteractive {
		fmt.Print(outputsettings.StringInfo(texts.DeployChangesetMessageAutoDeploy))
	} else {
		fmt.Print(outputsettings.StringSuccess(texts.DeployChangesetMessageWillDeploy))
	}
	err := deployment.Changeset.DeployChangeset(awsConfig.CloudformationClient())
	if err != nil {
		fmt.Print(outputsettings.StringFailure("Could not execute changeset! See details below"))
		fmt.Println(err)
	}
	latest := deployment.Changeset.CreationTime
	time.Sleep(3 * time.Second)
	fmt.Print(outputsettings.StringBold("Showing the events for the deployment:"))
	ongoing := true
	for ongoing {
		latest = showEvents(deployment, latest, awsConfig)
		time.Sleep(3 * time.Second)
		ongoing = deployment.IsOngoing(awsConfig.CloudformationClient())
	}
	// One last time after the deployment finished in case of a timing mismatch
	showEvents(deployment, latest, awsConfig)
}

func showEvents(deployment lib.DeployInfo, latest time.Time, awsConfig config.AWSConfig) time.Time {
	events, err := deployment.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		fmt.Print(outputsettings.StringFailure("Something went wrong trying to get the events of the stack"))
		fmt.Println(err)
	}
	sort.Sort(ReverseEvents(events))
	for _, event := range events {
		if event.Timestamp.After(latest) {
			latest = *event.Timestamp
			message := fmt.Sprintf("%v: %v %v in status %v", event.Timestamp.In(settings.GetTimezoneLocation()).Format(time.RFC3339), *event.ResourceType, *event.LogicalResourceId, event.ResourceStatus)
			switch event.ResourceStatus {
			case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed, types.ResourceStatusImportRollbackComplete, types.ResourceStatus(types.StackStatusRollbackComplete), types.ResourceStatus(types.StackStatusUpdateRollbackComplete):
				fmt.Print(outputsettings.StringWarning(message))
			case types.ResourceStatusCreateComplete, types.ResourceStatusImportComplete, types.ResourceStatusUpdateComplete, types.ResourceStatusDeleteComplete:
				fmt.Print(outputsettings.StringPositive(message))
			default:
				fmt.Println(message)
			}
		}
	}
	return latest
}

func showFailedEvents(deployment lib.DeployInfo, awsConfig config.AWSConfig) []map[string]interface{} {
	events, err := deployment.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		fmt.Print(outputsettings.StringFailure("Something went wrong trying to get the events of the stack"))
		fmt.Println(err)
	}
	changesetkeys := []string{"CfnName", "Type", "Status", "Reason"}
	changesettitle := fmt.Sprintf("Failed events in deployment of changeset %v", deployment.Changeset.Name)
	output := format.OutputArray{Keys: changesetkeys, Settings: outputsettings}
	output.Settings.Title = changesettitle
	sort.Sort(ReverseEvents(events))
	result := make([]map[string]interface{}, 0)
	for _, event := range events {
		if event.Timestamp.After(deployment.Changeset.CreationTime) {
			switch event.ResourceStatus {
			case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed:
				content := make(map[string]interface{})
				content["CfnName"] = *event.LogicalResourceId
				content["Type"] = *event.ResourceType
				content["Status"] = string(event.ResourceStatus)
				content["Reason"] = *event.ResourceStatusReason
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
				result = append(result, content)
			}
		}
	}
	output.Write()
	return result
}

type ReverseEvents []types.StackEvent

func (a ReverseEvents) Len() int           { return len(a) }
func (a ReverseEvents) Less(i, j int) bool { return a[i].Timestamp.Before(*a[j].Timestamp) }
func (a ReverseEvents) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
