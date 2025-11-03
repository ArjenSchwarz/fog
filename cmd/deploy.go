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
	"context"
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
	output "github.com/ArjenSchwarz/go-output/v2"
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

var deployFlags DeployFlags
var deployment lib.DeployInfo

func init() {
	stackCmd.AddCommand(deployCmd)
	deployFlags.RegisterFlags(deployCmd)
}

func deployTemplate(cmd *cobra.Command, args []string) {
	viper.Set("output", "table") // Enforce table output for deployments

	deployment, awsConfig, err := prepareDeployment()
	if err != nil {
		printMessage(formatError(err.Error()))
		os.Exit(1)
	}

	deploymentLog := lib.NewDeploymentLog(awsConfig, deployment)

	precheckOutput := runPrechecks(&deployment, &deploymentLog)
	if precheckOutput != "" {
		printMessage(precheckOutput)
	}

	changeset := createAndShowChangeset(&deployment, awsConfig, &deploymentLog)
	if confirmAndDeployChangeset(changeset, &deployment, awsConfig) {
		printDeploymentResults(&deployment, awsConfig, &deploymentLog)
	}
}

// showDeploymentInfo shows what kind of deployment this (New/Update) and where it's happening
func showDeploymentInfo(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	bold := color.New(color.Bold).SprintFunc()
	method := determineDeploymentMethod(deployment.IsNew, deployFlags.Dryrun)
	account := formatAccountDisplay(awsConfig.AccountID, awsConfig.AccountAlias)

	if deployment.IsNew {
		fmt.Printf("%v new stack '%v' to region %v of account %v\n\n", method, bold(deployFlags.StackName), awsConfig.Region, account)
	} else {
		fmt.Printf("%v stack '%v' in region %v of account %v\n\n", method, bold(deployFlags.StackName), awsConfig.Region, awsConfig.AccountID)
	}
	printBasicStackInfo(deployment, true, awsConfig)
}

func setDeployTemplate(deployment *lib.DeployInfo, awsConfig config.AWSConfig) {
	var template string
	var path string
	var err error
	if deployment.StackDeploymentFile != nil {
		// The deployment file has the path relative to that file
		template, path, err = lib.ReadFile(&deployment.StackDeploymentFile.TemplateFilePath, "templates")
	} else {
		template, path, err = lib.ReadTemplate(&deployFlags.Template)
	}
	deployment.TemplateRelativePath = path
	if err != nil {
		printMessage(formatError(string(texts.FileTemplateReadFailure)))
		log.Fatalln(err)
	}
	if deployFlags.Bucket != "" {
		objectname, err := lib.UploadTemplate(&deployFlags.Template, template, &deployFlags.Bucket, awsConfig.S3Client())
		if err != nil {
			printMessage(formatError("Failed to upload template to S3"))
			log.Fatalln(err)
		}
		url := fmt.Sprintf("https://%v.s3-%v.amazonaws.com/%v", deployFlags.Bucket, awsConfig.Region, objectname)
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
	if deployFlags.DefaultTags {
		for key, value := range viper.GetStringMapString("tags.default") {
			tag := types.Tag{
				Key:   aws.String(key),
				Value: aws.String(placeholderParser(value, deployment)),
			}
			tagresult = append(tagresult, tag)
		}
	}
	if deployment.StackDeploymentFile != nil {
		for key, value := range deployment.StackDeploymentFile.Tags {
			tag := types.Tag{
				Key:   aws.String(key),
				Value: aws.String(placeholderParser(value, deployment)),
			}
			tagresult = append(tagresult, tag)
		}
	} else if deployFlags.Tags != "" {
		for tagfile := range strings.SplitSeq(deployFlags.Tags, ",") {
			tags, _, err := lib.ReadTagsfile(tagfile)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileTagsReadFailure, tagfile)
				printMessage(formatError(message))
				log.Fatalln(err)
			}
			parsedtags, err := lib.ParseTagString(tags)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileTagsParseFailure, tagfile)
				printMessage(formatError(message))
				log.Fatalln(err)
			}
			tagresult = append(tagresult, parsedtags...)
		}
	}
	deployment.Tags = tagresult
}

func placeholderParser(value string, deployment *lib.DeployInfo) string {
	if deployment != nil {
		value = strings.ReplaceAll(value, "$TEMPLATEPATH", deployment.TemplateLocalPath)
	}
	// value = strings.Replace(value, "$CURRENTDIR", os.Di)
	value = strings.ReplaceAll(value, "$TIMESTAMP", time.Now().In(settings.GetTimezoneLocation()).Format("2006-01-02T15-04-05"))
	return value
}

func setDeployParameters(deployment *lib.DeployInfo) {
	parameterresult := make([]types.Parameter, 0)
	if deployment.StackDeploymentFile != nil {
		for key, value := range deployment.StackDeploymentFile.Parameters {
			parameter := types.Parameter{
				ParameterKey:   aws.String(key),
				ParameterValue: aws.String(value),
			}
			parameterresult = append(parameterresult, parameter)
		}
	} else if deployFlags.Parameters != "" {
		for parameterfile := range strings.SplitSeq(deployFlags.Parameters, ",") {
			parameters, _, err := lib.ReadParametersfile(parameterfile)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileParametersReadFailure, parameterfile)
				printMessage(formatError(message))
				log.Fatalln(err)
			}
			parsedparameters, err := lib.ParseParameterString(parameters)
			if err != nil {
				message := fmt.Sprintf("%v '%v'", texts.FileParametersParseFailure, parameterfile)
				printMessage(formatError(message))
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
		printMessage(formatInfo(text))
	}
	_, err := deployment.CreateChangeSet(awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError(string(texts.DeployChangesetMessageCreationFailed)))
		log.Fatalln(err)
	}
	changeset, err := deployment.WaitUntilChangesetDone(awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError(string(texts.DeployChangesetMessageCreationFailed)))
		log.Fatalln(err)
	}
	if changeset.Status != string(types.ChangeSetStatusCreateComplete) {
		// When the creation fails because there are no changes, say so and complete successfully
		if changeset.StatusReason == string(texts.DeployReceivedErrorMessagesNoChanges) || changeset.StatusReason == string(texts.DeployReceivedErrorMessagesNoUpdates) {
			message := fmt.Sprintf(string(texts.DeployChangesetMessageNoChanges), deployment.StackName)
			printMessage(formatSuccess(message))
			os.Exit(0)
		}
		// Otherwise, show the error and clean up
		printMessage(formatError(string(texts.DeployChangesetMessageCreationFailed)))
		fmt.Println(changeset.StatusReason)
		fmt.Printf("\n%v %v\n", texts.DeployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
		var deleteChangesetConfirmation bool
		if deployFlags.NonInteractive {
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
	switch {
	case deployFlags.Dryrun:
		printMessage(formatInfo(string(texts.DeployChangesetMessageDryrunDelete)))
	case deployFlags.NonInteractive:
		printMessage(formatInfo(string(texts.DeployChangesetMessageAutoDelete)))
	default:
		printMessage(formatSuccess(string(texts.DeployChangesetMessageWillDelete)))
	}
	deleteAttempt := deployment.Changeset.DeleteChangeset(awsConfig.CloudformationClient())
	if !deleteAttempt {
		printMessage(formatError(string(texts.DeployChangesetMessageDeleteFailed)))
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
	if deployFlags.Dryrun || deployFlags.NonInteractive {
		deleteStackConfirmation = true
	} else {
		deleteStackConfirmation = askForConfirmation("Do you want me to delete this empty stack for you?")
	}
	if deleteStackConfirmation {
		if !deployment.DeleteStack(awsConfig.CloudformationClient()) {
			printMessage(formatError("Something went wrong while trying to delete the stack. Please check manually."))
		} else {
			switch {
			case deployFlags.Dryrun:
				printMessage(formatInfo(string(texts.DeployStackMessageNewStackDryrunDelete)))
			case deployFlags.NonInteractive:
				printMessage(formatInfo(string(texts.DeployStackMessageNewStackAutoDelete)))
			default:
				printMessage(formatSuccess(string(texts.DeployStackMessageNewStackDeleteSuccess)))
			}
		}
	} else {
		fmt.Println("No problem. I have left the stack intact, please delete it manually once you're done.")
	}
}

func deployChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	if deployFlags.NonInteractive {
		printMessage(formatInfo(string(texts.DeployChangesetMessageAutoDeploy)))
	} else {
		printMessage(formatSuccess(string(texts.DeployChangesetMessageWillDeploy)))
	}
	err := deployment.Changeset.DeployChangeset(awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError("Could not execute changeset! See details below"))
		fmt.Println(err)
	}
	latest := deployment.Changeset.CreationTime
	time.Sleep(3 * time.Second)
	fmt.Println(formatBold("Showing the events for the deployment:"))
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
		printMessage(formatError("Something went wrong trying to get the events of the stack"))
		fmt.Println(err)
	}
	sort.Sort(ReverseEvents(events))
	for _, event := range events {
		if event.Timestamp.After(latest) {
			latest = *event.Timestamp
			message := fmt.Sprintf("%v: %v %v in status %v", event.Timestamp.In(settings.GetTimezoneLocation()).Format(time.RFC3339), *event.ResourceType, *event.LogicalResourceId, event.ResourceStatus)
			switch event.ResourceStatus {
			case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed, types.ResourceStatusImportRollbackComplete, types.ResourceStatus(types.StackStatusRollbackComplete), types.ResourceStatus(types.StackStatusUpdateRollbackComplete):
				// For streaming logs, just apply color without extra spacing
				fmt.Println(output.StyleWarning(message))
			case types.ResourceStatusCreateComplete, types.ResourceStatusImportComplete, types.ResourceStatusUpdateComplete, types.ResourceStatusDeleteComplete:
				// For streaming logs, just apply color without extra spacing
				fmt.Println(output.StylePositive(message))
			default:
				fmt.Println(message)
			}
		}
	}
	return latest
}

func showFailedEvents(deployment lib.DeployInfo, awsConfig config.AWSConfig, prefixMessage string) []map[string]any {
	events, err := deployment.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError("Something went wrong trying to get the events of the stack"))
		fmt.Println(err)
		return nil
	}
	changesetkeys := []string{"CfnName", "Type", "Status", "Reason"}
	changesettitle := fmt.Sprintf("Failed events in deployment of changeset %v", deployment.Changeset.Name)
	sort.Sort(ReverseEvents(events))
	result := make([]map[string]any, 0)
	for _, event := range events {
		if event.Timestamp.After(deployment.Changeset.CreationTime) {
			switch event.ResourceStatus {
			case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed:
				content := make(map[string]any)
				content["CfnName"] = *event.LogicalResourceId
				content["Type"] = *event.ResourceType
				content["Status"] = string(event.ResourceStatus)
				content["Reason"] = *event.ResourceStatusReason
				result = append(result, content)
			}
		}
	}

	// Build unified document with optional prefix message and failed events table
	if len(result) > 0 {
		builder := output.New()

		// Add prefix message if provided
		if prefixMessage != "" {
			builder = builder.Text(prefixMessage)
		}

		// Add the failed events table
		builder = builder.Table(
			changesettitle,
			result,
			output.WithKeys(changesetkeys...),
		)

		doc := builder.Build()
		out := output.NewOutput(settings.GetOutputOptions()...)
		if err := out.Render(context.Background(), doc); err != nil {
			fmt.Printf("ERROR: Failed to render failed events: %v\n", err)
		}
	} else if prefixMessage != "" {
		// If no failed events but we have a prefix message, still show it
		printMessage(prefixMessage)
	}

	return result
}

// ReverseEvents implements sort.Interface for reverse-chronological sorting of stack events
type ReverseEvents []types.StackEvent

func (a ReverseEvents) Len() int           { return len(a) }
func (a ReverseEvents) Less(i, j int) bool { return a[i].Timestamp.Before(*a[j].Timestamp) }
func (a ReverseEvents) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
