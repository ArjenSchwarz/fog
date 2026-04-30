package lib

// Tests in this file cover the behaviour of GetResources, including
// successful retrieval of resources, retry handling on throttling
// errors, and failure behaviour for other API errors.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCloudFormationClient implements the CloudFormation interfaces for testing.
// Each field holds the output or error that should be returned for a call.
type mockCloudFormationClient struct {
	describeStacksOutput          cloudformation.DescribeStacksOutput
	describeStacksErr             error
	describeStackResourcesOutputs []cloudformation.DescribeStackResourcesOutput
	describeStackResourcesErrs    []error
	describeStackResourcesCalls   int
}

func (m *mockCloudFormationClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return &m.describeStacksOutput, m.describeStacksErr
}

func (m *mockCloudFormationClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	idx := m.describeStackResourcesCalls
	m.describeStackResourcesCalls++
	var out cloudformation.DescribeStackResourcesOutput
	if idx < len(m.describeStackResourcesOutputs) {
		out = m.describeStackResourcesOutputs[idx]
	}
	var err error
	if idx < len(m.describeStackResourcesErrs) {
		err = m.describeStackResourcesErrs[idx]
	}
	return &out, err
}

// mockAPIError is used to simulate throttling and other API errors.
type mockAPIError struct {
	code    string
	message string
	fault   smithy.ErrorFault
}

func (e mockAPIError) Error() string                 { return e.message }
func (e mockAPIError) ErrorCode() string             { return e.code }
func (e mockAPIError) ErrorMessage() string          { return e.message }
func (e mockAPIError) ErrorFault() smithy.ErrorFault { return e.fault }

// TestGetResourcesSuccess verifies that resources are returned when both API calls succeed.
func TestGetResourcesSuccess(t *testing.T) {
	stackName := "test-stack"
	stacksOut := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{{StackName: aws.String(stackName)}},
	}
	resOut := cloudformation.DescribeStackResourcesOutput{
		StackResources: []types.StackResource{
			{LogicalResourceId: aws.String("Res1"), PhysicalResourceId: aws.String("phys1"), ResourceType: aws.String("AWS::S3::Bucket"), ResourceStatus: types.ResourceStatusCreateComplete},
			{LogicalResourceId: aws.String("Res2"), PhysicalResourceId: aws.String("phys2"), ResourceType: aws.String("AWS::IAM::Role"), ResourceStatus: types.ResourceStatusCreateComplete},
		},
	}
	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{resOut},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []CfnResource{
		{StackName: stackName, Type: "AWS::S3::Bucket", ResourceID: "phys1", LogicalID: "Res1", Status: string(types.ResourceStatusCreateComplete)},
		{StackName: stackName, Type: "AWS::IAM::Role", ResourceID: "phys2", LogicalID: "Res2", Status: string(types.ResourceStatusCreateComplete)},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d resources, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("resource %d = %#v, want %#v", i, got[i], want[i])
		}
	}
	if mock.describeStackResourcesCalls != 1 {
		t.Errorf("expected 1 call, got %d", mock.describeStackResourcesCalls)
	}
}

// TestGetResourcesThrottlingRetry ensures that throttling errors trigger a retry after waiting.
func TestGetResourcesThrottlingRetry(t *testing.T) {
	stackName := "throttle-stack"
	stacksOut := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{{StackName: aws.String(stackName)}},
	}
	resOut := cloudformation.DescribeStackResourcesOutput{
		StackResources: []types.StackResource{{LogicalResourceId: aws.String("Res"), PhysicalResourceId: aws.String("phys"), ResourceType: aws.String("AWS::S3::Bucket"), ResourceStatus: types.ResourceStatusCreateComplete}},
	}
	throttleErr := mockAPIError{code: "Throttling", message: "Rate exceeded", fault: smithy.FaultServer}

	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{{}, resOut},
		describeStackResourcesErrs:    []error{throttleErr, nil},
	}

	start := time.Now()
	got, err := GetResources(context.Background(), &stackName, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if time.Since(start) < 5*time.Second {
		t.Errorf("expected sleep during retry")
	}
	want := []CfnResource{{StackName: stackName, Type: "AWS::S3::Bucket", ResourceID: "phys", LogicalID: "Res", Status: string(types.ResourceStatusCreateComplete)}}

	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("GetResources() = %#v, want %#v", got, want)
	}
	if mock.describeStackResourcesCalls != 2 {
		t.Errorf("expected 2 calls, got %d", mock.describeStackResourcesCalls)
	}
}

// paginatingMockClient supports multi-page DescribeStacks responses.
// Pages are keyed by NextToken ("" for the first call).
type paginatingMockClient struct {
	pages                         map[string]cloudformation.DescribeStacksOutput
	describeStackResourcesOutputs []cloudformation.DescribeStackResourcesOutput
	describeStackResourcesErrs    []error
	describeStackResourcesCalls   int
}

func (m *paginatingMockClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	token := ""
	if params.NextToken != nil {
		token = *params.NextToken
	}
	out := m.pages[token]
	return &out, nil
}

func (m *paginatingMockClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	idx := m.describeStackResourcesCalls
	m.describeStackResourcesCalls++
	var out cloudformation.DescribeStackResourcesOutput
	if idx < len(m.describeStackResourcesOutputs) {
		out = m.describeStackResourcesOutputs[idx]
	}
	var err error
	if idx < len(m.describeStackResourcesErrs) {
		err = m.describeStackResourcesErrs[idx]
	}
	return &out, err
}

// TestGetResourcesPagination verifies that stacks from multiple DescribeStacks pages are all processed.
func TestGetResourcesPagination(t *testing.T) {
	stackName := ""
	mock := &paginatingMockClient{
		pages: map[string]cloudformation.DescribeStacksOutput{
			"": {
				Stacks:    []types.Stack{{StackName: aws.String("stack-page1")}},
				NextToken: aws.String("token2"),
			},
			"token2": {
				Stacks:    []types.Stack{{StackName: aws.String("stack-page2")}},
				NextToken: aws.String("token3"),
			},
			"token3": {
				Stacks: []types.Stack{{StackName: aws.String("stack-page3")}},
			},
		},
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{
			{StackResources: []types.StackResource{
				{LogicalResourceId: aws.String("R1"), PhysicalResourceId: aws.String("p1"), ResourceType: aws.String("AWS::S3::Bucket"), ResourceStatus: types.ResourceStatusCreateComplete},
			}},
			{StackResources: []types.StackResource{
				{LogicalResourceId: aws.String("R2"), PhysicalResourceId: aws.String("p2"), ResourceType: aws.String("AWS::Lambda::Function"), ResourceStatus: types.ResourceStatusCreateComplete},
			}},
			{StackResources: []types.StackResource{
				{LogicalResourceId: aws.String("R3"), PhysicalResourceId: aws.String("p3"), ResourceType: aws.String("AWS::IAM::Role"), ResourceStatus: types.ResourceStatusCreateComplete},
			}},
		},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 resources from 3 pages, got %d", len(got))
	}
	want := []CfnResource{
		{StackName: "stack-page1", Type: "AWS::S3::Bucket", ResourceID: "p1", LogicalID: "R1", Status: string(types.ResourceStatusCreateComplete)},
		{StackName: "stack-page2", Type: "AWS::Lambda::Function", ResourceID: "p2", LogicalID: "R2", Status: string(types.ResourceStatusCreateComplete)},
		{StackName: "stack-page3", Type: "AWS::IAM::Role", ResourceID: "p3", LogicalID: "R3", Status: string(types.ResourceStatusCreateComplete)},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("resource %d = %#v, want %#v", i, got[i], want[i])
		}
	}
	if mock.describeStackResourcesCalls != 3 {
		t.Errorf("expected 3 DescribeStackResources calls (one per stack), got %d", mock.describeStackResourcesCalls)
	}
}

// TestGetResourcesSkipsStacksWithoutNameDuringWildcardFiltering verifies that
// malformed DescribeStacks entries with nil StackName do not panic wildcard
// filtering and are ignored.
func TestGetResourcesSkipsStacksWithoutNameDuringWildcardFiltering(t *testing.T) {
	stackName := "stack-*"
	mock := &paginatingMockClient{
		pages: map[string]cloudformation.DescribeStacksOutput{
			"": {
				Stacks:    []types.Stack{{StackName: aws.String("stack-page1")}},
				NextToken: aws.String("token2"),
			},
			"token2": {
				Stacks: []types.Stack{{}},
			},
		},
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{
			{StackResources: []types.StackResource{
				{LogicalResourceId: aws.String("R1"), PhysicalResourceId: aws.String("p1"), ResourceType: aws.String("AWS::S3::Bucket"), ResourceStatus: types.ResourceStatusCreateComplete},
			}},
		},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "stack-page1", got[0].StackName)
	assert.Equal(t, 1, mock.describeStackResourcesCalls)
}

// TestGetResourcesSkipsStacksWithoutNameWhenListingAllStacks verifies that
// malformed DescribeStacks entries with nil StackName do not panic when listing
// all stacks and are ignored.
func TestGetResourcesSkipsStacksWithoutNameWhenListingAllStacks(t *testing.T) {
	stackName := ""
	mock := &paginatingMockClient{
		pages: map[string]cloudformation.DescribeStacksOutput{
			"": {
				Stacks:    []types.Stack{{StackName: aws.String("stack-page1")}},
				NextToken: aws.String("token2"),
			},
			"token2": {
				Stacks: []types.Stack{{}},
			},
		},
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{
			{StackResources: []types.StackResource{
				{LogicalResourceId: aws.String("R1"), PhysicalResourceId: aws.String("p1"), ResourceType: aws.String("AWS::S3::Bucket"), ResourceStatus: types.ResourceStatusCreateComplete},
			}},
		},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "stack-page1", got[0].StackName)
	assert.Equal(t, 1, mock.describeStackResourcesCalls)
}

// TestGetResourcesSkipsMissingPhysicalResourceID verifies that resources without a usable physical ID are ignored.
func TestGetResourcesSkipsMissingPhysicalResourceID(t *testing.T) {
	stackName := "nil-physical-id-stack"
	stacksOut := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{{StackName: aws.String(stackName)}},
	}
	resOut := cloudformation.DescribeStackResourcesOutput{
		StackResources: []types.StackResource{
			{
				LogicalResourceId:  aws.String("PendingResource"),
				PhysicalResourceId: nil,
				ResourceType:       aws.String("AWS::EC2::NatGateway"),
				ResourceStatus:     types.ResourceStatusCreateInProgress,
			},
			{
				LogicalResourceId:  aws.String("ReadyResource"),
				PhysicalResourceId: aws.String("nat-123"),
				ResourceType:       aws.String("AWS::EC2::NatGateway"),
				ResourceStatus:     types.ResourceStatusCreateComplete,
			},
			{
				LogicalResourceId:  aws.String("EmptyResource"),
				PhysicalResourceId: aws.String(""),
				ResourceType:       aws.String("AWS::EC2::NatGateway"),
				ResourceStatus:     types.ResourceStatusCreateComplete,
			},
		},
	}
	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{resOut},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resource after skipping missing physical IDs, got %d", len(got))
	}
	if got[0].ResourceID != "nat-123" {
		t.Fatalf("expected remaining resource ID nat-123, got %q", got[0].ResourceID)
	}
}

// TestGetResourcesPaginationError verifies that DescribeStacks pagination errors
// are returned to the caller instead of exiting the process.
func TestGetResourcesPaginationError(t *testing.T) {
	stackName := "err-stack"
	opErr := &smithy.OperationError{
		ServiceID:     "CloudFormation",
		OperationName: "DescribeStacks",
		Err:           fmt.Errorf("connection reset"),
	}
	mock := &mockCloudFormationClient{
		describeStacksErr: opErr,
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err == nil {
		t.Fatal("expected error from GetResources when DescribeStacks fails, got nil")
	}
	if got != nil {
		t.Errorf("expected nil resources on error, got %v", got)
	}
	if !strings.Contains(err.Error(), "connection reset") {
		t.Errorf("expected error to contain original message, got: %v", err)
	}
}

// TestGetResourcesNonThrottlingAPIError verifies that non-throttling API errors
// from DescribeStackResources are returned to the caller instead of exiting the process.
func TestGetResourcesNonThrottlingAPIError(t *testing.T) {
	stackName := "err-stack"
	stacksOut := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{{StackName: aws.String(stackName)}},
	}
	apiErr := mockAPIError{code: "ValidationError", message: "bad", fault: smithy.FaultClient}
	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{{}},
		describeStackResourcesErrs:    []error{apiErr},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err == nil {
		t.Fatal("expected error from GetResources on non-throttling API error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil resources on error, got %v", got)
	}
	if !strings.Contains(err.Error(), "ValidationError") {
		t.Errorf("expected error to contain 'ValidationError', got: %v", err)
	}
	if !strings.Contains(err.Error(), stackName) {
		t.Errorf("expected error to contain stack name %q, got: %v", stackName, err)
	}
	// Verify the original API error is wrapped and accessible via errors.As
	var unwrapped smithy.APIError
	if !errors.As(err, &unwrapped) {
		t.Errorf("expected error to wrap smithy.APIError, but errors.As failed")
	}
}

// TestGetResourcesThrottlingRetryExhausted verifies that when throttling retry
// also fails, the error is returned instead of exiting the process.
func TestGetResourcesThrottlingRetryExhausted(t *testing.T) {
	stackName := "throttle-fail-stack"
	stacksOut := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{{StackName: aws.String(stackName)}},
	}
	throttleErr := mockAPIError{code: "Throttling", message: "Rate exceeded", fault: smithy.FaultServer}

	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{{}, {}},
		describeStackResourcesErrs:    []error{throttleErr, fmt.Errorf("still throttled")},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err == nil {
		t.Fatal("expected error from GetResources when throttling retry fails, got nil")
	}
	if got != nil {
		t.Errorf("expected nil resources on error, got %v", got)
	}
}

// TestGetResourcesGenericDescribeStackResourcesError verifies that non-API errors
// from DescribeStackResources are returned to the caller instead of exiting the process.
func TestGetResourcesGenericDescribeStackResourcesError(t *testing.T) {
	stackName := "generic-err-stack"
	stacksOut := cloudformation.DescribeStacksOutput{
		Stacks: []types.Stack{{StackName: aws.String(stackName)}},
	}
	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{{}},
		describeStackResourcesErrs:    []error{fmt.Errorf("unexpected network error")},
	}

	got, err := GetResources(context.Background(), &stackName, mock)
	if err == nil {
		t.Fatal("expected error from GetResources on generic error, got nil")
	}
	if got != nil {
		t.Errorf("expected nil resources on error, got %v", got)
	}
	if !strings.Contains(err.Error(), "unexpected network error") {
		t.Errorf("expected error to contain original message, got: %v", err)
	}
	if !strings.Contains(err.Error(), stackName) {
		t.Errorf("expected error to contain stack name %q, got: %v", stackName, err)
	}
}
