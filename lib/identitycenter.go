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
func GetPermissionSetArns(ctx context.Context, ssoClient interface {
	SSOAdminListInstancesAPI
	SSOAdminListPermissionSetsAPI
}) (map[string]string, error) {
	// Get the SSO instance ARN
	ssoInstanceArn, err := GetSSOInstanceArn(ctx, ssoClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO instance ARN: %w", err)
	}
	input := &ssoadmin.ListPermissionSetsInput{
		InstanceArn: &ssoInstanceArn,
	}

	permissionSetArns := map[string]string{}
	paginator := ssoadmin.NewListPermissionSetsPaginator(ssoClient, input)

	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list permission sets: %w", err)
		}

		for _, permissionSetArn := range result.PermissionSets {
			permissionSetArns[fmt.Sprintf("%s|%s", ssoInstanceArn, permissionSetArn)] = "AWS::SSO::PermissionSet"
		}
	}

	return permissionSetArns, nil
}

// GetSSOInstanceArn retrieves the ARN of the first SSO instance in the organization.
func GetSSOInstanceArn(ctx context.Context, ssoClient SSOAdminListInstancesAPI) (string, error) {
	input := &ssoadmin.ListInstancesInput{}

	result, err := ssoClient.ListInstances(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to list SSO instances: %w", err)
	}

	if len(result.Instances) == 0 {
		return "", fmt.Errorf("no SSO instances found")
	}

	for _, instance := range result.Instances {
		if instance.InstanceArn != nil {
			return *instance.InstanceArn, nil
		}
	}

	return "", fmt.Errorf("no SSO instances with non-nil InstanceArn found")
}

// GetAssignmentArns retrieves all SSO account assignment ARNs across all accounts and permission sets.
func GetAssignmentArns(ctx context.Context, ssoClient interface {
	SSOAdminListInstancesAPI
	SSOAdminListPermissionSetsAPI
	SSOAdminListAccountAssignmentsAPI
}, organizationsClient OrganizationsListAccountsAPI) (map[string]string, error) {
	// Get the SSO instance ARN
	ssoInstanceArn, err := GetSSOInstanceArn(ctx, ssoClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO instance ARN: %w", err)
	}
	permissionSets, err := GetPermissionSetArns(ctx, ssoClient)
	if err != nil {
		return map[string]string{}, err
	}
	assignmentArns := map[string]string{}
	// for all permission sets loop over the accounts and get the assignments
	for permissionSet := range permissionSets {
		permissionSetArn := strings.Split(permissionSet, "|")[1]
		assignments, err := GetAccountAssignmentArnsForPermissionSet(ctx, ssoClient, organizationsClient, ssoInstanceArn, permissionSetArn)
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
func GetAccountAssignmentArnsForPermissionSet(ctx context.Context, ssoClient SSOAdminListAccountAssignmentsAPI, organizationsClient OrganizationsListAccountsAPI, ssoInstanceArn string, permissionSetArn string) (map[string]string, error) {
	// Get the list of accounts
	accounts, err := GetAccountIDs(ctx, organizationsClient)
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

		paginator := ssoadmin.NewListAccountAssignmentsPaginator(ssoClient, input)

		for paginator.HasMorePages() {
			result, err := paginator.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to list account assignments: %w", err)
			}

			for _, assignment := range result.AccountAssignments {
				if assignment.PrincipalId == nil {
					continue
				}

				// Use the known request account when the response omits AccountId
				accountID := assignment.AccountId
				if accountID == nil {
					accountID = &account
				}

				assignmentArns[fmt.Sprintf("%s|%s|AWS_ACCOUNT|%s|%s|%s", ssoInstanceArn, *accountID, permissionSetArn, assignment.PrincipalType, *assignment.PrincipalId)] = resourceTypeSSOAssignment
			}
		}
	}

	return assignmentArns, nil
}

// GetAccountIDs retrieves all AWS account IDs in the organization.
func GetAccountIDs(ctx context.Context, organizationsClient OrganizationsListAccountsAPI) ([]string, error) {
	input := &organizations.ListAccountsInput{}

	var accounts []string
	paginator := organizations.NewListAccountsPaginator(organizationsClient, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list accounts: %w", err)
		}

		for _, account := range output.Accounts {
			if account.Id == nil {
				continue
			}
			accounts = append(accounts, *account.Id)
		}
	}

	return accounts, nil
}
