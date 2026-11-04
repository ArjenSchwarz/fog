package cmd

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	types "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

type mockSpecialCasesClient struct {
	listStackResourcesOutput cloudformation.ListStackResourcesOutput
	listStackResourcesErr    error
	// listStackResourcesPages supports multi-page responses keyed by NextToken ("" = first page).
	listStackResourcesPages map[string]cloudformation.ListStackResourcesOutput
	listExportsOutput       cloudformation.ListExportsOutput
	// listExportsPages supports multi-page responses keyed by NextToken ("" = first page).
	listExportsPages map[string]cloudformation.ListExportsOutput
}

func (m *mockSpecialCasesClient) ListStackResources(ctx context.Context, params *cloudformation.ListStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStackResourcesOutput, error) {
	if m.listStackResourcesPages != nil {
		page, ok := m.listStackResourcesPages[aws.ToString(params.NextToken)]
		if !ok {
			return nil, fmt.Errorf("unexpected NextToken %q", aws.ToString(params.NextToken))
		}
		return &page, nil
	}
	return &m.listStackResourcesOutput, m.listStackResourcesErr
}

// stackResourceSummaries returns count resources with logical IDs Resource<start>..
// and physical IDs phys-<n>, mimicking one page of a large stack.
func stackResourceSummaries(start, count int) []types.StackResourceSummary {
	summaries := make([]types.StackResourceSummary, 0, count)
	for i := start; i < start+count; i++ {
		summaries = append(summaries, types.StackResourceSummary{
			LogicalResourceId:  aws.String(fmt.Sprintf("Resource%d", i)),
			PhysicalResourceId: aws.String(fmt.Sprintf("phys-%d", i)),
			ResourceType:       aws.String("AWS::EC2::Route"),
		})
	}
	return summaries
}

// TestSeparateSpecialCasesPaginatesListStackResources verifies that resources
// beyond the first 100 in a stack are included in logicalToPhysical (T-1511).
// DescribeStackResources capped results at 100, so mappings for later resources
// (e.g. TGW attachments referenced by routes) were silently missing.
func TestSeparateSpecialCasesPaginatesListStackResources(t *testing.T) {
	stackName := "large-stack"
	mock := &mockSpecialCasesClient{
		listStackResourcesPages: map[string]cloudformation.ListStackResourcesOutput{
			"": {
				StackResourceSummaries: stackResourceSummaries(0, 100),
				NextToken:              aws.String("page2"),
			},
			"page2": {
				StackResourceSummaries: stackResourceSummaries(100, 50),
			},
		},
	}

	_, _, _, logicalToPhysical, err := separateSpecialCases(t.Context(), nil, &stackName, mock)
	if err != nil {
		t.Fatalf("separateSpecialCases() returned unexpected error: %v", err)
	}
	if got := len(logicalToPhysical); got != 150 {
		t.Fatalf("expected 150 logical-to-physical mappings, got %d", got)
	}
	for _, id := range []string{"Resource0", "Resource100", "Resource149"} {
		if _, ok := logicalToPhysical[id]; !ok {
			t.Errorf("expected %s to be present in logicalToPhysical", id)
		}
	}
	if got := logicalToPhysical["Resource120"]; got != "phys-120" {
		t.Errorf("expected Resource120 to map to phys-120, got %q", got)
	}
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
		listStackResourcesOutput: cloudformation.ListStackResourcesOutput{
			StackResourceSummaries: []types.StackResourceSummary{
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
		listStackResourcesOutput: cloudformation.ListStackResourcesOutput{
			StackResourceSummaries: []types.StackResourceSummary{},
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

func TestSeparateSpecialCasesReturnsListStackResourcesError(t *testing.T) {
	stackName := "test-stack"
	expectedErr := errors.New("describe stack resources failed")

	mock := &mockSpecialCasesClient{
		listStackResourcesErr: expectedErr,
	}

	_, _, _, _, err := separateSpecialCases(context.Background(), nil, &stackName, mock)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
