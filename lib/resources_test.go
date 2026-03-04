package lib

// Tests in this file cover the behaviour of GetResources, including
// successful retrieval of resources, retry handling on throttling
// errors, and failure behaviour for other API errors.

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/smithy-go"
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

	got := GetResources(&stackName, mock)
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
	got := GetResources(&stackName, mock)
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

	got := GetResources(&stackName, mock)

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

// TestGetResourcesSkipsNilPhysicalResourceID verifies that resources without a physical ID are ignored.
func TestGetResourcesSkipsNilPhysicalResourceID(t *testing.T) {
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
		},
	}
	mock := &mockCloudFormationClient{
		describeStacksOutput:          stacksOut,
		describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{resOut},
	}

	got := GetResources(&stackName, mock)
	if len(got) != 1 {
		t.Fatalf("expected 1 resource after skipping nil physical IDs, got %d", len(got))
	}
	if got[0].ResourceID != "nat-123" {
		t.Fatalf("expected remaining resource ID nat-123, got %q", got[0].ResourceID)
	}
}

// TestGetResourcesNonThrottlingError verifies that non-throttling API errors cause the function to log and exit.
func TestGetResourcesNonThrottlingError(t *testing.T) {
	if os.Getenv("FOG_TEST_HELPER") == "1" {
		stackName := "err-stack"
		stacksOut := cloudformation.DescribeStacksOutput{Stacks: []types.Stack{{StackName: aws.String(stackName)}}}
		apiErr := mockAPIError{code: "ValidationError", message: "bad", fault: smithy.FaultClient}
		mock := &mockCloudFormationClient{
			describeStacksOutput:          stacksOut,
			describeStackResourcesOutputs: []cloudformation.DescribeStackResourcesOutput{{}},
			describeStackResourcesErrs:    []error{apiErr},
		}
		GetResources(&stackName, mock)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestGetResourcesNonThrottlingError")
	cmd.Env = append(os.Environ(), "FOG_TEST_HELPER=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %v", err)
	}
	if exitErr.ExitCode() == 0 {
		t.Errorf("expected non-zero exit code")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("ValidationError")) {
		t.Errorf("expected log output to contain error message; got %s", stderr.String())
	}
}
