package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

// TestCheckIfResourcesAreManaged_ValueLookup verifies that checkIfResourcesAreManaged
// correctly identifies managed resources by checking allresources keys (physical IDs
// from AWS APIs) against logicalToPhysical map values (physical IDs from CloudFormation).
//
// The allresources map is populated by ListAllResources which uses the Cloud Control
// API — its keys are physical resource identifiers (ARNs, IDs, etc.), not logical IDs.
// The logicalToPhysical map has logical IDs as keys and physical IDs as values.
// Therefore the check must match allresources keys against logicalToPhysical values.
func TestCheckIfResourcesAreManaged_ValueLookup(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because settings uses viper global state

	tests := map[string]struct {
		allresources      map[string]string
		logicalToPhysical map[string]string
		ignoreResources   []string
		wantUnmanaged     int
		wantLogicalIds    []string
	}{
		"managed_resource_not_marked_unmanaged": {
			// The physical ID "bucket-abc-123" matches a value in logicalToPhysical,
			// so it should be recognised as managed.
			allresources: map[string]string{
				"bucket-abc-123": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"MyBucket": "bucket-abc-123",
			},
			wantUnmanaged:  0,
			wantLogicalIds: nil,
		},
		"unmanaged_resource_correctly_reported": {
			// "rogue-bucket-456" is not a value in logicalToPhysical,
			// so it should be reported as UNMANAGED.
			allresources: map[string]string{
				"rogue-bucket-456": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"MyBucket": "bucket-abc-123",
			},
			wantUnmanaged:  1,
			wantLogicalIds: []string{"rogue-bucket-456"},
		},
		"logical_id_match_does_not_count_as_managed": {
			// "MyBucket" matches a KEY in logicalToPhysical but not a VALUE.
			// The function should check values, so this should be UNMANAGED.
			allresources: map[string]string{
				"MyBucket": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"MyBucket": "bucket-abc-123",
			},
			wantUnmanaged:  1,
			wantLogicalIds: []string{"MyBucket"},
		},
		"mixed_managed_and_unmanaged": {
			allresources: map[string]string{
				"vpc-12345":    "AWS::EC2::VPC",
				"vpc-99999":    "AWS::EC2::VPC",
				"subnet-67890": "AWS::EC2::Subnet",
			},
			logicalToPhysical: map[string]string{
				"ManagedVPC":    "vpc-12345",
				"ManagedSubnet": "subnet-67890",
			},
			wantUnmanaged:  1,
			wantLogicalIds: []string{"vpc-99999"},
		},
		"ignored_unmanaged_resource_not_reported": {
			allresources: map[string]string{
				"ignored-bucket-789": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"OtherResource": "other-physical-id",
			},
			ignoreResources: []string{"ignored-bucket-789"},
			wantUnmanaged:   0,
			wantLogicalIds:  nil,
		},
		"empty_allresources": {
			allresources:      map[string]string{},
			logicalToPhysical: map[string]string{"MyBucket": "bucket-123"},
			wantUnmanaged:     0,
			wantLogicalIds:    nil,
		},
		"empty_logicalToPhysical_all_unmanaged": {
			allresources: map[string]string{
				"resource-1": "AWS::S3::Bucket",
				"vpc-99999":  "AWS::EC2::VPC",
			},
			logicalToPhysical: map[string]string{},
			wantUnmanaged:     2,
			wantLogicalIds:    []string{"resource-1", "vpc-99999"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset viper state
			viper.Reset()
			if len(tc.ignoreResources) > 0 {
				viper.Set("drift.ignore-unmanaged-resources", tc.ignoreResources)
			}

			var rows []map[string]any
			checkIfResourcesAreManaged(tc.allresources, tc.logicalToPhysical, &rows)

			if len(rows) != tc.wantUnmanaged {
				t.Errorf("got %d unmanaged resources, want %d", len(rows), tc.wantUnmanaged)
				for _, row := range rows {
					t.Logf("  unmanaged: LogicalId=%v, Type=%v", row["LogicalId"], row["Type"])
				}
			}

			// Verify the correct logical IDs are reported
			if tc.wantLogicalIds != nil {
				gotIds := make(map[string]bool)
				for _, row := range rows {
					gotIds[row["LogicalId"].(string)] = true
				}
				for _, wantId := range tc.wantLogicalIds {
					if !gotIds[wantId] {
						t.Errorf("expected LogicalId %q in unmanaged results, but not found", wantId)
					}
				}
			}

			// Verify all reported rows have UNMANAGED change type
			for _, row := range rows {
				if row["ChangeType"] != "UNMANAGED" {
					t.Errorf("expected ChangeType=UNMANAGED, got %v", row["ChangeType"])
				}
			}
		})
	}
}
