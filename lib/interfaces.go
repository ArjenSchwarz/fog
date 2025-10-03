package lib

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

type EC2DescribeNaclsAPI interface {
	DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)
}

type EC2DescribeRouteTablesAPI interface {
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
}

type EC2DescribeManagedPrefixListsAPI interface {
	DescribeManagedPrefixLists(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error)
}

type SSOAdminListInstancesAPI interface {
	ListInstances(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error)
}

type SSOAdminListPermissionSetsAPI interface {
	ListPermissionSets(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error)
}

type SSOAdminListAccountAssignmentsAPI interface {
	ListAccountAssignments(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error)
}

type OrganizationsListAccountsAPI interface {
	ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsOutput, error)
}

type CloudFormationDescribeStacksAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type CloudFormationDescribeStackResourcesAPI interface {
	DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
}

type CloudFormationDescribeStackEventsAPI interface {
	DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error)
}

type CloudFormationDeleteChangeSetAPI interface {
	DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
}

type CloudFormationExecuteChangeSetAPI interface {
	ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error)
}

type CFNDescribeStacksAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

type CFNListImportsAPI interface {
	ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error)
}

// CFNExportsAPI combines the CloudFormation operations used for export retrieval
type CFNExportsAPI interface {
	CFNDescribeStacksAPI
	CFNListImportsAPI
}

// S3UploadAPI defines the S3 operations for uploading objects
type S3UploadAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// S3HeadAPI defines the S3 operations for retrieving object metadata
type S3HeadAPI interface {
	HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
}

// CloudFormationCreateStackAPI defines the CloudFormation CreateStack operation
type CloudFormationCreateStackAPI interface {
	CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
}

// CloudFormationUpdateStackAPI defines the CloudFormation UpdateStack operation
type CloudFormationUpdateStackAPI interface {
	UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
}

// CloudFormationDeleteStackAPI defines the CloudFormation DeleteStack operation
type CloudFormationDeleteStackAPI interface {
	DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
}

// CFNStackOperationsAPI combines CloudFormation operations for stack lifecycle management
type CFNStackOperationsAPI interface {
	CloudFormationDescribeStacksAPI
	CloudFormationCreateStackAPI
	CloudFormationUpdateStackAPI
	CloudFormationDeleteStackAPI
}

// CloudFormationCreateChangeSetAPI defines the CloudFormation CreateChangeSet operation
type CloudFormationCreateChangeSetAPI interface {
	CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error)
}

// CloudFormationDescribeChangeSetAPI defines the CloudFormation DescribeChangeSet operation
type CloudFormationDescribeChangeSetAPI interface {
	DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error)
}
