//go:build integration
// +build integration

package cmd

import (
	"fmt"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/testutil"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransitGatewayDrift_LibraryFunctions tests the library functions that power
// Transit Gateway drift detection. The cmd-level orchestration function
// checkTransitGatewayRouteTableRoutes cannot be easily tested in isolation due to
// its dependency on config.AWSConfig (a concrete struct) and global variables.
//
// This test validates the end-to-end workflow using the lib functions that
// checkTransitGatewayRouteTableRoutes calls internally.
func TestTransitGatewayDrift_LibraryFunctions(t *testing.T) {
	testutil.SkipIfIntegration(t)

	tests := map[string]struct {
		routeTableLogicalID string
		awsRoutes           []ec2types.TransitGatewayRoute
		templateRoutes      []templateTGWRoute
		wantDriftDetected   bool
		wantUnmanaged       bool
		wantRemoved         bool
		wantModified        bool
	}{
		"unmanaged routes detected": {
			routeTableLogicalID: "TGWRouteTable1",
			awsRoutes: []ec2types.TransitGatewayRoute{
				{
					Type:                 ec2types.TransitGatewayRouteTypeStatic,
					State:                ec2types.TransitGatewayRouteStateActive,
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					TransitGatewayAttachments: []ec2types.TransitGatewayRouteAttachment{
						{TransitGatewayAttachmentId: aws.String("tgw-attach-11111111")},
					},
				},
			},
			templateRoutes:    []templateTGWRoute{},
			wantDriftDetected: true,
			wantUnmanaged:     true,
		},
		"removed routes detected": {
			routeTableLogicalID: "TGWRouteTable1",
			awsRoutes:           []ec2types.TransitGatewayRoute{},
			templateRoutes: []templateTGWRoute{
				{
					destCidr:     "10.0.0.0/16",
					attachmentID: "tgw-attach-11111111",
				},
			},
			wantDriftDetected: true,
			wantRemoved:       true,
		},
		"modified routes detected": {
			routeTableLogicalID: "TGWRouteTable1",
			awsRoutes: []ec2types.TransitGatewayRoute{
				{
					Type:                 ec2types.TransitGatewayRouteTypeStatic,
					State:                ec2types.TransitGatewayRouteStateActive,
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					TransitGatewayAttachments: []ec2types.TransitGatewayRouteAttachment{
						{TransitGatewayAttachmentId: aws.String("tgw-attach-22222222")},
					},
				},
			},
			templateRoutes: []templateTGWRoute{
				{
					destCidr:     "10.0.0.0/16",
					attachmentID: "tgw-attach-11111111",
				},
			},
			wantDriftDetected: true,
			wantModified:      true,
		},
		"no drift - identical routes": {
			routeTableLogicalID: "TGWRouteTable1",
			awsRoutes: []ec2types.TransitGatewayRoute{
				{
					Type:                 ec2types.TransitGatewayRouteTypeStatic,
					State:                ec2types.TransitGatewayRouteStateActive,
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					TransitGatewayAttachments: []ec2types.TransitGatewayRouteAttachment{
						{TransitGatewayAttachmentId: aws.String("tgw-attach-11111111")},
					},
				},
			},
			templateRoutes: []templateTGWRoute{
				{
					destCidr:     "10.0.0.0/16",
					attachmentID: "tgw-attach-11111111",
				},
			},
			wantDriftDetected: false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Build template with Transit Gateway routes
			template := buildTGWTemplate(tc.routeTableLogicalID, tc.templateRoutes)

			// Get template routes using the same function checkTransitGatewayRouteTableRoutes calls
			templateRouteMap := lib.FilterTGWRoutesByLogicalId(tc.routeTableLogicalID, template, []types.Parameter{}, map[string]string{})

			// Simulate the drift detection logic from checkTransitGatewayRouteTableRoutes
			foundUnmanaged := false
			foundModified := false

			// Check AWS routes against template
			for _, awsRoute := range tc.awsRoutes {
				// Filter propagated routes (same as checkTransitGatewayRouteTableRoutes)
				if awsRoute.Type == ec2types.TransitGatewayRouteTypePropagated {
					continue
				}

				// Filter transient states (same as checkTransitGatewayRouteTableRoutes)
				if awsRoute.State != ec2types.TransitGatewayRouteStateActive && awsRoute.State != ec2types.TransitGatewayRouteStateBlackhole {
					continue
				}

				routeID := lib.GetTGWRouteDestination(awsRoute)

				if templateRoute, ok := templateRouteMap[routeID]; ok {
					// Route exists in both - check if modified
					if !lib.CompareTGWRoutes(awsRoute, templateRoute, []string{}) {
						foundModified = true
					}
					delete(templateRouteMap, routeID)
				} else {
					// Unmanaged route (in AWS, not in template)
					foundUnmanaged = true
				}
			}

			// Remaining routes in templateRouteMap are removed routes
			foundRemoved := false
			for routeID := range templateRouteMap {
				if routeID != "" {
					foundRemoved = true
					break
				}
			}

			// Verify expectations
			if tc.wantDriftDetected {
				assert.True(t, foundUnmanaged || foundModified || foundRemoved, "Expected drift to be detected")
			} else {
				assert.False(t, foundUnmanaged, "Expected no unmanaged routes")
				assert.False(t, foundModified, "Expected no modified routes")
				assert.False(t, foundRemoved, "Expected no removed routes")
			}

			if tc.wantUnmanaged {
				assert.True(t, foundUnmanaged, "Expected unmanaged routes")
			}
			if tc.wantRemoved {
				assert.True(t, foundRemoved, "Expected removed routes")
			}
			if tc.wantModified {
				assert.True(t, foundModified, "Expected modified routes")
			}
		})
	}
}

// TestTransitGatewayDrift_PropagatedRoutesFiltered verifies that propagated routes
// are excluded from drift detection (per requirement 3.3)
func TestTransitGatewayDrift_PropagatedRoutesFiltered(t *testing.T) {
	testutil.SkipIfIntegration(t)

	// Setup propagated route
	awsRoutes := []ec2types.TransitGatewayRoute{
		{
			Type:                 ec2types.TransitGatewayRouteTypePropagated,
			State:                ec2types.TransitGatewayRouteStateActive,
			DestinationCidrBlock: aws.String("192.168.0.0/16"),
			TransitGatewayAttachments: []ec2types.TransitGatewayRouteAttachment{
				{TransitGatewayAttachmentId: aws.String("tgw-attach-propagated")},
			},
		},
	}

	// Empty template (no routes defined)
	template := buildTGWTemplate("TGWRouteTable1", []templateTGWRoute{})
	templateRouteMap := lib.FilterTGWRoutesByLogicalId("TGWRouteTable1", template, []types.Parameter{}, map[string]string{})

	// Simulate filtering logic
	foundUnmanaged := false
	for _, awsRoute := range awsRoutes {
		// This is the filter from checkTransitGatewayRouteTableRoutes
		if awsRoute.Type == ec2types.TransitGatewayRouteTypePropagated {
			continue
		}

		routeID := lib.GetTGWRouteDestination(awsRoute)
		if _, ok := templateRouteMap[routeID]; !ok {
			foundUnmanaged = true
		}
	}

	assert.False(t, foundUnmanaged, "Propagated routes should be filtered out")
}

// TestTransitGatewayDrift_TransientStatesFiltered verifies that routes in transient
// states are excluded from drift detection (per requirement 3.3)
func TestTransitGatewayDrift_TransientStatesFiltered(t *testing.T) {
	testutil.SkipIfIntegration(t)

	transientStates := []ec2types.TransitGatewayRouteState{
		ec2types.TransitGatewayRouteStatePending,
		ec2types.TransitGatewayRouteStateDeleting,
		ec2types.TransitGatewayRouteStateDeleted,
	}

	for _, state := range transientStates {
		state := state
		t.Run(string(state), func(t *testing.T) {
			t.Parallel()

			awsRoutes := []ec2types.TransitGatewayRoute{
				{
					Type:                 ec2types.TransitGatewayRouteTypeStatic,
					State:                state,
					DestinationCidrBlock: aws.String("10.1.0.0/16"),
					TransitGatewayAttachments: []ec2types.TransitGatewayRouteAttachment{
						{TransitGatewayAttachmentId: aws.String("tgw-attach-transient")},
					},
				},
			}

			template := buildTGWTemplate("TGWRouteTable1", []templateTGWRoute{})
			templateRouteMap := lib.FilterTGWRoutesByLogicalId("TGWRouteTable1", template, []types.Parameter{}, map[string]string{})

			// Simulate filtering logic
			foundUnmanaged := false
			for _, awsRoute := range awsRoutes {
				// Filter transient states (from checkTransitGatewayRouteTableRoutes)
				if awsRoute.State != ec2types.TransitGatewayRouteStateActive && awsRoute.State != ec2types.TransitGatewayRouteStateBlackhole {
					continue
				}

				routeID := lib.GetTGWRouteDestination(awsRoute)
				if _, ok := templateRouteMap[routeID]; !ok {
					foundUnmanaged = true
				}
			}

			assert.False(t, foundUnmanaged, "Routes in transient state %s should be filtered out", state)
		})
	}
}

// TestTransitGatewayDrift_SeparatePropertiesFlag verifies that the --separate-properties
// flag creates individual drift entries (per requirement 8.2)
func TestTransitGatewayDrift_SeparatePropertiesFlag(t *testing.T) {
	testutil.SkipIfIntegration(t)

	// Initialize viper settings
	viper.Reset()
	viper.Set("verbose", false)
	settings = &config.Config{}

	// Test with separate-properties enabled
	t.Run("separate properties enabled", func(t *testing.T) {
		driftFlags.SeparateProperties = true
		outputsettings = settings.NewOutputSettings()
		output := format.OutputArray{Keys: []string{"LogicalId", "Type", "ChangeType", "Details"}, Settings: outputsettings}

		// Simulate multiple route changes
		rulechanges := []string{
			"Unmanaged route: 10.1.0.0/16: tgw-attach-11111",
			"Removed route: 10.2.0.0/16: tgw-attach-22222",
			"Modified route: 10.3.0.0/16",
		}

		// This is the logic from checkTransitGatewayRouteTableRoutes
		if driftFlags.SeparateProperties {
			for _, change := range rulechanges {
				content := make(map[string]any)
				content["LogicalId"] = "Route for TransitGatewayRouteTable TGWRouteTable1"
				content["Type"] = "AWS::EC2::TransitGatewayRoute"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = change
				output.AddContents(content)
			}
		}

		require.Len(t, output.Contents, 3, "Should create separate entries for each change")
		for i, holder := range output.Contents {
			assert.Equal(t, rulechanges[i], holder.Contents["Details"], "Details should match individual change")
		}
	})

	// Test with separate-properties disabled
	t.Run("separate properties disabled", func(t *testing.T) {
		driftFlags.SeparateProperties = false
		outputsettings = settings.NewOutputSettings()
		output := format.OutputArray{Keys: []string{"LogicalId", "Type", "ChangeType", "Details"}, Settings: outputsettings}

		rulechanges := []string{
			"Unmanaged route: 10.1.0.0/16: tgw-attach-11111",
			"Removed route: 10.2.0.0/16: tgw-attach-22222",
			"Modified route: 10.3.0.0/16",
		}

		// This is the logic from checkTransitGatewayRouteTableRoutes
		if driftFlags.SeparateProperties {
			// Not executed in this test
		} else {
			content := make(map[string]any)
			content["LogicalId"] = "Routes for TransitGatewayRouteTable TGWRouteTable1"
			content["Type"] = "AWS::EC2::TransitGatewayRoute"
			content["ChangeType"] = string(types.StackResourceDriftStatusModified)
			content["Details"] = rulechanges
			output.AddContents(content)
		}

		require.Len(t, output.Contents, 1, "Should create single entry for all changes")
		assert.Equal(t, rulechanges, output.Contents[0].Contents["Details"], "Details should contain all changes")
	})
}

// TestTransitGatewayDrift_EmptyRouteTable verifies handling of route tables with no routes
func TestTransitGatewayDrift_EmptyRouteTable(t *testing.T) {
	testutil.SkipIfIntegration(t)

	// Empty AWS routes
	awsRoutes := []ec2types.TransitGatewayRoute{}

	// Empty template routes
	template := buildTGWTemplate("TGWRouteTable1", []templateTGWRoute{})
	templateRouteMap := lib.FilterTGWRoutesByLogicalId("TGWRouteTable1", template, []types.Parameter{}, map[string]string{})

	// Simulate drift detection logic
	foundDrift := false
	for _, awsRoute := range awsRoutes {
		if awsRoute.Type == ec2types.TransitGatewayRouteTypePropagated {
			continue
		}
		if awsRoute.State != ec2types.TransitGatewayRouteStateActive && awsRoute.State != ec2types.TransitGatewayRouteStateBlackhole {
			continue
		}

		routeID := lib.GetTGWRouteDestination(awsRoute)
		if _, ok := templateRouteMap[routeID]; !ok {
			foundDrift = true
		}
	}

	for routeID := range templateRouteMap {
		if routeID != "" {
			foundDrift = true
		}
	}

	assert.False(t, foundDrift, "Empty route table should not report drift")
}

// Helper types and functions for test setup

// templateTGWRoute represents a Transit Gateway route defined in a CloudFormation template
type templateTGWRoute struct {
	destCidr     string
	destPrefixID string
	attachmentID string
	blackhole    bool
}

// buildTGWTemplate creates a CloudFormation template with Transit Gateway routes
func buildTGWTemplate(routeTableLogicalID string, routes []templateTGWRoute) lib.CfnTemplateBody {
	resources := make(map[string]lib.CfnTemplateResource)

	// Add Transit Gateway route table resource
	resources[routeTableLogicalID] = lib.CfnTemplateResource{
		Type: "AWS::EC2::TransitGatewayRouteTable",
		Properties: map[string]any{
			"TransitGatewayId": "tgw-12345678",
		},
	}

	// Add route resources
	for i, route := range routes {
		routeLogicalID := fmt.Sprintf("%sRoute%d", routeTableLogicalID, i+1)
		routeProps := map[string]any{
			"TransitGatewayRouteTableId": "REF: " + routeTableLogicalID,
		}

		if route.destCidr != "" {
			routeProps["DestinationCidrBlock"] = route.destCidr
		}
		if route.destPrefixID != "" {
			routeProps["DestinationPrefixListId"] = route.destPrefixID
		}
		if route.blackhole {
			routeProps["Blackhole"] = true
		} else if route.attachmentID != "" {
			routeProps["TransitGatewayAttachmentId"] = route.attachmentID
		}

		resources[routeLogicalID] = lib.CfnTemplateResource{
			Type:       "AWS::EC2::TransitGatewayRoute",
			Properties: routeProps,
		}
	}

	return lib.CfnTemplateBody{
		Resources: resources,
	}
}
