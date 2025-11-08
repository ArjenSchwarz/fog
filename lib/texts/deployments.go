// Package texts provides standardized user-facing message constants for the fog CLI.
//
// This package centralizes all user-facing text messages used throughout the fog
// application, ensuring consistent messaging and making it easier to maintain and
// update text content. Messages are organized by category and use typed constants
// for type safety.
//
// Message Categories
//
// The package defines several message type categories:
//
//   - DeployChangesetMessage: Messages related to changeset operations (create, deploy, delete)
//   - DeployStackMessage: Messages for stack deployment operations
//   - FileMessage: Messages for file operations and template processing
//   - DeployReceivedErrorMessage: Error messages received during deployment
//
// Each category is represented by a custom type, and individual messages are defined
// as constants of that type.
//
// Usage
//
// Messages are designed to be used directly in code where user-facing text is needed:
//
//	fmt.Println(texts.DeployChangesetMessageSuccess)
//	fmt.Printf(string(texts.DeployChangesetMessageRetrieveFailed), changesetID)
//
// Some messages contain format specifiers (e.g., %v) and should be used with
// string formatting functions like fmt.Printf or fmt.Sprintf.
//
// Design Rationale
//
// Centralizing messages in this package provides several benefits:
//   - Consistency: All similar messages use the same wording
//   - Maintainability: Text changes require updates in only one location
//   - Type Safety: Custom types prevent mixing message categories
//   - Discoverability: All messages for a category are grouped together
//   - i18n Ready: Centralized messages make future internationalization easier
//
// Examples
//
// Using a simple message:
//
//	fmt.Println(texts.DeployStackMessageSuccess)
//	// Output: Deployment completed successfully.
//
// Using a formatted message:
//
//	changesetID := "my-changeset"
//	msg := fmt.Sprintf(string(texts.DeployChangesetMessageRetrieveFailed), changesetID)
//	// Output: Something went wrong when trying to retrieve change set my-changeset
//
// Checking message type:
//
//	var msg texts.DeployChangesetMessage = texts.DeployChangesetMessageSuccess
package texts

// DeployChangesetMessage represents deployment changeset message types
type DeployChangesetMessage string

// Deployment changeset message constants
const (
	DeployChangesetMessageAutoDelete        DeployChangesetMessage = "Non-interactive mode: Automatically deleting the change set for you."
	DeployChangesetMessageAutoDeploy        DeployChangesetMessage = "Non-interactive mode: Automatically deploying the change set for you."
	DeployChangesetMessageConsole           DeployChangesetMessage = "If you want to look at the change set in the Console, please go to"
	DeployChangesetMessageCreationFailed    DeployChangesetMessage = "Something went wrong when trying to create the change set"
	DeployChangesetMessageRetrieveFailed    DeployChangesetMessage = "Something went wrong when trying to retrieve change set %v"
	DeployChangesetMessageDeleteConfirm     DeployChangesetMessage = "Do you want to delete this change set?"
	DeployChangesetMessageDeleteFailed      DeployChangesetMessage = "Something went wrong while trying to delete the change set"
	DeployChangesetMessageDeployConfirm     DeployChangesetMessage = "Do you want to deploy this change set?"
	DeployChangesetMessageWillDeploy        DeployChangesetMessage = "OK. Deploying this Changeset."
	DeployChangesetMessageDryrunDelete      DeployChangesetMessage = "Dry run: Automatically deleting the change set for you."
	DeployChangesetMessageDryrunSuccess     DeployChangesetMessage = "Dry run: Change set has been successfully created."
	DeployChangesetMessageSuccess           DeployChangesetMessage = "Change set has been successfully created."
	DeployChangesetMessageNoChanges         DeployChangesetMessage = "No changes have been found in the change set for %v"
	DeployChangesetMessageNoResourceChanges DeployChangesetMessage = "No changes to resources have been found, but there are still changes to other parts of the stack"
	DeployChangesetMessageChanges           DeployChangesetMessage = "Changes found in change set"
	DeployChangesetMessageWillDelete        DeployChangesetMessage = "OK. I will now delete this change set for you."
)

// DeployStackMessage represents deployment stack message types
type DeployStackMessage string

// Deployment stack message constants
const (
	DeployStackMessageNewStackDryrunDelete  DeployStackMessage = "Dry run: Automatically deleting the empty stack for you."
	DeployStackMessageNewStackAutoDelete    DeployStackMessage = "Non-interactive mode: Automatically deleting the empty stack for you."
	DeployStackMessageNewStackDeleteSuccess DeployStackMessage = "OK. I have deleted the stack. You can try to deploy it again."
	DeployStackMessageNewStackDeleteInfo    DeployStackMessage = "It looks like this was a new stack and doesn't have any resources. You can't deploy a stack with the same name until this one has been deleted."
	DeployStackMessageSuccess               DeployStackMessage = "Deployment completed successfully."
	DeployStackMessageFailed                DeployStackMessage = "The deployment had a problem, please look at the error messages below to figure out what happened."
	DeployStackMessageRetrievePostFailed    DeployStackMessage = "Something went wrong when I tried to fetch the stack after the deployment."
)

// FileMessage represents file operation message types
type FileMessage string

// File operation message constants
const (
	FileTemplateReadFailure     FileMessage = "Something went wrong trying to read the template file"
	FileTagsReadFailure         FileMessage = "Something went wrong trying to read the tags file"
	FileTagsParseFailure        FileMessage = "Something went wrong trying to parse the tags file"
	FileParametersReadFailure   FileMessage = "Something went wrong trying to read the parameters file"
	FileParametersParseFailure  FileMessage = "Something went wrong trying to parse the parameters file"
	FilePrecheckStarted         FileMessage = "Starting %v prechecks..."
	FilePrecheckSuccess         FileMessage = "All prechecks finished successfully"
	FilePrecheckFailureStop     FileMessage = "Issues detected during prechecks, stopping deployment. Please read the below output and fix before trying again"
	FilePrecheckFailureContinue FileMessage = "Issues detected during prechecks, continuing regardless"
)

// DeployReceivedErrorMessage represents deployment error message types
type DeployReceivedErrorMessage string

// Deployment error message constants
const (
	DeployReceivedErrorMessagesNoChanges DeployReceivedErrorMessage = "The submitted information didn't contain changes. Submit different information to create a change set."
	DeployReceivedErrorMessagesNoUpdates DeployReceivedErrorMessage = "No updates are to be performed."
)
