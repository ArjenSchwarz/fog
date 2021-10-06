package texts

type DeployChangesetMessage string

const (
	DeployChangesetMessageAutoDelete        DeployChangesetMessage = "Non-interactive mode: Automatically deleting the change set for you. You can still look at it with the above link after it's been deleted."
	DeployChangesetMessageAutoDeploy        DeployChangesetMessage = "Non-interactive mode: Automatically deploying the change set for you."
	DeployChangesetMessageConsole           DeployChangesetMessage = "If you want to look at the change set in the Console, please go to"
	DeployChangesetMessageCreationFailed    DeployChangesetMessage = "Something went wrong when trying to create the change set"
	DeployChangesetMessageRetrieveFailed    DeployChangesetMessage = "Something went wrong when trying to retrieve change set %v"
	DeployChangesetMessageDeleteConfirm     DeployChangesetMessage = "Do you want to delete this change set?"
	DeployChangesetMessageDeleteFailed      DeployChangesetMessage = "Something went wrong while trying to delete the change set"
	DeployChangesetMessageDeployConfirm     DeployChangesetMessage = "Do you want to deploy this change set?"
	DeployChangesetMessageWillDeploy        DeployChangesetMessage = "OK. Deploying this Changeset."
	DeployChangesetMessageDryrunDelete      DeployChangesetMessage = "Dry run: Automatically deleting the change set for you. You can still look at it with the above link after it's been deleted."
	DeployChangesetMessageDryrunSuccess     DeployChangesetMessage = "Dry run: Change set has been successfully created."
	DeployChangesetMessageSuccess           DeployChangesetMessage = "Change set has been successfully created."
	DeployChangesetMessageNoChanges         DeployChangesetMessage = "No changes have been found in the change set for %v"
	DeployChangesetMessageNoResourceChanges DeployChangesetMessage = "No changes to resources have been found, but there are still changes to other parts of the stack"
	DeployChangesetMessageChanges           DeployChangesetMessage = "Changes found in change set"
	DeployChangesetMessageWillDelete        DeployChangesetMessage = "OK. I will now delete this change set for you. You can still look at it with the above link after it's been deleted."
)

type DeployStackMessage string

const (
	DeployStackMessageNewStackDryrunDelete  DeployStackMessage = "Dry run: Automatically deleting the empty stack for you."
	DeployStackMessageNewStackAutoDelete    DeployStackMessage = "Non-interactive mode: Automatically deleting the empty stack for you."
	DeployStackMessageNewStackDeleteSuccess DeployStackMessage = "OK. I have deleted the stack. You can try to deploy it again."
	DeployStackMessageNewStackDeleteInfo    DeployStackMessage = "It looks like this was a new stack and doesn't have any resources. You can't deploy a stack with the same name until this one has been deleted."
	DeployStackMessageSuccess               DeployStackMessage = "Deployment completed successfully."
	DeployStackMessageFailed                DeployStackMessage = "The deployment had a problem, please look at the error messages below to figure out what happened."
	DeployStackMessageRetrievePostFailed    DeployStackMessage = "Something went wrong when I tried to fetch the stack after the deployment."
)

type FileMessage string

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

type DeployReceivedErrorMessage string

const (
	DeployReceivedErrorMessagesNoChanges DeployReceivedErrorMessage = "The submitted information didn't contain changes. Submit different information to create a change set."
	DeployReceivedErrorMessagesNoUpdates DeployReceivedErrorMessage = "No updates are to be performed."
)
