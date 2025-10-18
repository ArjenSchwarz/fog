package lib

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/lib/testutil"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetStack_WithDependencyInjection tests GetStack function with dependency injection
func TestGetStack_WithDependencyInjection(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name      string
		stackName string
		setup     func(*testutil.MockCFNClient)
		want      types.Stack
		wantErr   bool
		errMsg    string
	}{
		"existing stack": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					WithDescription("Test stack description").
					Build()
				client.WithStack(stack)
			},
			want: types.Stack{
				StackName:    strPtr("test-stack"),
				StackId:      strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/12345678-1234-1234-1234-123456789012"),
				StackStatus:  types.StackStatusCreateComplete,
				Description:  strPtr("Test stack description"),
				CreationTime: ptrTime(time.Now()),
			},
			wantErr: false,
		},
		"stack with parameters and outputs": {
			stackName: "param-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("param-stack").
					WithStatus(types.StackStatusUpdateComplete).
					WithParameter("Environment", "Production").
					WithParameter("Version", "1.0.0").
					WithOutput("BucketName", "my-bucket").
					Build()
				client.WithStack(stack)
			},
			want: types.Stack{
				StackName:   strPtr("param-stack"),
				StackId:     strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/param-stack/12345678-1234-1234-1234-123456789012"),
				StackStatus: types.StackStatusUpdateComplete,
				Parameters: []types.Parameter{
					{ParameterKey: strPtr("Environment"), ParameterValue: strPtr("Production")},
					{ParameterKey: strPtr("Version"), ParameterValue: strPtr("1.0.0")},
				},
				Outputs: []types.Output{
					{OutputKey: strPtr("BucketName"), OutputValue: strPtr("my-bucket")},
				},
				CreationTime: ptrTime(time.Now()),
			},
			wantErr: false,
		},
		"stack not found": {
			stackName: "non-existent",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack with name non-existent does not exist"))
			},
			wantErr: true,
			errMsg:  "Stack with name non-existent does not exist",
		},
		"empty stack name - returns all stacks": {
			stackName: "",
			setup: func(client *testutil.MockCFNClient) {
				stack1 := testutil.NewStackBuilder("stack1").Build()
				stack2 := testutil.NewStackBuilder("stack2").Build()
				client.WithStack(stack1).WithStack(stack2)
			},
			wantErr: false,
		},
		"API throttling error": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Rate exceeded"))
			},
			wantErr: true,
			errMsg:  "Rate exceeded",
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			got, err := GetStack(&tc.stackName, mockClient)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)

			// Use cmp.Diff for better comparison with options to ignore time differences and unexported fields
			opts := []cmp.Option{
				cmpopts.IgnoreFields(types.Stack{}, "CreationTime"),
				cmpopts.IgnoreUnexported(types.Stack{}),
				cmpopts.IgnoreUnexported(types.Parameter{}),
				cmpopts.IgnoreUnexported(types.Output{}),
				cmpopts.IgnoreUnexported(types.Tag{}),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" && tc.stackName != "" {
				t.Errorf("GetStack() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestStackExists_WithDependencyInjection tests StackExists function
func TestStackExists_WithDependencyInjection(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		deploymentStackName string
		setup               func(*testutil.MockCFNClient)
		want                bool
	}{
		"stack exists": {
			deploymentStackName: "existing-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("existing-stack").Build()
				client.WithStack(stack)
			},
			want: true,
		},
		"stack does not exist": {
			deploymentStackName: "non-existent",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack does not exist"))
			},
			want: false,
		},
		"stack in review state": {
			deploymentStackName: "review-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("review-stack").
					WithStatus(types.StackStatusReviewInProgress).
					Build()
				client.WithStack(stack)
			},
			want: true,
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			deployment := &DeployInfo{
				StackName: tc.deploymentStackName,
			}

			// Execute
			got := StackExists(deployment, mockClient)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeployInfo_IsReadyForUpdate tests the IsReadyForUpdate method
func TestDeployInfo_IsReadyForUpdate(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName  string
		setup      func(*testutil.MockCFNClient)
		wantReady  bool
		wantStatus string
	}{
		"CREATE_COMPLETE - ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				client.WithStack(stack)
			},
			wantReady:  true,
			wantStatus: string(types.StackStatusCreateComplete),
		},
		"UPDATE_COMPLETE - ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateComplete).
					Build()
				client.WithStack(stack)
			},
			wantReady:  true,
			wantStatus: string(types.StackStatusUpdateComplete),
		},
		"IMPORT_COMPLETE - ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusImportComplete).
					Build()
				client.WithStack(stack)
			},
			wantReady:  true,
			wantStatus: string(types.StackStatusImportComplete),
		},
		"ROLLBACK_COMPLETE - ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusRollbackComplete).
					Build()
				client.WithStack(stack)
			},
			wantReady:  true,
			wantStatus: string(types.StackStatusRollbackComplete),
		},
		"UPDATE_ROLLBACK_COMPLETE - ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateRollbackComplete).
					Build()
				client.WithStack(stack)
			},
			wantReady:  true,
			wantStatus: string(types.StackStatusUpdateRollbackComplete),
		},
		"UPDATE_IN_PROGRESS - not ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateInProgress).
					Build()
				client.WithStack(stack)
			},
			wantReady:  false,
			wantStatus: string(types.StackStatusUpdateInProgress),
		},
		"CREATE_IN_PROGRESS - not ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateInProgress).
					Build()
				client.WithStack(stack)
			},
			wantReady:  false,
			wantStatus: string(types.StackStatusCreateInProgress),
		},
		"DELETE_IN_PROGRESS - not ready": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusDeleteInProgress).
					Build()
				client.WithStack(stack)
			},
			wantReady:  false,
			wantStatus: string(types.StackStatusDeleteInProgress),
		},
		"stack does not exist": {
			stackName: "non-existent",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack does not exist"))
			},
			wantReady:  false,
			wantStatus: "",
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			deployment := DeployInfo{
				StackName: tc.stackName,
			}

			// Execute
			gotReady, gotStatus := deployment.IsReadyForUpdate(mockClient)

			// Assert
			assert.Equal(t, tc.wantReady, gotReady)
			assert.Equal(t, tc.wantStatus, gotStatus)
		})
	}
}

// TestDeployInfo_IsOngoing tests the IsOngoing method
func TestDeployInfo_IsOngoing(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName string
		stackArn  string
		setup     func(*testutil.MockCFNClient)
		want      bool
	}{
		"CREATE_IN_PROGRESS - ongoing": {
			stackName: "test-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateInProgress).
					Build()
				// IsOngoing calls GetFreshStack which uses StackArn
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123"] = stack
			},
			want: true,
		},
		"UPDATE_IN_PROGRESS - ongoing": {
			stackName: "test-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusUpdateInProgress).
					Build()
				// IsOngoing calls GetFreshStack which uses StackArn
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123"] = stack
			},
			want: true,
		},
		"DELETE_IN_PROGRESS - ongoing": {
			stackName: "test-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusDeleteInProgress).
					Build()
				// IsOngoing calls GetFreshStack which uses StackArn
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123"] = stack
			},
			want: true,
		},
		"CREATE_COMPLETE - not ongoing": {
			stackName: "test-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("test-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				// IsOngoing calls GetFreshStack which uses StackArn
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/abc123"] = stack
			},
			want: false,
		},
		"stack does not exist": {
			stackName: "non-existent",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/non-existent/abc123",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack does not exist"))
			},
			want: false,
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			deployment := DeployInfo{
				StackName: tc.stackName,
				StackArn:  tc.stackArn,
			}

			// Execute
			got := deployment.IsOngoing(mockClient)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeployInfo_IsNewStack tests the IsNewStack method
func TestDeployInfo_IsNewStack(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName string
		stackArn  string
		setup     func(*testutil.MockCFNClient)
		want      bool
	}{
		"stack does not exist - new": {
			stackName: "new-stack",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack does not exist"))
			},
			want: true,
		},
		"REVIEW_IN_PROGRESS - new": {
			stackName: "review-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/review-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("review-stack").
					WithStatus(types.StackStatusReviewInProgress).
					Build()
				// IsNewStack calls both StackExists (with StackName) and GetFreshStack (with StackArn)
				client.Stacks["review-stack"] = stack
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/review-stack/abc123"] = stack
			},
			want: true,
		},
		"CREATE_COMPLETE - not new": {
			stackName: "existing-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/existing-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("existing-stack").
					WithStatus(types.StackStatusCreateComplete).
					Build()
				// IsNewStack calls both StackExists (with StackName) and GetFreshStack (with StackArn)
				client.Stacks["existing-stack"] = stack
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/existing-stack/abc123"] = stack
			},
			want: false,
		},
		"UPDATE_COMPLETE - not new": {
			stackName: "updated-stack",
			stackArn:  "arn:aws:cloudformation:us-west-2:123456789012:stack/updated-stack/abc123",
			setup: func(client *testutil.MockCFNClient) {
				stack := testutil.NewStackBuilder("updated-stack").
					WithStatus(types.StackStatusUpdateComplete).
					Build()
				// IsNewStack calls both StackExists (with StackName) and GetFreshStack (with StackArn)
				client.Stacks["updated-stack"] = stack
				client.Stacks["arn:aws:cloudformation:us-west-2:123456789012:stack/updated-stack/abc123"] = stack
			},
			want: false,
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			deployment := DeployInfo{
				StackName: tc.stackName,
				StackArn:  tc.stackArn,
			}

			// Execute
			got := deployment.IsNewStack(mockClient)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeployInfo_CreateChangeSet tests the CreateChangeSet method
func TestDeployInfo_CreateChangeSet(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		deployment *DeployInfo
		setup      func(*testutil.MockCFNClient)
		wantId     string
		wantErr    bool
		errMsg     string
	}{
		"create changeset with template body": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "test-changeset",
				Template:      `{"Resources": {}}`,
				IsNew:         false,
				Parameters: []types.Parameter{
					{ParameterKey: strPtr("Env"), ParameterValue: strPtr("prod")},
				},
				Tags: []types.Tag{
					{Key: strPtr("Environment"), Value: strPtr("Production")},
				},
			},
			setup: func(client *testutil.MockCFNClient) {
				// Mock will return default changeset ID
			},
			wantId:  "arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/12345678-1234-1234-1234-123456789012",
			wantErr: false,
		},
		"create changeset with template URL": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "test-changeset",
				TemplateUrl:   "https://s3.amazonaws.com/bucket/template.yaml",
				IsNew:         true,
			},
			setup: func(client *testutil.MockCFNClient) {
				// Mock will return default changeset ID
			},
			wantId:  "arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/12345678-1234-1234-1234-123456789012",
			wantErr: false,
		},
		"create changeset with use previous template": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "test-changeset",
				// No Template or TemplateUrl
				IsNew: false,
			},
			setup: func(client *testutil.MockCFNClient) {
				// Mock will return default changeset ID
			},
			wantId:  "arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/12345678-1234-1234-1234-123456789012",
			wantErr: false,
		},
		"create changeset fails": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "test-changeset",
				Template:      `{"Resources": {}}`,
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.InsufficientCapabilitiesException{
					Message: strPtr("Requires CAPABILITY_IAM"),
				})
			},
			wantErr: true,
			errMsg:  "Requires CAPABILITY_IAM",
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			gotId, err := tc.deployment.CreateChangeSet(mockClient)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.wantId, gotId)
		})
	}
}

// TestDeployInfo_GetChangeset tests the GetChangeset method
func TestDeployInfo_GetChangeset(t *testing.T) {
	t.Helper()

	now := time.Now()

	tests := map[string]struct {
		deployment *DeployInfo
		setup      func(*testutil.MockCFNClient)
		wantLen    int
		wantErr    bool
		errMsg     string
	}{
		"get single changeset": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "test-changeset",
			},
			setup: func(client *testutil.MockCFNClient) {
				changeset := &cloudformation.DescribeChangeSetOutput{
					ChangeSetId:   strPtr("arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/abc123"),
					ChangeSetName: strPtr("test-changeset"),
					StackId:       strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/def456"),
					StackName:     strPtr("test-stack"),
					Status:        types.ChangeSetStatusCreateComplete,
					CreationTime:  &now,
					Changes: []types.Change{
						{
							ResourceChange: &types.ResourceChange{
								Action:             types.ChangeActionAdd,
								LogicalResourceId:  strPtr("MyBucket"),
								ResourceType:       strPtr("AWS::S3::Bucket"),
								PhysicalResourceId: strPtr(""),
							},
						},
					},
				}
				client.WithChangeset("test-changeset", changeset)
			},
			wantLen: 1,
			wantErr: false,
		},
		"get paginated changeset": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "paginated-changeset",
			},
			setup: func(client *testutil.MockCFNClient) {
				// Custom function to simulate pagination
				callCount := 0
				client.DescribeChangeSetFn = func(ctx context.Context, input *cloudformation.DescribeChangeSetInput, opts ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
					callCount++
					if callCount == 1 {
						return &cloudformation.DescribeChangeSetOutput{
							ChangeSetId:   strPtr("arn:aws:cloudformation:us-west-2:123456789012:changeSet/paginated-changeset/abc123"),
							ChangeSetName: strPtr("paginated-changeset"),
							StackId:       strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/def456"),
							StackName:     strPtr("test-stack"),
							Status:        types.ChangeSetStatusCreateComplete,
							CreationTime:  &now,
							NextToken:     strPtr("token1"),
						}, nil
					}
					return &cloudformation.DescribeChangeSetOutput{
						ChangeSetId:   strPtr("arn:aws:cloudformation:us-west-2:123456789012:changeSet/paginated-changeset/abc123"),
						ChangeSetName: strPtr("paginated-changeset"),
						StackId:       strPtr("arn:aws:cloudformation:us-west-2:123456789012:stack/test-stack/def456"),
						StackName:     strPtr("test-stack"),
						Status:        types.ChangeSetStatusCreateComplete,
						CreationTime:  &now,
					}, nil
				}
			},
			wantLen: 2,
			wantErr: false,
		},
		"changeset not found": {
			deployment: &DeployInfo{
				StackName:     "test-stack",
				ChangesetName: "non-existent",
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.ChangeSetNotFoundException{
					Message: strPtr("ChangeSet [non-existent] does not exist"),
				})
			},
			wantLen: 0, // Error occurs before any results are appended
			wantErr: true,
			errMsg:  "ChangeSet [non-existent] does not exist",
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			got, err := tc.deployment.GetChangeset(mockClient)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, got, tc.wantLen)
		})
	}
}

// TestDeployInfo_DeleteStack tests the DeleteStack method
func TestDeployInfo_DeleteStack(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		stackName string
		setup     func(*testutil.MockCFNClient)
		want      bool
	}{
		"successful deletion": {
			stackName: "test-stack",
			setup: func(client *testutil.MockCFNClient) {
				// Mock will return successful deletion
			},
			want: true,
		},
		"deletion fails": {
			stackName: "protected-stack",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(&types.OperationInProgressException{
					Message: strPtr("Stack is being updated"),
				})
			},
			want: false,
		},
		"stack with termination protection": {
			stackName: "protected-stack",
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Stack [protected-stack] has termination protection enabled"))
			},
			want: false,
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			deployment := &DeployInfo{
				StackName: tc.stackName,
			}

			// Execute
			got := deployment.DeleteStack(mockClient)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestDeployInfo_GetExecutionTimesRefactored tests the GetExecutionTimes method with dependency injection
func TestDeployInfo_GetExecutionTimesRefactored(t *testing.T) {
	t.Helper()

	now := time.Now()

	tests := map[string]struct {
		deployment *DeployInfo
		setup      func(*testutil.MockCFNClient)
		want       map[string]map[string]time.Time
		wantErr    bool
		errMsg     string
	}{
		"get execution times for recent events": {
			deployment: &DeployInfo{
				StackName: "test-stack",
				Changeset: &ChangesetInfo{
					CreationTime: now.Add(-10 * time.Minute),
				},
			},
			setup: func(client *testutil.MockCFNClient) {
				events := []types.StackEvent{
					{
						EventId:           strPtr("event1"),
						LogicalResourceId: strPtr("MyBucket"),
						ResourceType:      strPtr("AWS::S3::Bucket"),
						ResourceStatus:    types.ResourceStatusCreateInProgress,
						Timestamp:         ptrTime(now.Add(-5 * time.Minute)),
					},
					{
						EventId:           strPtr("event2"),
						LogicalResourceId: strPtr("MyBucket"),
						ResourceType:      strPtr("AWS::S3::Bucket"),
						ResourceStatus:    types.ResourceStatusCreateComplete,
						Timestamp:         ptrTime(now.Add(-3 * time.Minute)),
					},
					{
						EventId:           strPtr("event3"),
						LogicalResourceId: strPtr("MyRole"),
						ResourceType:      strPtr("AWS::IAM::Role"),
						ResourceStatus:    types.ResourceStatusCreateInProgress,
						Timestamp:         ptrTime(now.Add(-4 * time.Minute)),
					},
					{
						EventId:           strPtr("event4"),
						LogicalResourceId: strPtr("MyRole"),
						ResourceType:      strPtr("AWS::IAM::Role"),
						ResourceStatus:    types.ResourceStatusCreateComplete,
						Timestamp:         ptrTime(now.Add(-2 * time.Minute)),
					},
					// Old event that should be filtered out
					{
						EventId:           strPtr("event5"),
						LogicalResourceId: strPtr("OldResource"),
						ResourceType:      strPtr("AWS::EC2::Instance"),
						ResourceStatus:    types.ResourceStatusCreateComplete,
						Timestamp:         ptrTime(now.Add(-15 * time.Minute)),
					},
				}
				client.WithStackEvents(events...)
			},
			want: map[string]map[string]time.Time{
				"AWS  S3  Bucket (MyBucket)": {
					string(types.ResourceStatusCreateInProgress): now.Add(-5 * time.Minute),
					string(types.ResourceStatusCreateComplete):   now.Add(-3 * time.Minute),
				},
				"AWS  IAM  Role (MyRole)": {
					string(types.ResourceStatusCreateInProgress): now.Add(-4 * time.Minute),
					string(types.ResourceStatusCreateComplete):   now.Add(-2 * time.Minute),
				},
			},
			wantErr: false,
		},
		"error retrieving events": {
			deployment: &DeployInfo{
				StackName: "test-stack",
				Changeset: &ChangesetInfo{
					CreationTime: now,
				},
			},
			setup: func(client *testutil.MockCFNClient) {
				client.WithError(errors.New("Access denied"))
			},
			wantErr: true,
			errMsg:  "Access denied",
		},
		"no events after changeset creation": {
			deployment: &DeployInfo{
				StackName: "test-stack",
				Changeset: &ChangesetInfo{
					CreationTime: now,
				},
			},
			setup: func(client *testutil.MockCFNClient) {
				events := []types.StackEvent{
					{
						EventId:           strPtr("event1"),
						LogicalResourceId: strPtr("OldResource"),
						ResourceType:      strPtr("AWS::EC2::Instance"),
						ResourceStatus:    types.ResourceStatusCreateComplete,
						Timestamp:         ptrTime(now.Add(-1 * time.Hour)),
					},
				}
				client.WithStackEvents(events...)
			},
			want:    map[string]map[string]time.Time{},
			wantErr: false,
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockClient := testutil.NewMockCFNClient()
			if tc.setup != nil {
				tc.setup(mockClient)
			}

			// Execute
			got, err := tc.deployment.GetExecutionTimes(mockClient)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
