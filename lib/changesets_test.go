package lib

import (
	"context"
	"errors"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock implementation of the CloudFormation client for changesets testing
type MockChangesetCloudFormationClient struct {
	deleteChangeSetError  error
	executeChangeSetError error
	describeStacksOutput  cloudformation.DescribeStacksOutput
	describeStacksError   error
}

// Implement the CloudFormationDeleteChangeSetAPI interface
func (m *MockChangesetCloudFormationClient) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	return &cloudformation.DeleteChangeSetOutput{}, m.deleteChangeSetError
}

// Implement the CloudFormationExecuteChangeSetAPI interface
func (m *MockChangesetCloudFormationClient) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	return &cloudformation.ExecuteChangeSetOutput{}, m.executeChangeSetError
}

// Implement the CloudFormationDescribeStacksAPI interface
func (m *MockChangesetCloudFormationClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return &m.describeStacksOutput, m.describeStacksError
}

func TestChangesetInfo_DeleteChangeset(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		mockError error
		want      bool
	}{
		"successful deletion": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: nil,
			want:      true,
		},
		"failed deletion": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: errors.New("deletion failed"),
			want:      false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create mock client
			mockClient := &MockChangesetCloudFormationClient{
				deleteChangeSetError: tc.mockError,
			}

			got := tc.changeset.DeleteChangeset(mockClient)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestChangesetInfo_DeployChangeset(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		mockError error
		wantErr   bool
	}{
		"successful deployment": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: nil,
			wantErr:   false,
		},
		"failed deployment": {
			changeset: &ChangesetInfo{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: errors.New("deployment failed"),
			wantErr:   true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create mock client
			mockClient := &MockChangesetCloudFormationClient{
				executeChangeSetError: tc.mockError,
			}

			err := tc.changeset.DeployChangeset(mockClient)

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestChangesetInfo_AddChange(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset         *ChangesetInfo
		changeToAdd       ChangesetChanges
		wantChangesLength int
		wantHasModule     bool
	}{
		"add first change without module": {
			changeset: &ChangesetInfo{
				Changes:   nil,
				HasModule: false,
			},
			changeToAdd: ChangesetChanges{
				Action:    "Add",
				LogicalID: "Resource1",
				Type:      "AWS::S3::Bucket",
				Module:    "",
			},
			wantChangesLength: 1,
			wantHasModule:     false,
		},
		"add change with module": {
			changeset: &ChangesetInfo{
				Changes: []ChangesetChanges{
					{
						Action:    "Add",
						LogicalID: "Resource1",
						Type:      "AWS::S3::Bucket",
						Module:    "",
					},
				},
				HasModule: false,
			},
			changeToAdd: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "Resource2",
				Type:      "AWS::IAM::Role",
				Module:    "SecurityModule",
			},
			wantChangesLength: 2,
			wantHasModule:     true,
		},
		"add another change with module already set": {
			changeset: &ChangesetInfo{
				Changes: []ChangesetChanges{
					{
						Action:    "Add",
						LogicalID: "Resource1",
						Type:      "AWS::S3::Bucket",
						Module:    "",
					},
					{
						Action:    "Modify",
						LogicalID: "Resource2",
						Type:      "AWS::IAM::Role",
						Module:    "SecurityModule",
					},
				},
				HasModule: true,
			},
			changeToAdd: ChangesetChanges{
				Action:    "Remove",
				LogicalID: "Resource3",
				Type:      "AWS::Lambda::Function",
				Module:    "",
			},
			wantChangesLength: 3,
			wantHasModule:     true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.changeset.AddChange(tc.changeToAdd)

			// Check if the change was added correctly
			assert.Len(t, tc.changeset.Changes, tc.wantChangesLength)

			// Check if HasModule was set correctly
			assert.Equal(t, tc.wantHasModule, tc.changeset.HasModule)

			// Check if the last change added matches what we expect
			if len(tc.changeset.Changes) > 0 {
				lastChange := tc.changeset.Changes[len(tc.changeset.Changes)-1]
				assert.Equal(t, tc.changeToAdd.Action, lastChange.Action)
				assert.Equal(t, tc.changeToAdd.LogicalID, lastChange.LogicalID)
				assert.Equal(t, tc.changeToAdd.Type, lastChange.Type)
				assert.Equal(t, tc.changeToAdd.Module, lastChange.Module)
			}
		})
	}
}

func TestChangesetInfo_GetStack(t *testing.T) {
	t.Helper()

	// Create test stack
	testStack := types.Stack{
		StackName:   aws.String("test-stack"),
		StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
		StackStatus: types.StackStatusCreateComplete,
	}

	tests := map[string]struct {
		changeset  *ChangesetInfo
		mockOutput cloudformation.DescribeStacksOutput
		mockError  error
		want       types.Stack
		wantErr    bool
	}{
		"successful get stack": {
			changeset: &ChangesetInfo{
				StackID: "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			},
			mockOutput: cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{testStack},
			},
			mockError: nil,
			want:      testStack,
			wantErr:   false,
		},
		"failed get stack": {
			changeset: &ChangesetInfo{
				StackID: "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			},
			mockOutput: cloudformation.DescribeStacksOutput{},
			mockError:  errors.New("stack not found"),
			want:       types.Stack{},
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create mock client
			mockClient := &MockChangesetCloudFormationClient{
				describeStacksOutput: tc.mockOutput,
				describeStacksError:  tc.mockError,
			}

			got, err := tc.changeset.GetStack(mockClient)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			opts := []cmp.Option{
				cmpopts.IgnoreUnexported(types.Stack{}),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("GetStack() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestChangesetInfo_GenerateChangesetUrl(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeset *ChangesetInfo
		settings  config.AWSConfig
		want      string
	}{
		"generate URL for ap-southeast-2": {
			changeset: &ChangesetInfo{
				ID:      "arn:aws:cloudformation:ap-southeast-2:123456789012:changeSet/test-changeset/abc123",
				StackID: "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/test-stack/def456",
			},
			settings: config.AWSConfig{
				Region: "ap-southeast-2",
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/changesets/changes?stackId=arn:aws:cloudformation:ap-southeast-2:123456789012:stack/test-stack/def456&changeSetId=arn:aws:cloudformation:ap-southeast-2:123456789012:changeSet/test-changeset/abc123",
		},
		"generate URL for us-east-1": {
			changeset: &ChangesetInfo{
				ID:      "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/abc123",
				StackID: "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/def456",
			},
			settings: config.AWSConfig{
				Region: "us-east-1",
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/changesets/changes?stackId=arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/def456&changeSetId=arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/abc123",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.changeset.GenerateChangesetUrl(tc.settings)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestChangesetChanges_GetDangerDetails(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changes ChangesetChanges
		want    []string
	}{
		"no danger details": {
			changes: ChangesetChanges{
				Action:    "Add",
				LogicalID: "Resource1",
				Type:      "AWS::S3::Bucket",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties",
							RequiresRecreation: "Never",
						},
					},
				},
			},
			want: []string{},
		},
		"with danger details - Always": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "Resource1",
				Type:      "AWS::S3::Bucket",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.BucketName",
							RequiresRecreation: "Always",
						},
						CausingEntity: aws.String("BucketName"),
					},
				},
			},
			want: []string{"Static: Properties.BucketName - BucketName"},
		},
		"with danger details - Conditional": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "Resource1",
				Type:      "AWS::S3::Bucket",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeDynamic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.Tags",
							RequiresRecreation: "Conditional",
						},
						CausingEntity: aws.String("Tags"),
					},
				},
			},
			want: []string{"Dynamic: Properties.Tags - Tags"},
		},
		"multiple danger details": {
			changes: ChangesetChanges{
				Action:    "Modify",
				LogicalID: "Resource1",
				Type:      "AWS::S3::Bucket",
				Details: []types.ResourceChangeDetail{
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.BucketName",
							RequiresRecreation: "Always",
						},
						CausingEntity: aws.String("BucketName"),
					},
					{
						Evaluation: types.EvaluationTypeStatic,
						Target: &types.ResourceTargetDefinition{
							Attribute:          "Properties.AccessControl",
							RequiresRecreation: "Never",
						},
						CausingEntity: aws.String("AccessControl"),
					},
				},
			},
			want: []string{"Static: Properties.BucketName - BucketName"},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.changes.GetDangerDetails()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetDangerDetails() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestGetStackAndChangesetFromURL tests parsing of stack and changeset IDs from console URLs
func TestGetStackAndChangesetFromURL(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		changeseturl  string
		region        string
		wantStack     string
		wantChangeset string
	}{
		"valid URL with escaped characters": {
			changeseturl:  "https://console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/changesets/changes?stackId=arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123&changeSetId=arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/xyz789",
			region:        "us-east-1",
			wantStack:     "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			wantChangeset: "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/xyz789",
		},
		"valid URL with URL encoding": {
			changeseturl:  "https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/changesets/changes?stackId=arn%3Aaws%3Acloudformation%3Aap-southeast-2%3A123456789012%3Astack%2Fmy-stack%2F12345&changeSetId=arn%3Aaws%3Acloudformation%3Aap-southeast-2%3A123456789012%3AchangeSet%2Fmy-cs%2F67890",
			region:        "ap-southeast-2",
			wantStack:     "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/my-stack/12345",
			wantChangeset: "arn:aws:cloudformation:ap-southeast-2:123456789012:changeSet/my-cs/67890",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotStack, gotChangeset := GetStackAndChangesetFromURL(tc.changeseturl, tc.region)

			assert.Equal(t, tc.wantStack, gotStack)
			assert.Equal(t, tc.wantChangeset, gotChangeset)
		})
	}
}
