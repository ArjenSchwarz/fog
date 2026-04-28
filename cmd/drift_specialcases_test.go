package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	types "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type mockSpecialCasesClient struct {
	describeStackResourcesOutput cloudformation.DescribeStackResourcesOutput
	describeStackResourcesErr    error
	listExportsOutput            cloudformation.ListExportsOutput
	// listExportsPages supports multi-page responses keyed by NextToken ("" = first page).
	listExportsPages map[string]cloudformation.ListExportsOutput
}

func (m *mockSpecialCasesClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	return &m.describeStackResourcesOutput, m.describeStackResourcesErr
}

func (m *mockSpecialCasesClient) ListExports(ctx context.Context, params *cloudformation.ListExportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListExportsOutput, error) {
	if m.listExportsPages != nil {
		token := ""
		if params.NextToken != nil {
			token = *params.NextToken
		}
		page, ok := m.listExportsPages[token]
		if !ok {
			return &cloudformation.ListExportsOutput{}, nil
		}
		return &page, nil
	}
	return &m.listExportsOutput, nil
}

func TestSeparateSpecialCasesSkipsNilPhysicalResourceID(t *testing.T) {
	stackName := "test-stack"
	defaultDrift := []types.StackResourceDrift{
		{
			ResourceType:       aws.String("AWS::EC2::NetworkAcl"),
			LogicalResourceId:  aws.String("NaclResource"),
			PhysicalResourceId: aws.String("nacl-123"),
		},
		{
			ResourceType:       aws.String("AWS::EC2::RouteTable"),
			LogicalResourceId:  aws.String("RouteTableResource"),
			PhysicalResourceId: nil,
		},
		{
			ResourceType:       aws.String("AWS::EC2::TransitGatewayRouteTable"),
			LogicalResourceId:  aws.String("TransitGatewayResource"),
			PhysicalResourceId: aws.String("tgw-rtb-123"),
		},
	}

	mock := &mockSpecialCasesClient{
		describeStackResourcesOutput: cloudformation.DescribeStackResourcesOutput{
			StackResources: []types.StackResource{
				{
					LogicalResourceId:  aws.String("SubnetResource"),
					PhysicalResourceId: aws.String("subnet-123"),
				},
				{
					LogicalResourceId:  aws.String("PendingResource"),
					PhysicalResourceId: nil,
				},
			},
		},
		listExportsOutput: cloudformation.ListExportsOutput{
			Exports: []types.Export{
				{Name: aws.String("ExportedValue"), Value: aws.String("export-123")},
			},
		},
	}

	naclResources, routetableResources, tgwRouteTableResources, logicalToPhysical, err := separateSpecialCases(context.Background(), defaultDrift, &stackName, mock)
	if err != nil {
		t.Fatalf("separateSpecialCases() returned unexpected error: %v", err)
	}

	if got := logicalToPhysical["SubnetResource"]; got != "subnet-123" {
		t.Fatalf("expected SubnetResource to map to subnet-123, got %q", got)
	}
	if _, exists := logicalToPhysical["PendingResource"]; exists {
		t.Fatalf("expected PendingResource with nil physical ID to be skipped")
	}
	if got := logicalToPhysical["ExportedValue"]; got != "export-123" {
		t.Fatalf("expected ExportedValue to map to export-123, got %q", got)
	}

	if got := naclResources["NaclResource"]; got != "nacl-123" {
		t.Fatalf("expected NaclResource to map to nacl-123, got %q", got)
	}
	if _, exists := routetableResources["RouteTableResource"]; exists {
		t.Fatalf("expected RouteTableResource with nil physical ID to be skipped")
	}
	if got := tgwRouteTableResources["TransitGatewayResource"]; got != "tgw-rtb-123" {
		t.Fatalf("expected TransitGatewayResource to map to tgw-rtb-123, got %q", got)
	}
}

// TestSeparateSpecialCasesPaginatesListExports verifies that exports across
// multiple ListExports pages are all collected into logicalToPhysical.
// Before the fix, only the first page was read and subsequent pages were dropped.
func TestSeparateSpecialCasesPaginatesListExports(t *testing.T) {
	stackName := "test-stack"

	mock := &mockSpecialCasesClient{
		describeStackResourcesOutput: cloudformation.DescribeStackResourcesOutput{
			StackResources: []types.StackResource{},
		},
		listExportsPages: map[string]cloudformation.ListExportsOutput{
			"": {
				Exports: []types.Export{
					{Name: aws.String("ExportPage1"), Value: aws.String("value-page1")},
				},
				NextToken: aws.String("token2"),
			},
			"token2": {
				Exports: []types.Export{
					{Name: aws.String("ExportPage2"), Value: aws.String("value-page2")},
				},
				NextToken: aws.String("token3"),
			},
			"token3": {
				Exports: []types.Export{
					{Name: aws.String("ExportPage3"), Value: aws.String("value-page3")},
				},
				// No NextToken — last page.
			},
		},
	}

	_, _, _, logicalToPhysical, err := separateSpecialCases(context.Background(), nil, &stackName, mock)
	if err != nil {
		t.Fatalf("separateSpecialCases() returned unexpected error: %v", err)
	}

	for _, tc := range []struct {
		key  string
		want string
	}{
		{"ExportPage1", "value-page1"},
		{"ExportPage2", "value-page2"},
		{"ExportPage3", "value-page3"},
	} {
		got, ok := logicalToPhysical[tc.key]
		if !ok {
			t.Errorf("expected %s to be present in logicalToPhysical (pagination missed it)", tc.key)
		} else if got != tc.want {
			t.Errorf("expected %s=%q, got %q", tc.key, tc.want, got)
		}
	}
}

func TestSeparateSpecialCasesReturnsDescribeStackResourcesError(t *testing.T) {
	stackName := "test-stack"
	expectedErr := errors.New("describe stack resources failed")

	mock := &mockSpecialCasesClient{
		describeStackResourcesErr: expectedErr,
	}

	_, _, _, _, err := separateSpecialCases(context.Background(), nil, &stackName, mock)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
