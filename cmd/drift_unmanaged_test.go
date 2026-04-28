package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/spf13/viper"
)

// TestCheckIfResourcesAreManaged_CorrectlyIdentifiesManagedResources verifies that
// checkIfResourcesAreManaged correctly determines whether resources are managed by
// CloudFormation by matching physical IDs from allresources against physical IDs
// in the logicalToPhysical map.
//
// Bug T-455: The original implementation used stringValueInMap which scans map values
// via linear search. This should use a proper reverse lookup set built from the
// logicalToPhysical values for O(1) key-based lookups.
func TestCheckIfResourcesAreManaged_CorrectlyIdentifiesManagedResources(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because settings uses global state
	viper.Reset()

	// logicalToPhysical maps CloudFormation logical IDs to physical resource IDs
	logicalToPhysical := map[string]string{
		"MyPermissionSet": "arn:aws:sso:::permissionSet/ssoins-123/ps-abc",
		"MyBucket":        "my-bucket-xyz",
		"MyVPC":           "vpc-12345",
	}

	// allresources maps physical resource identifiers to resource types
	// These come from ListAllResources which queries actual AWS APIs
	allresources := map[string]string{
		"arn:aws:sso:::permissionSet/ssoins-123/ps-abc": "AWS::SSO::PermissionSet", // managed
		"arn:aws:sso:::permissionSet/ssoins-123/ps-def": "AWS::SSO::PermissionSet", // unmanaged
		"my-bucket-xyz": "AWS::S3::Bucket", // managed
	}

	var rows []map[string]any
	checkIfResourcesAreManaged(allresources, logicalToPhysical, &rows)

	// Only the unmanaged resource (ps-def) should appear in rows
	if len(rows) != 1 {
		t.Fatalf("expected 1 unmanaged resource, got %d: %v", len(rows), rows)
	}

	row := rows[0]
	if row["LogicalId"] != "arn:aws:sso:::permissionSet/ssoins-123/ps-def" {
		t.Errorf("expected unmanaged resource ID 'arn:aws:sso:::permissionSet/ssoins-123/ps-def', got %v", row["LogicalId"])
	}
	if row["ChangeType"] != "UNMANAGED" {
		t.Errorf("expected ChangeType 'UNMANAGED', got %v", row["ChangeType"])
	}
	if row["Type"] != "AWS::SSO::PermissionSet" {
		t.Errorf("expected Type 'AWS::SSO::PermissionSet', got %v", row["Type"])
	}
}

// TestCheckIfResourcesAreManaged_AllManaged verifies no rows are added when all
// resources are managed by CloudFormation.
func TestCheckIfResourcesAreManaged_AllManaged(t *testing.T) {
	viper.Reset()

	logicalToPhysical := map[string]string{
		"MyBucket": "my-bucket-xyz",
		"MyVPC":    "vpc-12345",
	}

	allresources := map[string]string{
		"my-bucket-xyz": "AWS::S3::Bucket",
		"vpc-12345":     "AWS::EC2::VPC",
	}

	var rows []map[string]any
	checkIfResourcesAreManaged(allresources, logicalToPhysical, &rows)

	if len(rows) != 0 {
		t.Fatalf("expected 0 unmanaged resources, got %d: %v", len(rows), rows)
	}
}

// TestCheckIfResourcesAreManaged_NoneManaged verifies all resources are reported
// as unmanaged when none match the CloudFormation stack.
func TestCheckIfResourcesAreManaged_NoneManaged(t *testing.T) {
	viper.Reset()

	logicalToPhysical := map[string]string{
		"MyBucket": "my-bucket-xyz",
	}

	allresources := map[string]string{
		"other-bucket-abc": "AWS::S3::Bucket",
		"vpc-99999":        "AWS::EC2::VPC",
	}

	var rows []map[string]any
	checkIfResourcesAreManaged(allresources, logicalToPhysical, &rows)

	if len(rows) != 2 {
		t.Fatalf("expected 2 unmanaged resources, got %d: %v", len(rows), rows)
	}
}

// TestCheckIfResourcesAreManaged_IgnoreList verifies that resources in the
// ignore list are not reported as unmanaged.
func TestCheckIfResourcesAreManaged_IgnoreList(t *testing.T) {
	viper.Reset()
	viper.Set("drift.ignore-unmanaged-resources", []string{"ignored-bucket"})

	logicalToPhysical := map[string]string{
		"MyBucket": "my-bucket-xyz",
	}

	allresources := map[string]string{
		"other-bucket":   "AWS::S3::Bucket", // unmanaged, not ignored
		"ignored-bucket": "AWS::S3::Bucket", // unmanaged, but ignored
	}

	var rows []map[string]any
	checkIfResourcesAreManaged(allresources, logicalToPhysical, &rows)

	if len(rows) != 1 {
		t.Fatalf("expected 1 unmanaged resource (ignored-bucket should be skipped), got %d: %v", len(rows), rows)
	}

	if rows[0]["LogicalId"] != "other-bucket" {
		t.Errorf("expected unmanaged resource 'other-bucket', got %v", rows[0]["LogicalId"])
	}

	// Clean up viper state
	viper.Reset()
}

// TestCheckIfResourcesAreManaged_EmptyInputs verifies correct behavior with empty maps.
func TestCheckIfResourcesAreManaged_EmptyInputs(t *testing.T) {
	viper.Reset()

	// Empty allresources should produce no rows
	var rows []map[string]any
	checkIfResourcesAreManaged(map[string]string{}, map[string]string{"A": "B"}, &rows)
	if len(rows) != 0 {
		t.Fatalf("expected 0 rows with empty allresources, got %d", len(rows))
	}

	// Empty logicalToPhysical means everything is unmanaged
	rows = nil
	allresources := map[string]string{"resource-1": "AWS::S3::Bucket"}
	checkIfResourcesAreManaged(allresources, map[string]string{}, &rows)
	if len(rows) != 1 {
		t.Fatalf("expected 1 unmanaged row with empty logicalToPhysical, got %d", len(rows))
	}
}

func TestDetectUnmanagedResourcesReturnsListAllResourcesError(t *testing.T) {
	expectedErr := errors.New("list all resources failed")

	originalListAllResources := listAllResourcesFunc
	listAllResourcesFunc = func(context.Context, string, lib.CloudControlListResourcesAPI, interface {
		lib.SSOAdminListInstancesAPI
		lib.SSOAdminListPermissionSetsAPI
		lib.SSOAdminListAccountAssignmentsAPI
	}, lib.OrganizationsListAccountsAPI) (map[string]string, error) {
		return nil, expectedErr
	}
	t.Cleanup(func() {
		listAllResourcesFunc = originalListAllResources
	})

	var rows []map[string]any
	err := detectUnmanagedResources(context.Background(), []string{"AWS::S3::Bucket"}, map[string]string{}, &rows, config.AWSConfig{})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected no rows when list fails, got %d", len(rows))
	}
}
