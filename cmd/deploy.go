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
	"github.com/mattn/go-isatty"
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
	deployment, awsConfig, err := prepareDeployment()
	if err != nil {
		printMessage(formatError(err.Error()))
		os.Exit(1)
	}

	// Capture deployment start timestamp before any AWS operations
	deployment.DeploymentStart = time.Now()

	deploymentLog := lib.NewDeploymentLog(awsConfig, deployment)

	precheckOutput, abort := runPrechecks(&deployment, &deploymentLog)
	if precheckOutput != "" {
		printMessage(precheckOutput)
	}
	if abort {
		if err := deploymentLog.Failed(nil); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to write deployment log: %v\n", err)
		}
		osExitFunc(1)
		return
	}

	var changeset *lib.ChangesetInfo
	if deployFlags.DeployChangeset {
		// Deploy an existing, previously-created changeset by name.
		changeset = runDeployChangesetFlow(&deployment, awsConfig, &deploymentLog, deployFlags.Quiet)
	} else {
		changeset = createAndShowChangeset(&deployment, awsConfig, &deploymentLog, deployFlags.Quiet)

		// Handle dry-run mode
		if deployment.IsDryRun {
			outputDryRunResult(&deployment, awsConfig)
			// Delete changeset after output for dry-run
			deleteChangeset(deployment, awsConfig)
			return
		}

		// Handle create-changeset mode
		if deployFlags.CreateChangeset {
			outputDryRunResult(&deployment, awsConfig)
			// Do NOT delete changeset for --create-changeset mode
			return
		}
	}

	deployed, err := confirmAndDeployChangeset(changeset, &deployment, awsConfig)
	if err != nil {
		printMessage(formatError(err.Error()))
		if deployment.IsNew {
			deleteStackIfNewFunc(deployment, awsConfig)
		}
		os.Exit(1)
	}
	if deployed {
		printDeploymentResults(&deployment, awsConfig, &deploymentLog)
	}
}

// showDeploymentInfo shows what kind of deployment this (New/Update) and where it's happening
func showDeploymentInfo(deployment lib.DeployInfo, awsConfig config.AWSConfig, quiet bool) {
	// Return early if quiet mode is enabled
	if quiet {
		return
	}

	bold := color.New(color.Bold).SprintFunc()
	method := determineDeploymentMethod(deployment.IsNew, deployFlags.Dryrun)
	account := formatAccountDisplay(awsConfig.AccountID, awsConfig.AccountAlias)

	if deployment.IsNew {
		fmt.Fprintf(os.Stderr, "%v new stack '%v' to region %v of account %v\n\n", method, bold(deployFlags.StackName), awsConfig.Region, account)
	} else {
		fmt.Fprintf(os.Stderr, "%v stack '%v' in region %v of account %v\n\n", method, bold(deployFlags.StackName), awsConfig.Region, account)
	}
	printBasicStackInfo(deployment, true, awsConfig)
}

func setDeployTemplate(deployment *lib.DeployInfo, awsConfig config.AWSConfig) {
	template, path := readTemplateFromSource(deployment)
	deployment.TemplateRelativePath = path
	deployment.Template = template

	if deployFlags.Bucket != "" {
		deployment.TemplateUrl = uploadTemplateToS3(template, awsConfig)
	}

	deployment.TemplateLocalPath = calculateTemplateLocalPath(path)
}

// readTemplateFromSource reads the template from either a deployment file or template flag
func readTemplateFromSource(deployment *lib.DeployInfo) (string, string) {
	var template string
	var path string
	var err error

	if deployment.StackDeploymentFile != nil {
		template, path, err = lib.ReadFile(&deployment.StackDeploymentFile.TemplateFilePath, "templates")
	} else {
		template, path, err = lib.ReadTemplate(&deployFlags.Template)
	}

	if err != nil {
		printMessage(formatError(string(texts.FileTemplateReadFailure)))
		log.Fatalln(err)
	}

	return template, path
}

// uploadTemplateToS3 uploads the template to S3 and returns the URL
func uploadTemplateToS3(template string, awsConfig config.AWSConfig) string {
	ctx := context.Background()
	objectname, err := lib.UploadTemplate(ctx, &deployFlags.Template, template, &deployFlags.Bucket, awsConfig.S3Client())
	if err != nil {
		printMessage(formatError("Failed to upload template to S3"))
		log.Fatalln(err)
	}
	return fmt.Sprintf("https://%v.s3-%v.amazonaws.com/%v", deployFlags.Bucket, awsConfig.Region, objectname)
}

// calculateTemplateLocalPath calculates the relative path from the root directory
func calculateTemplateLocalPath(path string) string {
	var confpath, localpath string
	var err error

	if cfgFile != "" {
		confdir := filepath.Dir(cfgFile)
		confpath, err = filepath.Abs(fmt.Sprintf("%s%s%s", confdir, string(os.PathSeparator), viper.GetString("rootdir")))
		if err != nil {
			log.Printf("Warning: failed to get absolute config path: %v", err)
			return path
		}
	} else {
		confpath, err = filepath.Abs(viper.GetString("rootdir"))
		if err != nil {
			log.Printf("Warning: failed to get absolute root path: %v", err)
			return path
		}
	}

	localpath, err = filepath.Abs(path)
	if err != nil {
		log.Printf("Warning: failed to get absolute path for template: %v", err)
		return path
	}

	relativePath, err := filepath.Rel(confpath, localpath)
	if err != nil {
		log.Printf("Warning: failed to calculate relative path: %v", err)
		return path
	}
	return relativePath
}

func setDeployTags(deployment *lib.DeployInfo) {
	tagresult := make([]types.Tag, 0)

	if deployFlags.DefaultTags {
		tagresult = append(tagresult, loadDefaultTags(deployment)...)
	}

	if deployment.StackDeploymentFile != nil {
		tagresult = append(tagresult, loadDeploymentFileTags(deployment)...)
	} else if deployFlags.Tags != "" {
		tagresult = append(tagresult, loadTagsFromFiles(deployFlags.Tags, deployment)...)
	}

	deployment.Tags = tagresult
}

// loadDefaultTags loads tags from the default configuration
func loadDefaultTags(deployment *lib.DeployInfo) []types.Tag {
	tags := make([]types.Tag, 0)
	for key, value := range viper.GetStringMapString("tags.default") {
		tag := types.Tag{
			Key:   aws.String(key),
			Value: aws.String(placeholderParser(value, deployment)),
		}
		tags = append(tags, tag)
	}
	return tags
}

// loadDeploymentFileTags loads tags from the stack deployment file
func loadDeploymentFileTags(deployment *lib.DeployInfo) []types.Tag {
	tags := make([]types.Tag, 0)
	for key, value := range deployment.StackDeploymentFile.Tags {
		tag := types.Tag{
			Key:   aws.String(key),
			Value: aws.String(placeholderParser(value, deployment)),
		}
		tags = append(tags, tag)
	}
	return tags
}

// loadTagsFromFiles loads and parses tags from comma-separated file list
func loadTagsFromFiles(tagFiles string, deployment *lib.DeployInfo) []types.Tag {
	tags := make([]types.Tag, 0)
	for tagfile := range strings.SplitSeq(tagFiles, ",") {
		tagContent, _, err := lib.ReadTagsfile(tagfile)
		if err != nil {
			message := fmt.Sprintf("%v '%v'", texts.FileTagsReadFailure, tagfile)
			printMessage(formatError(message))
			log.Fatalln(err)
		}
		parsedtags, err := lib.ParseTagString(tagContent)
		if err != nil {
			message := fmt.Sprintf("%v '%v'", texts.FileTagsParseFailure, tagfile)
			printMessage(formatError(message))
			log.Fatalln(err)
		}
		tags = append(tags, parsedtags...)
	}
	return tags
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
		parameterresult = append(parameterresult, loadDeploymentFileParameters(deployment)...)
	} else if deployFlags.Parameters != "" {
		parameterresult = append(parameterresult, loadParametersFromFiles(deployFlags.Parameters)...)
	}

	deployment.Parameters = parameterresult
}

// loadDeploymentFileParameters loads parameters from the stack deployment file
func loadDeploymentFileParameters(deployment *lib.DeployInfo) []types.Parameter {
	parameters := make([]types.Parameter, 0)
	for key, value := range deployment.StackDeploymentFile.Parameters {
		parameter := types.Parameter{
			ParameterKey:   aws.String(key),
			ParameterValue: aws.String(value),
		}
		parameters = append(parameters, parameter)
	}
	return parameters
}

// loadParametersFromFiles loads and parses parameters from comma-separated file list
func loadParametersFromFiles(parameterFiles string) []types.Parameter {
	parameters := make([]types.Parameter, 0)
	for parameterfile := range strings.SplitSeq(parameterFiles, ",") {
		paramContent, _, err := lib.ReadParametersfile(parameterfile)
		if err != nil {
			message := fmt.Sprintf("%v '%v'", texts.FileParametersReadFailure, parameterfile)
			printMessage(formatError(message))
			log.Fatalln(err)
		}
		parsedparameters, err := lib.ParseParameterString(paramContent)
		if err != nil {
			message := fmt.Sprintf("%v '%v'", texts.FileParametersParseFailure, parameterfile)
			printMessage(formatError(message))
			log.Fatalln(err)
		}
		parameters = append(parameters, parsedparameters...)
	}
	return parameters
}

// fetchChangeset retrieves an existing changeset (named by --changeset) and
// attaches it to the deployment so it can be executed without creating a new
// one. Used by the --deploy-changeset flow.
func fetchChangeset(deployment *lib.DeployInfo, awsConfig config.AWSConfig) *lib.ChangesetInfo {
	ctx := context.Background()
	rawchangeset, err := deployment.GetChangeset(ctx, awsConfig.CloudformationClient())
	if err != nil {
		message := fmt.Sprintf(string(texts.DeployChangesetMessageRetrieveFailed), deployment.ChangesetName)
		printMessage(formatError(message))
		log.Fatalln(err)
	}
	if len(rawchangeset) == 0 {
		message := fmt.Sprintf(string(texts.DeployChangesetMessageRetrieveFailed), deployment.ChangesetName)
		printMessage(formatError(message))
		os.Exit(1)
	}
	changeset := deployment.AddChangeset(rawchangeset)
	return &changeset
}

func createChangeset(deployment *lib.DeployInfo, awsConfig config.AWSConfig) *lib.ChangesetInfo {
	ctx := context.Background()
	if deployment.TemplateUrl != "" && !deployFlags.Quiet {
		text := fmt.Sprintf("Using template uploaded as %v", deployment.TemplateUrl)
		printMessage(formatInfo(text))
	}
	_, err := deployment.CreateChangeSet(ctx, awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError(string(texts.DeployChangesetMessageCreationFailed)))
		log.Fatalln(err)
	}
	changeset, err := deployment.WaitUntilChangesetDone(ctx, awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError(string(texts.DeployChangesetMessageCreationFailed)))
		log.Fatalln(err)
	}
	if changeset.Status != string(types.ChangeSetStatusCreateComplete) {
		// When the creation fails because there are no changes, say so and complete successfully
		if changeset.StatusReason == string(texts.DeployReceivedErrorMessagesNoChanges) || changeset.StatusReason == string(texts.DeployReceivedErrorMessagesNoUpdates) {
			// Output no-changes message to stderr (streaming output)
			if !deployFlags.Quiet {
				message := fmt.Sprintf(string(texts.DeployChangesetMessageNoChanges), deployment.StackName)
				printMessage(formatSuccess(message))
			}
			// Output final no-changes result to stdout
			if err := outputNoChangesResult(deployment); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)
			}
			os.Exit(0)
		}
		// Otherwise, show the error and clean up
		printMessage(formatError(string(texts.DeployChangesetMessageCreationFailed)))
		fmt.Fprintln(os.Stderr, changeset.StatusReason)
		fmt.Fprintf(os.Stderr, "\n%v %v\n", texts.DeployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
		handleFailedChangeset(deployment, awsConfig)
		os.Exit(1)
	}
	return changeset
}

// handleFailedChangeset prompts the user (or auto-confirms in non-interactive
// mode) and deletes a failed changeset when confirmed.
func handleFailedChangeset(deployment *lib.DeployInfo, awsConfig config.AWSConfig) {
	var deleteChangesetConfirmation bool
	if deployFlags.NonInteractive {
		deleteChangesetConfirmation = true
	} else {
		deleteChangesetConfirmation = askForConfirmationFunc(string(texts.DeployChangesetMessageDeleteConfirm))
	}
	if deleteChangesetConfirmation {
		deleteChangesetFunc(*deployment, awsConfig)
	}
}

func deleteChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	ctx := context.Background()
	switch {
	case deployFlags.Dryrun:
		printMessage(formatInfo(string(texts.DeployChangesetMessageDryrunDelete)))
	case deployFlags.NonInteractive:
		printMessage(formatInfo(string(texts.DeployChangesetMessageAutoDelete)))
	default:
		printMessage(formatSuccess(string(texts.DeployChangesetMessageWillDelete)))
	}
	deleteAttempt := deployment.Changeset.DeleteChangeset(ctx, awsConfig.CloudformationClient())
	if !deleteAttempt {
		printMessage(formatError(string(texts.DeployChangesetMessageDeleteFailed)))
	}
	// Likely a new deployment. Check if the stack is in status REVIEW_IN_PROGRESS and offer to delete
	if deployment.IsNew {
		stack, err := deployment.GetFreshStack(ctx, awsConfig.CloudformationClient())
		if err != nil {
			log.Fatalln(err)
		}
		if stack.StackStatus == types.StackStatusReviewInProgress {
			deleteStackIfNew(deployment, awsConfig)
		}
	}
}

func deleteStackIfNew(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	ctx := context.Background()
	fmt.Fprintln(os.Stderr, texts.DeployStackMessageNewStackDeleteInfo)
	var deleteStackConfirmation bool
	if deployFlags.Dryrun || deployFlags.NonInteractive {
		deleteStackConfirmation = true
	} else {
		deleteStackConfirmation = askForConfirmation("Do you want me to delete this empty stack for you?")
	}
	if deleteStackConfirmation {
		if !deployment.DeleteStack(ctx, awsConfig.CloudformationClient()) {
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
		fmt.Fprintln(os.Stderr, "No problem. I have left the stack intact, please delete it manually once you're done.")
	}
}

func deployChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) error {
	ctx := context.Background()
	if !deployFlags.Quiet {
		if deployFlags.NonInteractive {
			printMessage(formatInfo(string(texts.DeployChangesetMessageAutoDeploy)))
		} else {
			printMessage(formatSuccess(string(texts.DeployChangesetMessageWillDeploy)))
		}
	}
	err := deployment.Changeset.DeployChangeset(ctx, awsConfig.CloudformationClient())
	if err != nil {
		return fmt.Errorf("could not execute changeset: %w", err)
	}
	latest := deployment.Changeset.CreationTime
	time.Sleep(3 * time.Second)
	if !deployFlags.Quiet {
		fmt.Fprintln(os.Stderr, formatBold("Showing the events for the deployment:"))
	}
	ongoing := true
	for ongoing {
		latest = showEvents(deployment, latest, awsConfig, deployFlags.Quiet)
		time.Sleep(3 * time.Second)
		ongoing = deployment.IsOngoing(ctx, awsConfig.CloudformationClient())
	}
	// One last time after the deployment finished in case of a timing mismatch
	showEvents(deployment, latest, awsConfig, deployFlags.Quiet)
	return nil
}

func showEvents(deployment lib.DeployInfo, latest time.Time, awsConfig config.AWSConfig, quiet bool) time.Time {
	// Return early if quiet mode is enabled
	if quiet {
		return latest
	}

	ctx := context.Background()
	events, err := deployment.GetEvents(ctx, awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError("Something went wrong trying to get the events of the stack"))
		fmt.Fprintln(os.Stderr, err)
		return latest
	}
	sort.Sort(ReverseEvents(events))
	for _, event := range events {
		msg, newLatest, isNew := renderEvent(event, latest)
		if !isNew {
			continue
		}
		latest = newLatest
		switch event.ResourceStatus {
		case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed, types.ResourceStatusImportRollbackComplete, types.ResourceStatus(types.StackStatusRollbackComplete), types.ResourceStatus(types.StackStatusUpdateRollbackComplete):
			// For streaming logs, just apply color without extra spacing
			fmt.Fprintln(os.Stderr, output.StyleWarning(msg))
		case types.ResourceStatusCreateComplete, types.ResourceStatusImportComplete, types.ResourceStatusUpdateComplete, types.ResourceStatusDeleteComplete:
			// For streaming logs, just apply color without extra spacing
			fmt.Fprintln(os.Stderr, output.StylePositive(msg))
		default:
			fmt.Fprintln(os.Stderr, msg)
		}
	}
	return latest
}

func showFailedEvents(deployment lib.DeployInfo, awsConfig config.AWSConfig, prefixMessage string) []map[string]any {
	ctx := context.Background()
	events, err := deployment.GetEvents(ctx, awsConfig.CloudformationClient())
	if err != nil {
		printMessage(formatError("Something went wrong trying to get the events of the stack"))
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	changesetkeys := []string{"CfnName", "Type", "Status", "Reason"}
	changesettitle := fmt.Sprintf("Failed events in deployment of changeset %v", deployment.Changeset.Name)
	sort.Sort(ReverseEvents(events))
	result := make([]map[string]any, 0)
	for _, event := range events {
		row := renderFailedEvent(event, deployment.Changeset.CreationTime)
		if row != nil {
			result = append(result, row)
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
		out := createStderrOutput()
		if err := out.Render(context.Background(), doc); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to render failed events: %v\n", err)
		}
	} else if prefixMessage != "" {
		// If no failed events but we have a prefix message, still show it
		printMessage(prefixMessage)
	}

	return result
}

// ReverseEvents implements sort.Interface for reverse-chronological sorting of stack events
type ReverseEvents []types.StackEvent

func (a ReverseEvents) Len() int { return len(a) }

// Less sorts events chronologically (oldest first). Events with nil
// Timestamps are treated as the zero time so they sort to the beginning.
func (a ReverseEvents) Less(i, j int) bool {
	ti := safeTimestamp(a[i].Timestamp)
	tj := safeTimestamp(a[j].Timestamp)
	return ti.Before(tj)
}
func (a ReverseEvents) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// safeTimestamp returns the dereferenced time or the zero value when t is nil.
func safeTimestamp(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// safeString returns the dereferenced string or the fallback when s is nil.
func safeString(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

// renderEvent formats a single StackEvent for streaming output. It returns
// the event timestamp (or latest unchanged) and a boolean indicating whether
// the event was newer than latest. All pointer fields are handled nil-safely.
func renderEvent(event types.StackEvent, latest time.Time) (msg string, ts time.Time, isNew bool) {
	// Events without a timestamp cannot be ordered — skip them
	if event.Timestamp == nil {
		return "", latest, false
	}

	if !event.Timestamp.After(latest) {
		return "", latest, false
	}

	resourceType := safeString(event.ResourceType, "(unknown type)")
	logicalID := safeString(event.LogicalResourceId, "(unknown id)")

	msg = fmt.Sprintf("%v: %v %v in status %v",
		event.Timestamp.In(settings.GetTimezoneLocation()).Format(time.RFC3339),
		resourceType,
		logicalID,
		event.ResourceStatus,
	)
	return msg, *event.Timestamp, true
}

// renderFailedEvent builds a table row for a failed event. Returns nil if
// the event should be skipped (e.g. timestamp is nil or before cutoff).
// All pointer fields are handled nil-safely.
func renderFailedEvent(event types.StackEvent, cutoff time.Time) map[string]any {
	// Events without a timestamp cannot be compared against the cutoff — skip
	if event.Timestamp == nil {
		return nil
	}
	if !event.Timestamp.After(cutoff) {
		return nil
	}

	switch event.ResourceStatus {
	case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed,
		types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed:
		// continue
	default:
		return nil
	}

	return map[string]any{
		"CfnName": safeString(event.LogicalResourceId, "(unknown id)"),
		"Type":    safeString(event.ResourceType, "(unknown type)"),
		"Status":  string(event.ResourceStatus),
		"Reason":  safeString(event.ResourceStatusReason, "(no reason provided)"),
	}
}

// createStderrOutput creates an output writer for stderr with TTY detection
// This enables conditional formatting based on whether stderr is a TTY.
// When stderr is redirected to a file, ANSI codes are avoided.
func createStderrOutput() *output.Output {
	opts := []output.OutputOption{
		output.WithFormat(settings.GetTableFormat()),
		output.WithWriter(output.NewStderrWriter()),
	}

	// Only add colors and emojis if stderr is a TTY
	// When stderr is redirected to a file, avoid ANSI codes
	if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
		opts = append(opts,
			output.WithTransformers(
				&output.EnhancedEmojiTransformer{},
				&output.EnhancedColorTransformer{},
			),
		)
	}

	return output.NewOutput(opts...)
}
