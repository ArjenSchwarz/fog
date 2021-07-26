package cmd

type deployChangesetMessage string

const (
	deployChangesetMessageCreationFailed deployChangesetMessage = "Something went wrong when trying to create the change set"
	deployChangesetMessageDeleteFailed   deployChangesetMessage = "Something went wrong while trying to delete the change set"
	deployChangesetMessageConsole        deployChangesetMessage = "If you want to look at the change set in the Console, please go to"
	deployChangesetMessageNoChanges      deployChangesetMessage = "No changes to resources have been found, but there are still changes to other parts of the stack"
	deployChangesetMessageWillDelete     deployChangesetMessage = "OK. I will now delete this change set for you. You can still look at it with the above link after it's been deleted."
	deployChangesetMessageDryrunDelete   deployChangesetMessage = "Dry run: Automatically deleting the change set for you. You can still look at it with the above link after it's been deleted."
)

type deployStackMessage string

const (
	deployStackMessageNewStackDryrunDelete  deployStackMessage = "Dry run: Automatically deleting the empty stack for you."
	deployStackMessageNewStackDeleteSuccess deployStackMessage = "OK. I have deleted the stack. You can try to deploy it again."
	deployStackMessageNewStackDeleteInfo    deployStackMessage = "It looks like this was a new stack and doesn't have any resources. You can't deploy a stack with the same name until this one has been deleted."
)
