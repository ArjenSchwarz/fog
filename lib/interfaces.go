package lib

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

// EC2DescribeNaclsAPI defines the EC2 DescribeNetworkAcls operation.
type EC2DescribeNaclsAPI interface {
	DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)
}

// EC2DescribeRouteTablesAPI defines the EC2 DescribeRouteTables operation.
type EC2DescribeRouteTablesAPI interface {
	DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
}

// EC2DescribeManagedPrefixListsAPI defines the EC2 DescribeManagedPrefixLists operation.
type EC2DescribeManagedPrefixListsAPI interface {
	DescribeManagedPrefixLists(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error)
}

// EC2SearchTransitGatewayRoutesAPI defines the EC2 SearchTransitGatewayRoutes operation.
type EC2SearchTransitGatewayRoutesAPI interface {
	SearchTransitGatewayRoutes(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error)
}

// SSOAdminListInstancesAPI defines the SSO Admin ListInstances operation.
type SSOAdminListInstancesAPI interface {
	ListInstances(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error)
}

// SSOAdminListPermissionSetsAPI defines the SSO Admin ListPermissionSets operation.
type SSOAdminListPermissionSetsAPI interface {
	ListPermissionSets(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error)
}

// SSOAdminListAccountAssignmentsAPI defines the SSO Admin ListAccountAssignments operation.
type SSOAdminListAccountAssignmentsAPI interface {
	ListAccountAssignments(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error)
}

// OrganizationsListAccountsAPI defines the Organizations ListAccounts operation.
type OrganizationsListAccountsAPI interface {
	ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsOutput, error)
}

// CloudFormationDescribeStacksAPI defines the CloudFormation DescribeStacks operation.
type CloudFormationDescribeStacksAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

// CloudFormationDescribeStackResourcesAPI defines the CloudFormation DescribeStackResources operation.
type CloudFormationDescribeStackResourcesAPI interface {
	DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
}

// CloudFormationDescribeStackEventsAPI defines the CloudFormation DescribeStackEvents operation.
type CloudFormationDescribeStackEventsAPI interface {
	DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error)
}

// CloudFormationDeleteChangeSetAPI defines the CloudFormation DeleteChangeSet operation.
type CloudFormationDeleteChangeSetAPI interface {
	DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
}

// CloudFormationExecuteChangeSetAPI defines the CloudFormation ExecuteChangeSet operation.
type CloudFormationExecuteChangeSetAPI interface {
	ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error)
}

// CFNDescribeStacksAPI defines the CloudFormation DescribeStacks operation for export retrieval.
type CFNDescribeStacksAPI interface {
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

// CFNListImportsAPI defines the CloudFormation ListImports operation.
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

// CloudFormationDetectStackDriftAPI defines the CloudFormation DetectStackDrift operation
type CloudFormationDetectStackDriftAPI interface {
	DetectStackDrift(ctx context.Context, params *cloudformation.DetectStackDriftInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DetectStackDriftOutput, error)
}

// CloudFormationDescribeStackDriftDetectionStatusAPI defines the CloudFormation DescribeStackDriftDetectionStatus operation
type CloudFormationDescribeStackDriftDetectionStatusAPI interface {
	DescribeStackDriftDetectionStatus(ctx context.Context, params *cloudformation.DescribeStackDriftDetectionStatusInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackDriftDetectionStatusOutput, error)
}

// CloudFormationDescribeStackResourceDriftsAPI defines the CloudFormation DescribeStackResourceDrifts operation
type CloudFormationDescribeStackResourceDriftsAPI interface {
	DescribeStackResourceDrifts(ctx context.Context, params *cloudformation.DescribeStackResourceDriftsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourceDriftsOutput, error)
}

// CloudFormationGetTemplateAPI defines the CloudFormation GetTemplate operation
type CloudFormationGetTemplateAPI interface {
	GetTemplate(ctx context.Context, params *cloudformation.GetTemplateInput, optFns ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error)
}

// CloudFormationListExportsAPI defines the CloudFormation ListExports operation
type CloudFormationListExportsAPI interface {
	ListExports(ctx context.Context, params *cloudformation.ListExportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListExportsOutput, error)
}
