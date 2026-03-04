package cmd

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	types "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type mockSpecialCasesClient struct {
	describeStackResourcesOutput cloudformation.DescribeStackResourcesOutput
	listExportsOutput            cloudformation.ListExportsOutput
}

func (m *mockSpecialCasesClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	return &m.describeStackResourcesOutput, nil
}

func (m *mockSpecialCasesClient) ListExports(ctx context.Context, params *cloudformation.ListExportsInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListExportsOutput, error) {
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

	naclResources, routetableResources, tgwRouteTableResources, logicalToPhysical := separateSpecialCases(defaultDrift, &stackName, mock)

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
