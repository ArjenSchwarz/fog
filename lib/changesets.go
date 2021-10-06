package lib

import (
	"context"
	"fmt"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type ChangesetInfo struct {
	Changes      []ChangesetChanges
	CreationTime time.Time
	HasModule    bool
	ID           string
	Name         string
	Status       string
	StatusReason string
	StackID      string
	StackName    string
}

type ChangesetChanges struct {
	Action      string
	LogicalID   string
	Replacement string
	ResourceID  string
	Type        string
	Module      string
}

func (changeset *ChangesetInfo) DeleteChangeset(svc *cloudformation.Client) bool {
	input := &cloudformation.DeleteChangeSetInput{
		StackName:     &changeset.StackName,
		ChangeSetName: &changeset.Name,
	}
	_, err := svc.DeleteChangeSet(context.TODO(), input)

	return err == nil
}

func (changeset *ChangesetInfo) DeployChangeset(svc *cloudformation.Client) error {
	input := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: &changeset.Name,
		StackName:     &changeset.StackName,
	}
	_, err := svc.ExecuteChangeSet(context.TODO(), input)
	return err
}

func (changeset *ChangesetInfo) AddChange(changes ChangesetChanges) {
	var contents []ChangesetChanges
	if changeset.Changes != nil {
		contents = changeset.Changes
	}
	contents = append(contents, changes)
	changeset.Changes = contents
	if changes.Module != "" {
		changeset.HasModule = true
	}
}

func (changeset *ChangesetInfo) GetStack(svc *cloudformation.Client) (types.Stack, error) {
	return GetStack(&changeset.StackID, svc)
}

func (changeset *ChangesetInfo) GenerateChangesetUrl(settings config.AWSConfig) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%v#/stacks/changesets/changes?stackId=%v&changeSetId=%v",
		settings.Region, changeset.StackID, changeset.ID)
}
