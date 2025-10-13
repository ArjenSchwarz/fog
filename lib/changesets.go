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

// ChangesetInfo represents information about a CloudFormation changeset
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

// ChangesetChanges represents a single resource change within a changeset
type ChangesetChanges struct {
	Action      string
	LogicalID   string
	Replacement string
	ResourceID  string
	Type        string
	Module      string
	Details     []types.ResourceChangeDetail
}

// DeleteChangeset deletes the changeset and returns true if successful
func (changeset *ChangesetInfo) DeleteChangeset(svc CloudFormationDeleteChangeSetAPI) bool {
	input := &cloudformation.DeleteChangeSetInput{
		StackName:     &changeset.StackName,
		ChangeSetName: &changeset.Name,
	}
	_, err := svc.DeleteChangeSet(context.TODO(), input)

	return err == nil
}

// DeployChangeset executes the changeset to deploy the changes
func (changeset *ChangesetInfo) DeployChangeset(svc CloudFormationExecuteChangeSetAPI) error {
	input := &cloudformation.ExecuteChangeSetInput{
		ChangeSetName: &changeset.Name,
		StackName:     &changeset.StackName,
	}
	_, err := svc.ExecuteChangeSet(context.TODO(), input)
	return err
}

// AddChange adds a change to the changeset's list of changes
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

// GetStack retrieves the stack associated with this changeset
func (changeset *ChangesetInfo) GetStack(svc CloudFormationDescribeStacksAPI) (types.Stack, error) {
	return GetStack(&changeset.StackID, svc)
}

// GenerateChangesetUrl generates the AWS console URL for viewing the changeset
func (changeset *ChangesetInfo) GenerateChangesetUrl(settings config.AWSConfig) string {
	return fmt.Sprintf("https://console.aws.amazon.com/cloudformation/home?region=%v#/stacks/changesets/changes?stackId=%v&changeSetId=%v",
		settings.Region, changeset.StackID, changeset.ID)
}

// GetStackAndChangesetFromURL extracts the stack ID and changeset ID from an AWS console changeset URL
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

// GetDangerDetails returns details of dangerous changes that require resource recreation
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
