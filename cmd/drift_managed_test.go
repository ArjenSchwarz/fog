package cmd

import (
	"testing"

	"github.com/spf13/viper"
)

// TestCheckIfResourcesAreManaged_KeyLookup verifies that checkIfResourcesAreManaged
// correctly identifies managed resources by checking logicalToPhysical map keys
// (logical IDs), not map values (physical IDs).
//
// Bug T-435: The original implementation used stringValueInMap which checks map
// values (physical IDs). This means a resource is only found if its identifier
// happens to match a physical ID, not a logical ID. The function should check
// map keys instead, since allresources keys should be compared against logical
// resource IDs in the logicalToPhysical map.
func TestCheckIfResourcesAreManaged_KeyLookup(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because settings uses viper global state

	tests := map[string]struct {
		allresources      map[string]string
		logicalToPhysical map[string]string
		ignoreResources   []string
		wantUnmanaged     int
		wantLogicalIds    []string
	}{
		"managed_resource_not_marked_unmanaged": {
			// The resource key "MyBucket" matches a key in logicalToPhysical.
			// Before the fix, stringValueInMap checks values ("bucket-abc-123"),
			// so "MyBucket" would NOT be found, falsely marking it UNMANAGED.
			allresources: map[string]string{
				"MyBucket": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"MyBucket": "bucket-abc-123",
			},
			wantUnmanaged:  0,
			wantLogicalIds: nil,
		},
		"unmanaged_resource_correctly_reported": {
			// "RogueResource" is not a key in logicalToPhysical,
			// so it should be reported as UNMANAGED.
			allresources: map[string]string{
				"RogueResource": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"MyBucket": "bucket-abc-123",
			},
			wantUnmanaged:  1,
			wantLogicalIds: []string{"RogueResource"},
		},
		"physical_id_match_does_not_count_as_managed": {
			// "bucket-abc-123" matches a VALUE in logicalToPhysical but not a KEY.
			// The old code (stringValueInMap) would wrongly consider this managed.
			// The fix should report it as UNMANAGED.
			allresources: map[string]string{
				"bucket-abc-123": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"MyBucket": "bucket-abc-123",
			},
			wantUnmanaged:  1,
			wantLogicalIds: []string{"bucket-abc-123"},
		},
		"mixed_managed_and_unmanaged": {
			allresources: map[string]string{
				"ManagedVPC":    "AWS::EC2::VPC",
				"UnmanagedVPC":  "AWS::EC2::VPC",
				"ManagedSubnet": "AWS::EC2::Subnet",
			},
			logicalToPhysical: map[string]string{
				"ManagedVPC":    "vpc-12345",
				"ManagedSubnet": "subnet-67890",
			},
			wantUnmanaged:  1,
			wantLogicalIds: []string{"UnmanagedVPC"},
		},
		"ignored_unmanaged_resource_not_reported": {
			allresources: map[string]string{
				"IgnoredResource": "AWS::S3::Bucket",
			},
			logicalToPhysical: map[string]string{
				"OtherResource": "other-physical-id",
			},
			ignoreResources: []string{"IgnoredResource"},
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
				"Resource1": "AWS::S3::Bucket",
				"Resource2": "AWS::EC2::VPC",
			},
			logicalToPhysical: map[string]string{},
			wantUnmanaged:     2,
			wantLogicalIds:    []string{"Resource1", "Resource2"},
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
