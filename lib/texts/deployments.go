package texts

type DeployChangesetMessage string

const (
	DeployChangesetMessageAutoDelete     DeployChangesetMessage = "Auto approve: Automatically deleting the change set for you. You can still look at it with the above link after it's been deleted."
	DeployChangesetMessageAutoDeploy     DeployChangesetMessage = "Auto approve: Automatically deploying the change set for you."
	DeployChangesetMessageConsole        DeployChangesetMessage = "If you want to look at the change set in the Console, please go to"
	DeployChangesetMessageCreationFailed DeployChangesetMessage = "Something went wrong when trying to create the change set"
	DeployChangesetMessageDeleteConfirm  DeployChangesetMessage = "Do you want to delete this change set?"
	DeployChangesetMessageDeleteFailed   DeployChangesetMessage = "Something went wrong while trying to delete the change set"
	DeployChangesetMessageDeployConfirm  DeployChangesetMessage = "Do you want to deploy this change set?"
	DeployChangesetMessageDryrunDelete   DeployChangesetMessage = "Dry run: Automatically deleting the change set for you. You can still look at it with the above link after it's been deleted."
	DeployChangesetMessageNoChanges      DeployChangesetMessage = "No changes to resources have been found, but there are still changes to other parts of the stack"
	DeployChangesetMessageChanges        DeployChangesetMessage = "Changes found in change set"
	DeployChangesetMessageWillDelete     DeployChangesetMessage = "OK. I will now delete this change set for you. You can still look at it with the above link after it's been deleted."
)

type DeployStackMessage string

const (
	DeployStackMessageNewStackDryrunDelete  DeployStackMessage = "Dry run: Automatically deleting the empty stack for you."
	DeployStackMessageNewStackAutoDelete    DeployStackMessage = "Auto approve: Automatically deleting the empty stack for you."
	DeployStackMessageNewStackDeleteSuccess DeployStackMessage = "OK. I have deleted the stack. You can try to deploy it again."
	DeployStackMessageNewStackDeleteInfo    DeployStackMessage = "It looks like this was a new stack and doesn't have any resources. You can't deploy a stack with the same name until this one has been deleted."
	DeployStackMessageSuccess               DeployStackMessage = "Deployment completed successfully."
	DeployStackMessageFailed                DeployStackMessage = "The deployment had a problem, please look at the error messages below to figure out why."
	DeployStackMessageRetrievePostFailed    DeployStackMessage = "Something went wrong when I tried to fetch the stack after the deployment."
)

type FileMessage string

const (
	FileTemplateReadFailure   FileMessage = "Something went wrong trying to read the template file"
	FileTagsReadFailure       FileMessage = "Something went wrong trying to read the tags file"
	FileParametersReadFailure FileMessage = "Something went wrong trying to read the parameters file"
)
