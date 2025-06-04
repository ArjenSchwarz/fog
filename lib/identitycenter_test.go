package lib

// This file contains unit tests for the Identity Center helper functions.
// Each test uses stubbed AWS SDK clients to exercise success and failure
// scenarios without making real API calls.

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	orgtypes "github.com/aws/aws-sdk-go-v2/service/organizations/types"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/ssoadmin/types"
)

type mockSSOAdminClient struct {
	listInstancesOutput           *ssoadmin.ListInstancesOutput
	listInstancesErr              error
	listPermissionSetsOutput      *ssoadmin.ListPermissionSetsOutput
	listPermissionSetsErr         error
	listAccountAssignmentsOutputs []*ssoadmin.ListAccountAssignmentsOutput
	listAccountAssignmentsErr     error
	assignmentCall                int
}

func (m *mockSSOAdminClient) ListInstances(ctx context.Context, params *ssoadmin.ListInstancesInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListInstancesOutput, error) {
	if m.listInstancesOutput == nil {
		m.listInstancesOutput = &ssoadmin.ListInstancesOutput{}
	}
	return m.listInstancesOutput, m.listInstancesErr
}

func (m *mockSSOAdminClient) ListPermissionSets(ctx context.Context, params *ssoadmin.ListPermissionSetsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListPermissionSetsOutput, error) {
	if m.listPermissionSetsOutput == nil {
		m.listPermissionSetsOutput = &ssoadmin.ListPermissionSetsOutput{}
	}
	return m.listPermissionSetsOutput, m.listPermissionSetsErr
}

func (m *mockSSOAdminClient) ListAccountAssignments(ctx context.Context, params *ssoadmin.ListAccountAssignmentsInput, optFns ...func(*ssoadmin.Options)) (*ssoadmin.ListAccountAssignmentsOutput, error) {
	if m.listAccountAssignmentsErr != nil {
		return nil, m.listAccountAssignmentsErr
	}
	if m.assignmentCall >= len(m.listAccountAssignmentsOutputs) {
		return &ssoadmin.ListAccountAssignmentsOutput{}, nil
	}
	out := m.listAccountAssignmentsOutputs[m.assignmentCall]
	m.assignmentCall++
	return out, nil
}

type mockOrganizationsClient struct {
	outputs []*organizations.ListAccountsOutput
	err     error
	call    int
}

func (m *mockOrganizationsClient) ListAccounts(ctx context.Context, params *organizations.ListAccountsInput, optFns ...func(*organizations.Options)) (*organizations.ListAccountsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.call >= len(m.outputs) {
		return m.outputs[len(m.outputs)-1], nil
	}
	out := m.outputs[m.call]
	m.call++
	return out, nil
}

// TestGetSSOInstanceArn ensures that GetSSOInstanceArn returns the first
// available SSO instance ARN and properly reports errors or missing instances.
func TestGetSSOInstanceArn(t *testing.T) {
	arn := "arn:aws:sso:::instance/ins1"
	tests := []struct {
		name    string
		client  *mockSSOAdminClient
		want    string
		wantErr bool
	}{
		{
			name:    "success",
			client:  &mockSSOAdminClient{listInstancesOutput: &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{{InstanceArn: aws.String(arn)}}}},
			want:    arn,
			wantErr: false,
		},
		{
			name:    "api error",
			client:  &mockSSOAdminClient{listInstancesErr: errors.New("error")},
			wantErr: true,
		},
		{
			name:    "no instances",
			client:  &mockSSOAdminClient{listInstancesOutput: &ssoadmin.ListInstancesOutput{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSSOInstanceArn(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetPermissionSetArns verifies that GetPermissionSetArns collects the
// permission set ARNs for the SSO instance and handles API failures.
func TestGetPermissionSetArns(t *testing.T) {
	instArn := "arn:aws:sso:::instance/ins1"
	ps1 := "ps1"
	ps2 := "ps2"
	tests := []struct {
		name    string
		client  *mockSSOAdminClient
		want    map[string]string
		wantErr bool
	}{
		{
			name: "success",
			client: &mockSSOAdminClient{
				listInstancesOutput:      &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{{InstanceArn: aws.String(instArn)}}},
				listPermissionSetsOutput: &ssoadmin.ListPermissionSetsOutput{PermissionSets: []string{ps1, ps2}},
			},
			want: map[string]string{
				fmt.Sprintf("%s|%s", instArn, ps1): "AWS::SSO::PermissionSet",
				fmt.Sprintf("%s|%s", instArn, ps2): "AWS::SSO::PermissionSet",
			},
			wantErr: false,
		},
		{
			name:    "instance error",
			client:  &mockSSOAdminClient{listInstancesErr: errors.New("boom")},
			wantErr: true,
		},
		{
			name: "permission set error",
			client: &mockSSOAdminClient{
				listInstancesOutput:   &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{{InstanceArn: aws.String(instArn)}}},
				listPermissionSetsErr: errors.New("boom"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPermissionSetArns(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetAccountIDs retrieves AWS account IDs using the mock Organizations
// client and asserts correct pagination and error handling.
func TestGetAccountIDs(t *testing.T) {
	account1 := "111111111111"
	account2 := "222222222222"
	tests := []struct {
		name    string
		client  *mockOrganizationsClient
		want    []string
		wantErr bool
	}{
		{
			name: "success",
			client: &mockOrganizationsClient{outputs: []*organizations.ListAccountsOutput{
				{Accounts: []orgtypes.Account{{Id: aws.String(account1)}}, NextToken: aws.String("token")},
				{Accounts: []orgtypes.Account{{Id: aws.String(account2)}}},
			}},
			want:    []string{account1, account2},
			wantErr: false,
		},
		{
			name:    "list error",
			client:  &mockOrganizationsClient{err: errors.New("boom")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAccountIDs(tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetAccountAssignmentArnsForPermissionSet checks that assignments for a
// specific permission set are gathered for all accounts and that errors are
// surfaced correctly.
func TestGetAccountAssignmentArnsForPermissionSet(t *testing.T) {
	instArn := "arn:aws:sso:::instance/ins1"
	psArn := "ps1"
	account := "111111111111"
	userID := "user1"
	tests := []struct {
		name    string
		sso     *mockSSOAdminClient
		orgs    *mockOrganizationsClient
		want    map[string]string
		wantErr bool
	}{
		{
			name: "success",
			sso: &mockSSOAdminClient{
				listAccountAssignmentsOutputs: []*ssoadmin.ListAccountAssignmentsOutput{
					{AccountAssignments: []ssotypes.AccountAssignment{{AccountId: aws.String(account), PermissionSetArn: aws.String(psArn), PrincipalId: aws.String(userID), PrincipalType: ssotypes.PrincipalTypeUser}}},
				},
			},
			orgs: &mockOrganizationsClient{outputs: []*organizations.ListAccountsOutput{{Accounts: []orgtypes.Account{{Id: aws.String(account)}}}}},
			want: map[string]string{
				fmt.Sprintf("%s|%s|AWS_ACCOUNT|%s|%s|%s", instArn, account, psArn, ssotypes.PrincipalTypeUser, userID): "AWS::SSO::Assignment",
			},
			wantErr: false,
		},
		{
			name:    "account error",
			sso:     &mockSSOAdminClient{},
			orgs:    &mockOrganizationsClient{err: errors.New("boom")},
			wantErr: true,
		},
		{
			name: "assignment error",
			sso: &mockSSOAdminClient{
				listAccountAssignmentsErr: errors.New("boom"),
			},
			orgs:    &mockOrganizationsClient{outputs: []*organizations.ListAccountsOutput{{Accounts: []orgtypes.Account{{Id: aws.String(account)}}}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAccountAssignmentArnsForPermissionSet(tt.sso, tt.orgs, instArn, psArn)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetAssignmentArns exercises the overall assignment retrieval logic
// combining permission sets and account assignments. It validates successful
// aggregation as well as failures in the individual API calls.
func TestGetAssignmentArns(t *testing.T) {
	instArn := "arn:aws:sso:::instance/ins1"
	ps1 := "ps1"
	ps2 := "ps2"
	account := "111111111111"
	userID := "user1"
	tests := []struct {
		name    string
		sso     *mockSSOAdminClient
		orgs    *mockOrganizationsClient
		want    map[string]string
		wantErr bool
	}{
		{
			name: "success",
			sso: &mockSSOAdminClient{
				listInstancesOutput:      &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{{InstanceArn: aws.String(instArn)}}},
				listPermissionSetsOutput: &ssoadmin.ListPermissionSetsOutput{PermissionSets: []string{ps1, ps2}},
				listAccountAssignmentsOutputs: []*ssoadmin.ListAccountAssignmentsOutput{
					{AccountAssignments: []ssotypes.AccountAssignment{{AccountId: aws.String(account), PermissionSetArn: aws.String(ps1), PrincipalId: aws.String(userID), PrincipalType: ssotypes.PrincipalTypeUser}}},
					{AccountAssignments: []ssotypes.AccountAssignment{{AccountId: aws.String(account), PermissionSetArn: aws.String(ps2), PrincipalId: aws.String(userID), PrincipalType: ssotypes.PrincipalTypeUser}}},
				},
			},
			orgs: &mockOrganizationsClient{outputs: []*organizations.ListAccountsOutput{{Accounts: []orgtypes.Account{{Id: aws.String(account)}}}}},
			want: map[string]string{
				fmt.Sprintf("%s|%s|AWS_ACCOUNT|%s|%s|%s", instArn, account, ps1, ssotypes.PrincipalTypeUser, userID): "AWS::SSO::Assignment",
				fmt.Sprintf("%s|%s|AWS_ACCOUNT|%s|%s|%s", instArn, account, ps2, ssotypes.PrincipalTypeUser, userID): "AWS::SSO::Assignment",
			},
			wantErr: false,
		},
		{
			name:    "instance error",
			sso:     &mockSSOAdminClient{listInstancesErr: errors.New("boom")},
			orgs:    &mockOrganizationsClient{},
			wantErr: true,
		},
		{
			name: "permission set error",
			sso: &mockSSOAdminClient{
				listInstancesOutput:   &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{{InstanceArn: aws.String(instArn)}}},
				listPermissionSetsErr: errors.New("boom"),
			},
			orgs:    &mockOrganizationsClient{},
			wantErr: true,
		},
		{
			name: "assignment error",
			sso: &mockSSOAdminClient{
				listInstancesOutput:       &ssoadmin.ListInstancesOutput{Instances: []ssotypes.InstanceMetadata{{InstanceArn: aws.String(instArn)}}},
				listPermissionSetsOutput:  &ssoadmin.ListPermissionSetsOutput{PermissionSets: []string{ps1}},
				listAccountAssignmentsErr: errors.New("boom"),
			},
			orgs:    &mockOrganizationsClient{outputs: []*organizations.ListAccountsOutput{{Accounts: []orgtypes.Account{{Id: aws.String(account)}}}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAssignmentArns(tt.sso, tt.orgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}
