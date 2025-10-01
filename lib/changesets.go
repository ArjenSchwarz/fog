package lib

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/aws/aws-sdk-go-v2/aws"
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
	Details     []types.ResourceChangeDetail
}

func (changeset *ChangesetInfo) DeleteChangeset(svc CloudFormationDeleteChangeSetAPI) bool {
	input := &cloudformation.DeleteChangeSetInput{
		StackName:     &changeset.StackName,
		ChangeSetName: &changeset.Name,
	}
	_, err := svc.DeleteChangeSet(context.TODO(), input)

	return err == nil
}

func (changeset *ChangesetInfo) DeployChangeset(svc CloudFormationExecuteChangeSetAPI) error {
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

func (changeset *ChangesetInfo) GetStack(svc CloudFormationDescribeStacksAPI) (types.Stack, error) {
	return GetStack(&changeset.StackID, svc)
}

func (changeset *ChangesetInfo) GenerateChangesetUrl(settings config.AWSConfig) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%v#/stacks/changesets/changes?stackId=%v&changeSetId=%v",
		settings.Region, changeset.StackID, changeset.ID)
}

func GetStackAndChangesetFromURL(changeseturl string, region string) (string, string) {
	decodedValue, err := url.QueryUnescape(changeseturl)
	if err != nil {
		log.Fatal(err)
		return "", ""
	}
	decodedValue = strings.ReplaceAll(decodedValue, "\\", "")
	replacestring := fmt.Sprintf("?region=%s#/stacks/changesets/changes", region)
	decodedValue = strings.Replace(decodedValue, replacestring, "", 1)
	u, err := url.Parse(decodedValue)
	if err != nil {
		panic(err)
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		panic(err)
	}
	stackid := q.Get("stackId")
	changesetid := q.Get("changeSetId")
	return stackid, changesetid
}

func (changes *ChangesetChanges) GetDangerDetails() []string {
	details := []string{}
	for _, detail := range changes.Details {
		if detail.Target.RequiresRecreation != "Never" {
			change := fmt.Sprintf("%v: %v - %v", detail.Evaluation, detail.Target.Attribute, aws.ToString(detail.CausingEntity))
			details = append(details, change)
		}

	}
	return details
}
