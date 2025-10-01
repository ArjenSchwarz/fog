package lib

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
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
	type fields struct {
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

	tests := []struct {
		name      string
		fields    fields
		mockError error
		want      bool
	}{
		{
			name: "Successful deletion",
			fields: fields{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: nil,
			want:      true,
		},
		{
			name: "Failed deletion",
			fields: fields{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: errors.New("deletion failed"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockChangesetCloudFormationClient{
				deleteChangeSetError: tt.mockError,
			}

			changeset := &ChangesetInfo{
				Changes:      tt.fields.Changes,
				CreationTime: tt.fields.CreationTime,
				HasModule:    tt.fields.HasModule,
				ID:           tt.fields.ID,
				Name:         tt.fields.Name,
				Status:       tt.fields.Status,
				StatusReason: tt.fields.StatusReason,
				StackID:      tt.fields.StackID,
				StackName:    tt.fields.StackName,
			}

			if got := changeset.DeleteChangeset(mockClient); got != tt.want {
				t.Errorf("ChangesetInfo.DeleteChangeset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangesetInfo_DeployChangeset(t *testing.T) {
	type fields struct {
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

	tests := []struct {
		name      string
		fields    fields
		mockError error
		wantErr   bool
	}{
		{
			name: "Successful deployment",
			fields: fields{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: nil,
			wantErr:   false,
		},
		{
			name: "Failed deployment",
			fields: fields{
				Name:      "test-changeset",
				StackName: "test-stack",
			},
			mockError: errors.New("deployment failed"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockChangesetCloudFormationClient{
				executeChangeSetError: tt.mockError,
			}

			changeset := &ChangesetInfo{
				Changes:      tt.fields.Changes,
				CreationTime: tt.fields.CreationTime,
				HasModule:    tt.fields.HasModule,
				ID:           tt.fields.ID,
				Name:         tt.fields.Name,
				Status:       tt.fields.Status,
				StatusReason: tt.fields.StatusReason,
				StackID:      tt.fields.StackID,
				StackName:    tt.fields.StackName,
			}

			if err := changeset.DeployChangeset(mockClient); (err != nil) != tt.wantErr {
				t.Errorf("ChangesetInfo.DeployChangeset() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChangesetInfo_AddChange(t *testing.T) {
	type fields struct {
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

	type args struct {
		changes ChangesetChanges
	}

	tests := []struct {
		name              string
		fields            fields
		args              args
		wantChangesLength int
		wantHasModule     bool
	}{
		{
			name: "Add first change without module",
			fields: fields{
				Changes:   nil,
				HasModule: false,
			},
			args: args{
				changes: ChangesetChanges{
					Action:    "Add",
					LogicalID: "Resource1",
					Type:      "AWS::S3::Bucket",
					Module:    "",
				},
			},
			wantChangesLength: 1,
			wantHasModule:     false,
		},
		{
			name: "Add change with module",
			fields: fields{
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
			args: args{
				changes: ChangesetChanges{
					Action:    "Modify",
					LogicalID: "Resource2",
					Type:      "AWS::IAM::Role",
					Module:    "SecurityModule",
				},
			},
			wantChangesLength: 2,
			wantHasModule:     true,
		},
		{
			name: "Add another change with module already set",
			fields: fields{
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
			args: args{
				changes: ChangesetChanges{
					Action:    "Remove",
					LogicalID: "Resource3",
					Type:      "AWS::Lambda::Function",
					Module:    "",
				},
			},
			wantChangesLength: 3,
			wantHasModule:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changeset := &ChangesetInfo{
				Changes:      tt.fields.Changes,
				CreationTime: tt.fields.CreationTime,
				HasModule:    tt.fields.HasModule,
				ID:           tt.fields.ID,
				Name:         tt.fields.Name,
				Status:       tt.fields.Status,
				StatusReason: tt.fields.StatusReason,
				StackID:      tt.fields.StackID,
				StackName:    tt.fields.StackName,
			}

			changeset.AddChange(tt.args.changes)

			// Check if the change was added correctly
			if len(changeset.Changes) != tt.wantChangesLength {
				t.Errorf("ChangesetInfo.AddChange() resulted in %d changes, want %d", len(changeset.Changes), tt.wantChangesLength)
			}

			// Check if HasModule was set correctly
			if changeset.HasModule != tt.wantHasModule {
				t.Errorf("ChangesetInfo.AddChange() set HasModule to %v, want %v", changeset.HasModule, tt.wantHasModule)
			}

			// Check if the last change added matches what we expect
			if len(changeset.Changes) > 0 {
				lastChange := changeset.Changes[len(changeset.Changes)-1]
				if lastChange.Action != tt.args.changes.Action ||
					lastChange.LogicalID != tt.args.changes.LogicalID ||
					lastChange.Type != tt.args.changes.Type ||
					lastChange.Module != tt.args.changes.Module {
					t.Errorf("ChangesetInfo.AddChange() last change = %+v, want %+v", lastChange, tt.args.changes)
				}
			}
		})
	}
}

func TestChangesetInfo_GetStack(t *testing.T) {
	type fields struct {
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

	// Create test stack
	testStack := types.Stack{
		StackName:   aws.String("test-stack"),
		StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
		StackStatus: types.StackStatusCreateComplete,
	}

	tests := []struct {
		name       string
		fields     fields
		mockOutput cloudformation.DescribeStacksOutput
		mockError  error
		want       types.Stack
		wantErr    bool
	}{
		{
			name: "Successful get stack",
			fields: fields{
				StackID: "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			},
			mockOutput: cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{testStack},
			},
			mockError: nil,
			want:      testStack,
			wantErr:   false,
		},
		{
			name: "Failed get stack",
			fields: fields{
				StackID: "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			},
			mockOutput: cloudformation.DescribeStacksOutput{},
			mockError:  errors.New("stack not found"),
			want:       types.Stack{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &MockChangesetCloudFormationClient{
				describeStacksOutput: tt.mockOutput,
				describeStacksError:  tt.mockError,
			}

			changeset := &ChangesetInfo{
				Changes:      tt.fields.Changes,
				CreationTime: tt.fields.CreationTime,
				HasModule:    tt.fields.HasModule,
				ID:           tt.fields.ID,
				Name:         tt.fields.Name,
				Status:       tt.fields.Status,
				StatusReason: tt.fields.StatusReason,
				StackID:      tt.fields.StackID,
				StackName:    tt.fields.StackName,
			}

			got, err := changeset.GetStack(mockClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("ChangesetInfo.GetStack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChangesetInfo.GetStack() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangesetInfo_GenerateChangesetUrl(t *testing.T) {
	type fields struct {
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
	type args struct {
		settings config.AWSConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Generate URL for ap-southeast-2",
			fields: fields{
				ID:      "arn:aws:cloudformation:ap-southeast-2:123456789012:changeSet/test-changeset/abc123",
				StackID: "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/test-stack/def456",
			},
			args: args{
				settings: config.AWSConfig{
					Region: "ap-southeast-2",
				},
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/changesets/changes?stackId=arn:aws:cloudformation:ap-southeast-2:123456789012:stack/test-stack/def456&changeSetId=arn:aws:cloudformation:ap-southeast-2:123456789012:changeSet/test-changeset/abc123",
		},
		{
			name: "Generate URL for us-east-1",
			fields: fields{
				ID:      "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/abc123",
				StackID: "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/def456",
			},
			args: args{
				settings: config.AWSConfig{
					Region: "us-east-1",
				},
			},
			want: "https://console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/changesets/changes?stackId=arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/def456&changeSetId=arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/abc123",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changeset := &ChangesetInfo{
				Changes:      tt.fields.Changes,
				CreationTime: tt.fields.CreationTime,
				HasModule:    tt.fields.HasModule,
				ID:           tt.fields.ID,
				Name:         tt.fields.Name,
				Status:       tt.fields.Status,
				StatusReason: tt.fields.StatusReason,
				StackID:      tt.fields.StackID,
				StackName:    tt.fields.StackName,
			}
			if got := changeset.GenerateChangesetUrl(tt.args.settings); got != tt.want {
				t.Errorf("ChangesetInfo.GenerateChangesetUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChangesetChanges_GetDangerDetails(t *testing.T) {
	type fields struct {
		Action      string
		LogicalID   string
		Replacement string
		ResourceID  string
		Type        string
		Module      string
		Details     []types.ResourceChangeDetail
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "No danger details",
			fields: fields{
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
		{
			name: "With danger details - Always",
			fields: fields{
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
		{
			name: "With danger details - Conditional",
			fields: fields{
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
		{
			name: "Multiple danger details",
			fields: fields{
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := &ChangesetChanges{
				Action:      tt.fields.Action,
				LogicalID:   tt.fields.LogicalID,
				Replacement: tt.fields.Replacement,
				ResourceID:  tt.fields.ResourceID,
				Type:        tt.fields.Type,
				Module:      tt.fields.Module,
				Details:     tt.fields.Details,
			}
			got := changes.GetDangerDetails()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ChangesetChanges.GetDangerDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetStackAndChangesetFromURL tests parsing of stack and changeset IDs from console URLs
func TestGetStackAndChangesetFromURL(t *testing.T) {
	tests := []struct {
		name          string
		changeseturl  string
		region        string
		wantStack     string
		wantChangeset string
	}{
		{
			name:          "Valid URL with escaped characters",
			changeseturl:  "https://console.aws.amazon.com/cloudformation/home?region=us-east-1#/stacks/changesets/changes?stackId=arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123&changeSetId=arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/xyz789",
			region:        "us-east-1",
			wantStack:     "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			wantChangeset: "arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/xyz789",
		},
		{
			name:          "Valid URL with URL encoding",
			changeseturl:  "https://console.aws.amazon.com/cloudformation/home?region=ap-southeast-2#/stacks/changesets/changes?stackId=arn%3Aaws%3Acloudformation%3Aap-southeast-2%3A123456789012%3Astack%2Fmy-stack%2F12345&changeSetId=arn%3Aaws%3Acloudformation%3Aap-southeast-2%3A123456789012%3AchangeSet%2Fmy-cs%2F67890",
			region:        "ap-southeast-2",
			wantStack:     "arn:aws:cloudformation:ap-southeast-2:123456789012:stack/my-stack/12345",
			wantChangeset: "arn:aws:cloudformation:ap-southeast-2:123456789012:changeSet/my-cs/67890",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStack, gotChangeset := GetStackAndChangesetFromURL(tt.changeseturl, tt.region)
			if gotStack != tt.wantStack {
				t.Errorf("GetStackAndChangesetFromURL() gotStack = %v, want %v", gotStack, tt.wantStack)
			}
			if gotChangeset != tt.wantChangeset {
				t.Errorf("GetStackAndChangesetFromURL() gotChangeset = %v, want %v", gotChangeset, tt.wantChangeset)
			}
		})
	}
}
