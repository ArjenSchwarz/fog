package texts

import (
	"fmt"
	"testing"
)

func TestDeployChangesetMessage(t *testing.T) {
	// Test that all DeployChangesetMessage constants are defined and have the expected values
	tests := []struct {
		name     string
		message  DeployChangesetMessage
		expected string
	}{
		{
			name:     "AutoDelete",
			message:  DeployChangesetMessageAutoDelete,
			expected: "Non-interactive mode: Automatically deleting the change set for you.",
		},
		{
			name:     "AutoDeploy",
			message:  DeployChangesetMessageAutoDeploy,
			expected: "Non-interactive mode: Automatically deploying the change set for you.",
		},
		{
			name:     "Console",
			message:  DeployChangesetMessageConsole,
			expected: "If you want to look at the change set in the Console, please go to",
		},
		{
			name:     "CreationFailed",
			message:  DeployChangesetMessageCreationFailed,
			expected: "Something went wrong when trying to create the change set",
		},
		{
			name:     "RetrieveFailed",
			message:  DeployChangesetMessageRetrieveFailed,
			expected: "Something went wrong when trying to retrieve change set %v",
		},
		{
			name:     "DeleteConfirm",
			message:  DeployChangesetMessageDeleteConfirm,
			expected: "Do you want to delete this change set?",
		},
		{
			name:     "DeleteFailed",
			message:  DeployChangesetMessageDeleteFailed,
			expected: "Something went wrong while trying to delete the change set",
		},
		{
			name:     "DeployConfirm",
			message:  DeployChangesetMessageDeployConfirm,
			expected: "Do you want to deploy this change set?",
		},
		{
			name:     "WillDeploy",
			message:  DeployChangesetMessageWillDeploy,
			expected: "Deploying this Changeset.",
		},
		{
			name:     "DryrunDelete",
			message:  DeployChangesetMessageDryrunDelete,
			expected: "Dry run: Automatically deleting the change set for you.",
		},
		{
			name:     "DryrunSuccess",
			message:  DeployChangesetMessageDryrunSuccess,
			expected: "Dry run: Change set has been successfully created.",
		},
		{
			name:     "Success",
			message:  DeployChangesetMessageSuccess,
			expected: "Change set has been successfully created.",
		},
		{
			name:     "NoChanges",
			message:  DeployChangesetMessageNoChanges,
			expected: "The change set for %v contains no changes",
		},
		{
			name:     "NoResourceChanges",
			message:  DeployChangesetMessageNoResourceChanges,
			expected: "The change set contains no resource changes, but there are still changes to other parts of the stack",
		},
		{
			name:     "Changes",
			message:  DeployChangesetMessageChanges,
			expected: "Changes found in change set",
		},
		{
			name:     "WillDelete",
			message:  DeployChangesetMessageWillDelete,
			expected: "Deleting this change set for you.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.message) != tt.expected {
				t.Errorf("DeployChangesetMessage%s = %q, want %q", tt.name, tt.message, tt.expected)
			}
		})
	}
}

func TestDeployStackMessage(t *testing.T) {
	// Test that all DeployStackMessage constants are defined and have the expected values
	tests := []struct {
		name     string
		message  DeployStackMessage
		expected string
	}{
		{
			name:     "NewStackDryrunDelete",
			message:  DeployStackMessageNewStackDryrunDelete,
			expected: "Dry run: Automatically deleting the empty stack for you.",
		},
		{
			name:     "NewStackAutoDelete",
			message:  DeployStackMessageNewStackAutoDelete,
			expected: "Non-interactive mode: Automatically deleting the empty stack for you.",
		},
		{
			name:     "NewStackDeleteSuccess",
			message:  DeployStackMessageNewStackDeleteSuccess,
			expected: "The stack has been deleted. You can try to deploy it again.",
		},
		{
			name:     "NewStackDeleteInfo",
			message:  DeployStackMessageNewStackDeleteInfo,
			expected: "It looks like this was a new stack and doesn't have any resources. You can't deploy a stack with the same name until this one has been deleted.",
		},
		{
			name:     "Success",
			message:  DeployStackMessageSuccess,
			expected: "Deployment completed successfully.",
		},
		{
			name:     "Failed",
			message:  DeployStackMessageFailed,
			expected: "The deployment had a problem, please look at the error messages below to figure out what happened.",
		},
		{
			name:     "RetrievePostFailed",
			message:  DeployStackMessageRetrievePostFailed,
			expected: "Something went wrong when I tried to fetch the stack after the deployment.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.message) != tt.expected {
				t.Errorf("DeployStackMessage%s = %q, want %q", tt.name, tt.message, tt.expected)
			}
		})
	}
}

func TestFileMessage(t *testing.T) {
	// Test that all FileMessage constants are defined and have the expected values
	tests := []struct {
		name     string
		message  FileMessage
		expected string
	}{
		{
			name:     "TemplateReadFailure",
			message:  FileTemplateReadFailure,
			expected: "Something went wrong trying to read the template file",
		},
		{
			name:     "TagsReadFailure",
			message:  FileTagsReadFailure,
			expected: "Something went wrong trying to read the tags file",
		},
		{
			name:     "TagsParseFailure",
			message:  FileTagsParseFailure,
			expected: "Something went wrong trying to parse the tags file",
		},
		{
			name:     "ParametersReadFailure",
			message:  FileParametersReadFailure,
			expected: "Something went wrong trying to read the parameters file",
		},
		{
			name:     "ParametersParseFailure",
			message:  FileParametersParseFailure,
			expected: "Something went wrong trying to parse the parameters file",
		},
		{
			name:     "PrecheckStarted",
			message:  FilePrecheckStarted,
			expected: "Starting %v prechecks...",
		},
		{
			name:     "PrecheckSuccess",
			message:  FilePrecheckSuccess,
			expected: "All prechecks finished successfully",
		},
		{
			name:     "PrecheckFailureStop",
			message:  FilePrecheckFailureStop,
			expected: "Issues detected during prechecks, stopping deployment. Please read the below output and fix before trying again",
		},
		{
			name:     "PrecheckFailureContinue",
			message:  FilePrecheckFailureContinue,
			expected: "Issues detected during prechecks, continuing regardless",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.message) != tt.expected {
				t.Errorf("FileMessage%s = %q, want %q", tt.name, tt.message, tt.expected)
			}
		})
	}
}

func TestDeployReceivedErrorMessage(t *testing.T) {
	// Test that all DeployReceivedErrorMessage constants are defined and have the expected values
	tests := []struct {
		name     string
		message  DeployReceivedErrorMessage
		expected string
	}{
		{
			name:     "NoChanges",
			message:  DeployReceivedErrorMessagesNoChanges,
			expected: "The submitted information didn't contain changes. Submit different information to create a change set.",
		},
		{
			name:     "NoUpdates",
			message:  DeployReceivedErrorMessagesNoUpdates,
			expected: "No updates are to be performed.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.message) != tt.expected {
				t.Errorf("DeployReceivedErrorMessage%s = %q, want %q", tt.name, tt.message, tt.expected)
			}
		})
	}
}

// Test string formatting with the messages
func TestMessageFormatting(t *testing.T) {
	// Test that messages with format specifiers can be formatted correctly
	stackName := "test-stack"
	formatted := string(DeployChangesetMessageNoChanges)
	expected := "The change set for %v contains no changes"

	if formatted != expected {
		t.Errorf("Before formatting: got %q, want %q", formatted, expected)
	}

	// Format the message
	formatted = string(DeployChangesetMessageNoChanges)
	formatted = fmt.Sprintf(formatted, stackName)
	expected = "The change set for test-stack contains no changes"

	if formatted != expected {
		t.Errorf("After formatting: got %q, want %q", formatted, expected)
	}
}
