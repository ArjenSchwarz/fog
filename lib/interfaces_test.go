package lib

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

// TestAWSClientsSatisfyInterfaces verifies that AWS SDK clients implement our interfaces
func TestAWSClientsSatisfyInterfaces(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name       string
		verifyFunc func(t *testing.T)
	}{
		"CloudFormation client satisfies CloudFormationDescribeStacksAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDescribeStacksAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationDescribeStackResourcesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDescribeStackResourcesAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationDescribeStackEventsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDescribeStackEventsAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationDeleteChangeSetAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDeleteChangeSetAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationExecuteChangeSetAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationExecuteChangeSetAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CFNDescribeStacksAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CFNDescribeStacksAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CFNListImportsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CFNListImportsAPI = (*cloudformation.Client)(nil)
			},
		},
		"EC2 client satisfies EC2DescribeNaclsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ EC2DescribeNaclsAPI = (*ec2.Client)(nil)
			},
		},
		"EC2 client satisfies EC2DescribeRouteTablesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ EC2DescribeRouteTablesAPI = (*ec2.Client)(nil)
			},
		},
		"EC2 client satisfies EC2DescribeManagedPrefixListsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ EC2DescribeManagedPrefixListsAPI = (*ec2.Client)(nil)
			},
		},
		"SSO Admin client satisfies SSOAdminListInstancesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ SSOAdminListInstancesAPI = (*ssoadmin.Client)(nil)
			},
		},
		"SSO Admin client satisfies SSOAdminListPermissionSetsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ SSOAdminListPermissionSetsAPI = (*ssoadmin.Client)(nil)
			},
		},
		"SSO Admin client satisfies SSOAdminListAccountAssignmentsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ SSOAdminListAccountAssignmentsAPI = (*ssoadmin.Client)(nil)
			},
		},
		"Organizations client satisfies OrganizationsListAccountsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ OrganizationsListAccountsAPI = (*organizations.Client)(nil)
			},
		},
		"S3 client satisfies S3UploadAPI": {
			verifyFunc: func(t *testing.T) {
				var _ S3UploadAPI = (*s3.Client)(nil)
			},
		},
		"S3 client satisfies S3HeadAPI": {
			verifyFunc: func(t *testing.T) {
				var _ S3HeadAPI = (*s3.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationCreateStackAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationCreateStackAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationUpdateStackAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationUpdateStackAPI = (*cloudformation.Client)(nil)
			},
		},
		"CloudFormation client satisfies CloudFormationDeleteStackAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDeleteStackAPI = (*cloudformation.Client)(nil)
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.verifyFunc(t)
		})
	}
}

// Mock implementations for testing interface compliance
type mockCFNInterfaceClient struct {
	describeStacksFn         func(context.Context, *cloudformation.DescribeStacksInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	describeStackResourcesFn func(context.Context, *cloudformation.DescribeStackResourcesInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
	describeStackEventsFn    func(context.Context, *cloudformation.DescribeStackEventsInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error)
	deleteChangeSetFn        func(context.Context, *cloudformation.DeleteChangeSetInput, ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
	executeChangeSetFn       func(context.Context, *cloudformation.ExecuteChangeSetInput, ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error)
	listImportsFn            func(context.Context, *cloudformation.ListImportsInput, ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error)
	createStackFn            func(context.Context, *cloudformation.CreateStackInput, ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
	updateStackFn            func(context.Context, *cloudformation.UpdateStackInput, ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
	deleteStackFn            func(context.Context, *cloudformation.DeleteStackInput, ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
}

func (m *mockCFNInterfaceClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if m.describeStacksFn != nil {
		return m.describeStacksFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStacksOutput{}, nil
}

func (m *mockCFNInterfaceClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	if m.describeStackResourcesFn != nil {
		return m.describeStackResourcesFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStackResourcesOutput{}, nil
}

func (m *mockCFNInterfaceClient) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	if m.describeStackEventsFn != nil {
		return m.describeStackEventsFn(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStackEventsOutput{}, nil
}

func (m *mockCFNInterfaceClient) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	if m.deleteChangeSetFn != nil {
		return m.deleteChangeSetFn(ctx, params, optFns...)
	}
	return &cloudformation.DeleteChangeSetOutput{}, nil
}

func (m *mockCFNInterfaceClient) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	if m.executeChangeSetFn != nil {
		return m.executeChangeSetFn(ctx, params, optFns...)
	}
	return &cloudformation.ExecuteChangeSetOutput{}, nil
}

func (m *mockCFNInterfaceClient) ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
	if m.listImportsFn != nil {
		return m.listImportsFn(ctx, params, optFns...)
	}
	return &cloudformation.ListImportsOutput{}, nil
}

func (m *mockCFNInterfaceClient) CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
	if m.createStackFn != nil {
		return m.createStackFn(ctx, params, optFns...)
	}
	return &cloudformation.CreateStackOutput{}, nil
}

func (m *mockCFNInterfaceClient) UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
	if m.updateStackFn != nil {
		return m.updateStackFn(ctx, params, optFns...)
	}
	return &cloudformation.UpdateStackOutput{}, nil
}

func (m *mockCFNInterfaceClient) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	if m.deleteStackFn != nil {
		return m.deleteStackFn(ctx, params, optFns...)
	}
	return &cloudformation.DeleteStackOutput{}, nil
}

// TestMockImplementations verifies that mock implementations satisfy interfaces
func TestMockImplementations(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		name       string
		verifyFunc func(t *testing.T)
	}{
		"mockCFNInterfaceClient satisfies CloudFormationDescribeStacksAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDescribeStacksAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationDescribeStackResourcesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDescribeStackResourcesAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationDescribeStackEventsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDescribeStackEventsAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationDeleteChangeSetAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDeleteChangeSetAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationExecuteChangeSetAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationExecuteChangeSetAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CFNDescribeStacksAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CFNDescribeStacksAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CFNListImportsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CFNListImportsAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CFNExportsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CFNExportsAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationCreateStackAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationCreateStackAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationUpdateStackAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationUpdateStackAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CloudFormationDeleteStackAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CloudFormationDeleteStackAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockCFNInterfaceClient satisfies CFNStackOperationsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ CFNStackOperationsAPI = (*mockCFNInterfaceClient)(nil)
			},
		},
		"mockS3Client satisfies S3UploadAPI": {
			verifyFunc: func(t *testing.T) {
				var _ S3UploadAPI = (*mockS3Client)(nil)
			},
		},
		"mockS3Client satisfies S3HeadAPI": {
			verifyFunc: func(t *testing.T) {
				var _ S3HeadAPI = (*mockS3Client)(nil)
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.verifyFunc(t)
		})
	}
}

// Test that composed interfaces work correctly
func TestComposedInterfaces(t *testing.T) {
	t.Helper()

	t.Run("CFNExportsAPI composition", func(t *testing.T) {
		t.Parallel()

		// Create a mock that implements both component interfaces
		mock := &mockCFNInterfaceClient{
			describeStacksFn: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
				return &cloudformation.DescribeStacksOutput{}, nil
			},
			listImportsFn: func(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
				return &cloudformation.ListImportsOutput{}, nil
			},
		}

		// Verify it can be used as CFNExportsAPI
		var exportsAPI CFNExportsAPI = mock

		// Call both methods to ensure they work
		ctx := context.Background()
		_, err := exportsAPI.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{})
		if err != nil {
			t.Errorf("DescribeStacks() error = %v", err)
		}

		_, err = exportsAPI.ListImports(ctx, &cloudformation.ListImportsInput{})
		if err != nil {
			t.Errorf("ListImports() error = %v", err)
		}
	})

	t.Run("CFNStackOperationsAPI composition", func(t *testing.T) {
		t.Parallel()

		// Create a mock that implements all component interfaces
		mock := &mockCFNInterfaceClient{
			describeStacksFn: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
				return &cloudformation.DescribeStacksOutput{}, nil
			},
			createStackFn: func(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
				return &cloudformation.CreateStackOutput{}, nil
			},
			updateStackFn: func(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
				return &cloudformation.UpdateStackOutput{}, nil
			},
			deleteStackFn: func(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
				return &cloudformation.DeleteStackOutput{}, nil
			},
		}

		// Verify it can be used as CFNStackOperationsAPI
		var stackOpsAPI CFNStackOperationsAPI = mock

		// Call all methods to ensure they work
		ctx := context.Background()
		_, err := stackOpsAPI.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{})
		if err != nil {
			t.Errorf("DescribeStacks() error = %v", err)
		}

		_, err = stackOpsAPI.CreateStack(ctx, &cloudformation.CreateStackInput{})
		if err != nil {
			t.Errorf("CreateStack() error = %v", err)
		}

		_, err = stackOpsAPI.UpdateStack(ctx, &cloudformation.UpdateStackInput{})
		if err != nil {
			t.Errorf("UpdateStack() error = %v", err)
		}

		_, err = stackOpsAPI.DeleteStack(ctx, &cloudformation.DeleteStackInput{})
		if err != nil {
			t.Errorf("DeleteStack() error = %v", err)
		}
	})
}

// mockS3Client for testing S3 interfaces (will be used after adding S3 interfaces)
type mockS3Client struct {
	putObjectFn  func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	headObjectFn func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

func (m *mockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.putObjectFn != nil {
		return m.putObjectFn(ctx, params, optFns...)
	}
	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectFn != nil {
		return m.headObjectFn(ctx, params, optFns...)
	}
	return &s3.HeadObjectOutput{}, nil
}

// Additional mock implementations for other services
type mockEC2Client struct {
	describeNetworkAclsFn        func(context.Context, *ec2.DescribeNetworkAclsInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)
	describeRouteTablesFn        func(context.Context, *ec2.DescribeRouteTablesInput, ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	describeManagedPrefixListsFn func(context.Context, *ec2.DescribeManagedPrefixListsInput, ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error)
}

func (m *mockEC2Client) DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
	if m.describeNetworkAclsFn != nil {
		return m.describeNetworkAclsFn(ctx, params, optFns...)
	}
	return &ec2.DescribeNetworkAclsOutput{}, nil
}

func (m *mockEC2Client) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	if m.describeRouteTablesFn != nil {
		return m.describeRouteTablesFn(ctx, params, optFns...)
	}
	return &ec2.DescribeRouteTablesOutput{}, nil
}

func (m *mockEC2Client) DescribeManagedPrefixLists(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error) {
	if m.describeManagedPrefixListsFn != nil {
		return m.describeManagedPrefixListsFn(ctx, params, optFns...)
	}
	return &ec2.DescribeManagedPrefixListsOutput{}, nil
}

// TestEC2MockImplementations verifies EC2 mock implementations
func TestEC2MockImplementations(t *testing.T) {
	t.Helper()

	tests := map[string]struct {
		verifyFunc func(t *testing.T)
	}{
		"mockEC2Client satisfies EC2DescribeNaclsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ EC2DescribeNaclsAPI = (*mockEC2Client)(nil)
			},
		},
		"mockEC2Client satisfies EC2DescribeRouteTablesAPI": {
			verifyFunc: func(t *testing.T) {
				var _ EC2DescribeRouteTablesAPI = (*mockEC2Client)(nil)
			},
		},
		"mockEC2Client satisfies EC2DescribeManagedPrefixListsAPI": {
			verifyFunc: func(t *testing.T) {
				var _ EC2DescribeManagedPrefixListsAPI = (*mockEC2Client)(nil)
			},
		},
	}

	for name, tc := range tests {
		// capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			tc.verifyFunc(t)
		})
	}
}
