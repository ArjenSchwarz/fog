package lib

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/aws/aws-sdk-go-v2/service/ssoadmin"
)

const (
	resourceTypeSSOAssignment = "AWS::SSO::Assignment"
)

// GetPermissionSetArns retrieves all SSO permission set ARNs for the organization and returns them as a map.
func GetPermissionSetArns(ssoClient interface {
	SSOAdminListInstancesAPI
	SSOAdminListPermissionSetsAPI
}) (map[string]string, error) {
	// Get the SSO instance ARN
	ssoInstanceArn, err := GetSSOInstanceArn(ssoClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO instance ARN: %w", err)
	}
	input := &ssoadmin.ListPermissionSetsInput{
		InstanceArn: &ssoInstanceArn,
	}

	result, err := ssoClient.ListPermissionSets(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to list permission sets: %w", err)
	}

	permissionSetArns := map[string]string{}
	for _, permissionSetArn := range result.PermissionSets {
		permissionSetArns[fmt.Sprintf("%s|%s", ssoInstanceArn, permissionSetArn)] = "AWS::SSO::PermissionSet"
	}

	return permissionSetArns, nil
}

// GetSSOInstanceArn retrieves the ARN of the first SSO instance in the organization.
func GetSSOInstanceArn(ssoClient SSOAdminListInstancesAPI) (string, error) {
	input := &ssoadmin.ListInstancesInput{}

	result, err := ssoClient.ListInstances(context.TODO(), input)
	if err != nil {
		return "", fmt.Errorf("failed to list SSO instances: %w", err)
	}

	if len(result.Instances) == 0 {
		return "", fmt.Errorf("no SSO instances found")
	}

	return *result.Instances[0].InstanceArn, nil
}

// GetAssignmentArns retrieves all SSO account assignment ARNs across all accounts and permission sets.
func GetAssignmentArns(ssoClient interface {
	SSOAdminListInstancesAPI
	SSOAdminListPermissionSetsAPI
	SSOAdminListAccountAssignmentsAPI
}, organizationsClient OrganizationsListAccountsAPI) (map[string]string, error) {
	// Get the SSO instance ARN
	ssoInstanceArn, err := GetSSOInstanceArn(ssoClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO instance ARN: %w", err)
	}
	permissionSets, err := GetPermissionSetArns(ssoClient)
	if err != nil {
		return map[string]string{}, err
	}
	assignmentArns := map[string]string{}
	// for all permission sets loop over the accounts and get the assignments
	for permissionSet := range permissionSets {
		permissionSetArn := strings.Split(permissionSet, "|")[1]
		assignments, err := GetAccountAssignmentArnsForPermissionSet(ssoClient, organizationsClient, ssoInstanceArn, permissionSetArn)
		if err != nil {
			return map[string]string{}, err
		}
		for assignment := range assignments {
			assignmentArns[assignment] = resourceTypeSSOAssignment
		}
	}

	return assignmentArns, nil
}

// GetAccountAssignmentArnsForPermissionSet retrieves all account assignment ARNs for a specific permission set across all accounts.
func GetAccountAssignmentArnsForPermissionSet(ssoClient SSOAdminListAccountAssignmentsAPI, organizationsClient OrganizationsListAccountsAPI, ssoInstanceArn string, permissionSetArn string) (map[string]string, error) {
	// Get the list of accounts
	accounts, err := GetAccountIDs(organizationsClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	assignmentArns := map[string]string{}
	// for all accounts get the assignments
	for _, account := range accounts {
		input := &ssoadmin.ListAccountAssignmentsInput{
			AccountId:        &account,
			InstanceArn:      &ssoInstanceArn,
			PermissionSetArn: &permissionSetArn,
		}

		result, err := ssoClient.ListAccountAssignments(context.TODO(), input)
		if err != nil {
			return nil, fmt.Errorf("failed to list account assignments: %w", err)
		}

		for _, assignment := range result.AccountAssignments {
			assignmentArns[fmt.Sprintf("%s|%s|AWS_ACCOUNT|%s|%s|%s", ssoInstanceArn, *assignment.AccountId, permissionSetArn, assignment.PrincipalType, *assignment.PrincipalId)] = "AWS::SSO::Assignment"
		}
	}

	return assignmentArns, nil
}

// GetAccountIDs retrieves all AWS account IDs in the organization.
func GetAccountIDs(organizationsClient OrganizationsListAccountsAPI) ([]string, error) {
	input := &organizations.ListAccountsInput{}

	var accounts []string
	paginator := organizations.NewListAccountsPaginator(organizationsClient, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, account := range output.Accounts {
			accounts = append(accounts, *account.Id)
		}
	}

	return accounts, nil
}
