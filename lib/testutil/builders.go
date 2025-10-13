package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// MockCFNClient is a mock CloudFormation client for testing
type MockCFNClient struct {
	Stacks                   map[string]*types.Stack
	StackEvents              []types.StackEvent
	StackResources           []types.StackResource
	Changesets               map[string]*cloudformation.DescribeChangeSetOutput
	Error                    error
	DescribeStacksFn         func(context.Context, *cloudformation.DescribeStacksInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	CreateStackFn            func(context.Context, *cloudformation.CreateStackInput, ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error)
	UpdateStackFn            func(context.Context, *cloudformation.UpdateStackInput, ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error)
	DeleteStackFn            func(context.Context, *cloudformation.DeleteStackInput, ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error)
	DescribeStackEventsFn    func(context.Context, *cloudformation.DescribeStackEventsInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error)
	DescribeStackResourcesFn func(context.Context, *cloudformation.DescribeStackResourcesInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
	CreateChangeSetFn        func(context.Context, *cloudformation.CreateChangeSetInput, ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error)
	DescribeChangeSetFn      func(context.Context, *cloudformation.DescribeChangeSetInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error)
	ListStacksFn             func(context.Context, *cloudformation.ListStacksInput, ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
	GetTemplateFn            func(context.Context, *cloudformation.GetTemplateInput, ...func(*cloudformation.Options)) (*cloudformation.GetTemplateOutput, error)
	ValidateTemplateFn       func(context.Context, *cloudformation.ValidateTemplateInput, ...func(*cloudformation.Options)) (*cloudformation.ValidateTemplateOutput, error)
	DeleteChangeSetFn        func(context.Context, *cloudformation.DeleteChangeSetInput, ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error)
	ExecuteChangeSetFn       func(context.Context, *cloudformation.ExecuteChangeSetInput, ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error)
	ListImportsFn            func(context.Context, *cloudformation.ListImportsInput, ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error)
}

// NewMockCFNClient creates a new mock CloudFormation client with sensible defaults
func NewMockCFNClient() *MockCFNClient {
	return &MockCFNClient{
		Stacks:         make(map[string]*types.Stack),
		StackEvents:    []types.StackEvent{},
		StackResources: []types.StackResource{},
		Changesets:     make(map[string]*cloudformation.DescribeChangeSetOutput),
	}
}

// WithStack adds a stack to the mock client
func (m *MockCFNClient) WithStack(stack *types.Stack) *MockCFNClient {
	m.Stacks[*stack.StackName] = stack
	return m
}

// WithError configures the mock to return an error
func (m *MockCFNClient) WithError(err error) *MockCFNClient {
	m.Error = err
	return m
}

// WithStackEvents adds stack events to the mock client
func (m *MockCFNClient) WithStackEvents(events ...types.StackEvent) *MockCFNClient {
	m.StackEvents = append(m.StackEvents, events...)
	return m
}

// WithStackResources adds stack resources to the mock client
func (m *MockCFNClient) WithStackResources(resources ...types.StackResource) *MockCFNClient {
	m.StackResources = append(m.StackResources, resources...)
	return m
}

// WithChangeset adds a changeset to the mock client
func (m *MockCFNClient) WithChangeset(name string, changeset *cloudformation.DescribeChangeSetOutput) *MockCFNClient {
	m.Changesets[name] = changeset
	return m
}

// DescribeStacks implements the CloudFormationDescribeStacksAPI interface
func (m *MockCFNClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if m.DescribeStacksFn != nil {
		return m.DescribeStacksFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	if params.StackName != nil {
		if stack, ok := m.Stacks[*params.StackName]; ok {
			return &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{*stack},
			}, nil
		}
		return nil, fmt.Errorf("stack with name %s does not exist", *params.StackName)
	}

	// Return all stacks if no specific stack name is provided
	stacks := make([]types.Stack, 0, len(m.Stacks))
	for _, stack := range m.Stacks {
		stacks = append(stacks, *stack)
	}

	return &cloudformation.DescribeStacksOutput{
		Stacks: stacks,
	}, nil
}

// DescribeStackEvents implements the CloudFormationDescribeStackEventsAPI interface
func (m *MockCFNClient) DescribeStackEvents(ctx context.Context, params *cloudformation.DescribeStackEventsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackEventsOutput, error) {
	if m.DescribeStackEventsFn != nil {
		return m.DescribeStackEventsFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	return &cloudformation.DescribeStackEventsOutput{
		StackEvents: m.StackEvents,
	}, nil
}

// DescribeStackResources implements the CloudFormationDescribeStackResourcesAPI interface
func (m *MockCFNClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	if m.DescribeStackResourcesFn != nil {
		return m.DescribeStackResourcesFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	return &cloudformation.DescribeStackResourcesOutput{
		StackResources: m.StackResources,
	}, nil
}

// CreateStack implements the CloudFormationCreateStackAPI interface
func (m *MockCFNClient) CreateStack(ctx context.Context, params *cloudformation.CreateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateStackOutput, error) {
	if m.CreateStackFn != nil {
		return m.CreateStackFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	stackId := aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + *params.StackName + "/12345678-1234-1234-1234-123456789012")
	return &cloudformation.CreateStackOutput{
		StackId: stackId,
	}, nil
}

// UpdateStack implements the CloudFormationUpdateStackAPI interface
func (m *MockCFNClient) UpdateStack(ctx context.Context, params *cloudformation.UpdateStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.UpdateStackOutput, error) {
	if m.UpdateStackFn != nil {
		return m.UpdateStackFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	stackId := aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + *params.StackName + "/12345678-1234-1234-1234-123456789012")
	return &cloudformation.UpdateStackOutput{
		StackId: stackId,
	}, nil
}

// DeleteStack implements the CloudFormationDeleteStackAPI interface
func (m *MockCFNClient) DeleteStack(ctx context.Context, params *cloudformation.DeleteStackInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteStackOutput, error) {
	if m.DeleteStackFn != nil {
		return m.DeleteStackFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	// Remove the stack from our mock storage
	if params.StackName != nil {
		delete(m.Stacks, *params.StackName)
	}

	return &cloudformation.DeleteStackOutput{}, nil
}

// CreateChangeSet implements the CloudFormationCreateChangeSetAPI interface
func (m *MockCFNClient) CreateChangeSet(ctx context.Context, params *cloudformation.CreateChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.CreateChangeSetOutput, error) {
	if m.CreateChangeSetFn != nil {
		return m.CreateChangeSetFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	changeSetId := aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/" + *params.ChangeSetName + "/12345678-1234-1234-1234-123456789012")
	return &cloudformation.CreateChangeSetOutput{
		Id:      changeSetId,
		StackId: aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + *params.StackName + "/12345678-1234-1234-1234-123456789012"),
	}, nil
}

// DescribeChangeSet implements the CloudFormationDescribeChangeSetAPI interface
func (m *MockCFNClient) DescribeChangeSet(ctx context.Context, params *cloudformation.DescribeChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeChangeSetOutput, error) {
	if m.DescribeChangeSetFn != nil {
		return m.DescribeChangeSetFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	if params.ChangeSetName != nil {
		if changeset, ok := m.Changesets[*params.ChangeSetName]; ok {
			return changeset, nil
		}
	}

	// Return a default changeset if not found
	return &cloudformation.DescribeChangeSetOutput{
		ChangeSetId:   aws.String("arn:aws:cloudformation:us-west-2:123456789012:changeSet/test-changeset/12345678-1234-1234-1234-123456789012"),
		ChangeSetName: params.ChangeSetName,
		StackId:       aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + *params.StackName + "/12345678-1234-1234-1234-123456789012"),
		StackName:     params.StackName,
		Status:        types.ChangeSetStatusCreateComplete,
		CreationTime:  aws.Time(time.Now()),
	}, nil
}

// DeleteChangeSet implements the CloudFormationDeleteChangeSetAPI interface
func (m *MockCFNClient) DeleteChangeSet(ctx context.Context, params *cloudformation.DeleteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DeleteChangeSetOutput, error) {
	if m.DeleteChangeSetFn != nil {
		return m.DeleteChangeSetFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	// Remove the changeset from our mock storage
	if params.ChangeSetName != nil {
		delete(m.Changesets, *params.ChangeSetName)
	}

	return &cloudformation.DeleteChangeSetOutput{}, nil
}

// ExecuteChangeSet implements the CloudFormationExecuteChangeSetAPI interface
func (m *MockCFNClient) ExecuteChangeSet(ctx context.Context, params *cloudformation.ExecuteChangeSetInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ExecuteChangeSetOutput, error) {
	if m.ExecuteChangeSetFn != nil {
		return m.ExecuteChangeSetFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	return &cloudformation.ExecuteChangeSetOutput{}, nil
}

// ListImports implements the CFNListImportsAPI interface
func (m *MockCFNClient) ListImports(ctx context.Context, params *cloudformation.ListImportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListImportsOutput, error) {
	if m.ListImportsFn != nil {
		return m.ListImportsFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	// Return empty imports list by default
	return &cloudformation.ListImportsOutput{
		Imports: []string{},
	}, nil
}

// MockEC2Client is a mock EC2 client for testing
type MockEC2Client struct {
	RouteTables           []ec2types.RouteTable
	NetworkAcls           []ec2types.NetworkAcl
	Error                 error
	DescribeRouteTablesFn func(context.Context, *ec2.DescribeRouteTablesInput, ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)
	DescribeNetworkAclsFn func(context.Context, *ec2.DescribeNetworkAclsInput, ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)
}

// NewMockEC2Client creates a new mock EC2 client with sensible defaults
func NewMockEC2Client() *MockEC2Client {
	return &MockEC2Client{
		RouteTables: []ec2types.RouteTable{},
		NetworkAcls: []ec2types.NetworkAcl{},
	}
}

// WithRouteTable adds a route table to the mock client
func (m *MockEC2Client) WithRouteTable(rt ec2types.RouteTable) *MockEC2Client {
	m.RouteTables = append(m.RouteTables, rt)
	return m
}

// WithNetworkAcl adds a network ACL to the mock client
func (m *MockEC2Client) WithNetworkAcl(acl ec2types.NetworkAcl) *MockEC2Client {
	m.NetworkAcls = append(m.NetworkAcls, acl)
	return m
}

// WithError configures the mock to return an error
func (m *MockEC2Client) WithError(err error) *MockEC2Client {
	m.Error = err
	return m
}

// DescribeRouteTables implements the EC2DescribeRouteTablesAPI interface
func (m *MockEC2Client) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
	if m.DescribeRouteTablesFn != nil {
		return m.DescribeRouteTablesFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	return &ec2.DescribeRouteTablesOutput{
		RouteTables: m.RouteTables,
	}, nil
}

// DescribeNetworkAcls implements the EC2DescribeNaclsAPI interface
func (m *MockEC2Client) DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
	if m.DescribeNetworkAclsFn != nil {
		return m.DescribeNetworkAclsFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	return &ec2.DescribeNetworkAclsOutput{
		NetworkAcls: m.NetworkAcls,
	}, nil
}

// MockS3Client is a mock S3 client for testing
type MockS3Client struct {
	Objects       map[string][]byte
	Error         error
	PutObjectFn   func(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObjectFn   func(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadObjectFn  func(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	ListObjectsFn func(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// NewMockS3Client creates a new mock S3 client with sensible defaults
func NewMockS3Client() *MockS3Client {
	return &MockS3Client{
		Objects: make(map[string][]byte),
	}
}

// WithObject adds an object to the mock client
func (m *MockS3Client) WithObject(key string, data []byte) *MockS3Client {
	m.Objects[key] = data
	return m
}

// WithError configures the mock to return an error
func (m *MockS3Client) WithError(err error) *MockS3Client {
	m.Error = err
	return m
}

// PutObject implements the S3UploadAPI interface
func (m *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	if m.PutObjectFn != nil {
		return m.PutObjectFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	// Store the object in our mock storage
	// Note: In a real test, you might want to read from params.Body
	key := *params.Key
	m.Objects[key] = []byte{} // Simplified for testing

	return &s3.PutObjectOutput{
		ETag: aws.String("mock-etag"),
	}, nil
}

// HeadObject implements the S3HeadAPI interface
func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.HeadObjectFn != nil {
		return m.HeadObjectFn(ctx, params, optFns...)
	}

	if m.Error != nil {
		return nil, m.Error
	}

	if _, ok := m.Objects[*params.Key]; ok {
		return &s3.HeadObjectOutput{
			ContentLength: aws.Int64(int64(len(m.Objects[*params.Key]))),
		}, nil
	}

	return nil, fmt.Errorf("the specified key does not exist")
}

// StackBuilder builds test stacks with sensible defaults
type StackBuilder struct {
	stack types.Stack
}

// NewStackBuilder creates a new stack builder with default values
func NewStackBuilder(name string) *StackBuilder {
	now := time.Now()
	return &StackBuilder{
		stack: types.Stack{
			StackName:    aws.String(name),
			StackId:      aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + name + "/12345678-1234-1234-1234-123456789012"),
			StackStatus:  types.StackStatusCreateComplete,
			CreationTime: aws.Time(now),
		},
	}
}

// WithStatus sets the stack status
func (b *StackBuilder) WithStatus(status types.StackStatus) *StackBuilder {
	b.stack.StackStatus = status
	return b
}

// WithParameter adds a parameter to the stack
func (b *StackBuilder) WithParameter(key, value string) *StackBuilder {
	param := types.Parameter{
		ParameterKey:   aws.String(key),
		ParameterValue: aws.String(value),
	}
	b.stack.Parameters = append(b.stack.Parameters, param)
	return b
}

// WithOutput adds an output to the stack
func (b *StackBuilder) WithOutput(key, value string) *StackBuilder {
	output := types.Output{
		OutputKey:   aws.String(key),
		OutputValue: aws.String(value),
	}
	b.stack.Outputs = append(b.stack.Outputs, output)
	return b
}

// WithTag adds a tag to the stack
func (b *StackBuilder) WithTag(key, value string) *StackBuilder {
	tag := types.Tag{
		Key:   aws.String(key),
		Value: aws.String(value),
	}
	b.stack.Tags = append(b.stack.Tags, tag)
	return b
}

// WithCapability adds a capability to the stack
func (b *StackBuilder) WithCapability(capability types.Capability) *StackBuilder {
	b.stack.Capabilities = append(b.stack.Capabilities, capability)
	return b
}

// WithDescription sets the stack description
func (b *StackBuilder) WithDescription(desc string) *StackBuilder {
	b.stack.Description = aws.String(desc)
	return b
}

// WithDriftStatus sets the drift status
func (b *StackBuilder) WithDriftStatus(status types.StackDriftStatus) *StackBuilder {
	b.stack.DriftInformation = &types.StackDriftInformation{
		StackDriftStatus: status,
	}
	return b
}

// Build returns the constructed stack
func (b *StackBuilder) Build() *types.Stack {
	return &b.stack
}

// StackEventBuilder builds test stack events
type StackEventBuilder struct {
	event types.StackEvent
}

// NewStackEventBuilder creates a new stack event builder
func NewStackEventBuilder(stackName, logicalResourceId string) *StackEventBuilder {
	now := time.Now()
	eventId := "event-" + now.Format("20060102-150405")

	return &StackEventBuilder{
		event: types.StackEvent{
			EventId:            aws.String(eventId),
			StackName:          aws.String(stackName),
			StackId:            aws.String("arn:aws:cloudformation:us-west-2:123456789012:stack/" + stackName + "/12345678-1234-1234-1234-123456789012"),
			LogicalResourceId:  aws.String(logicalResourceId),
			PhysicalResourceId: aws.String("physical-" + logicalResourceId),
			ResourceType:       aws.String("AWS::S3::Bucket"),
			Timestamp:          aws.Time(now),
			ResourceStatus:     types.ResourceStatusCreateComplete,
		},
	}
}

// WithStatus sets the resource status
func (b *StackEventBuilder) WithStatus(status types.ResourceStatus) *StackEventBuilder {
	b.event.ResourceStatus = status
	return b
}

// WithResourceType sets the resource type
func (b *StackEventBuilder) WithResourceType(resourceType string) *StackEventBuilder {
	b.event.ResourceType = aws.String(resourceType)
	return b
}

// WithStatusReason sets the status reason
func (b *StackEventBuilder) WithStatusReason(reason string) *StackEventBuilder {
	b.event.ResourceStatusReason = aws.String(reason)
	return b
}

// WithTimestamp sets the event timestamp
func (b *StackEventBuilder) WithTimestamp(t time.Time) *StackEventBuilder {
	b.event.Timestamp = aws.Time(t)
	return b
}

// Build returns the constructed stack event
func (b *StackEventBuilder) Build() types.StackEvent {
	return b.event
}
