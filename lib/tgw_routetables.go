package lib

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

// GetTransitGatewayRouteTableRoutes returns all routes for a Transit Gateway route table
func GetTransitGatewayRouteTableRoutes(
	ctx context.Context,
	routeTableId string,
	svc EC2SearchTransitGatewayRoutesAPI,
) ([]types.TransitGatewayRoute, error) {
	// Add timeout to context for API call
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	input := ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableId),
		Filters: []types.Filter{
			{
				Name:   aws.String("state"),
				Values: []string{"active", "blackhole"},
			},
		},
	}

	result, err := svc.SearchTransitGatewayRoutes(ctx, &input)
	if err != nil {
		// Handle context timeout
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("API call timed out after 30 seconds: %w", err)
		}

		// Use type assertion for AWS API errors
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.ErrorCode() {
			case "InvalidRouteTableID.NotFound":
				return nil, fmt.Errorf("transit gateway route table %s not found: %w", routeTableId, err)
			case "UnauthorizedOperation":
				return nil, fmt.Errorf("insufficient IAM permissions to search transit gateway routes: %w", err)
			}
		}

		return nil, err
	}

	return result.Routes, nil
}

// GetTGWRouteDestination returns the destination identifier from a Transit Gateway route
func GetTGWRouteDestination(route types.TransitGatewayRoute) string {
	switch {
	case route.DestinationCidrBlock != nil:
		return *route.DestinationCidrBlock
	case route.PrefixListId != nil:
		return *route.PrefixListId
	default:
		return ""
	}
}

// GetTGWRouteTarget returns the target identifier from a Transit Gateway route
func GetTGWRouteTarget(route types.TransitGatewayRoute) string {
	if route.State == types.TransitGatewayRouteStateBlackhole {
		return "blackhole"
	}
	// Validate attachment array exists and is not empty
	if len(route.TransitGatewayAttachments) == 0 {
		return ""
	}
	// Use first attachment (ECMP limitation)
	if route.TransitGatewayAttachments[0].TransitGatewayAttachmentId != nil {
		return *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId
	}
	return ""
}

// TGWRouteResourceToTGWRoute converts a CloudFormation Transit Gateway route resource to a TransitGatewayRoute
func TGWRouteResourceToTGWRoute(resource CfnTemplateResource, params []cfntypes.Parameter, logicalToPhysical map[string]string) types.TransitGatewayRoute {
	prop := resource.Properties

	// Extract destination (CIDR block or prefix list)
	destCidr := extractStringProperty(prop, params, logicalToPhysical, "DestinationCidrBlock")
	prefixList := extractStringProperty(prop, params, logicalToPhysical, "DestinationPrefixListId")

	// Extract Blackhole property
	isBlackhole := false
	if blackholeProp, ok := prop["Blackhole"]; ok {
		if blackholeBool, ok := blackholeProp.(bool); ok {
			isBlackhole = blackholeBool
		}
	}

	// Build the route structure
	route := types.TransitGatewayRoute{
		DestinationCidrBlock: destCidr,
		PrefixListId:         prefixList,
		Type:                 types.TransitGatewayRouteTypeStatic,
	}

	// Set state and attachments based on blackhole property
	if isBlackhole {
		route.State = types.TransitGatewayRouteStateBlackhole
		// Blackhole routes have no attachments
	} else {
		route.State = types.TransitGatewayRouteStateActive
		// Extract attachment ID
		attachmentId := extractStringProperty(prop, params, logicalToPhysical, "TransitGatewayAttachmentId")
		if attachmentId != nil && *attachmentId != "" {
			route.TransitGatewayAttachments = []types.TransitGatewayRouteAttachment{
				{
					TransitGatewayAttachmentId: attachmentId,
				},
			}
		}
	}

	return route
}

// FilterTGWRoutesByLogicalId filters Transit Gateway routes from a template by logical route table ID
// Returns a map of destination -> TransitGatewayRoute for routes in the template
func FilterTGWRoutesByLogicalId(logicalId string, template CfnTemplateBody, params []cfntypes.Parameter, logicalToPhysical map[string]string) map[string]types.TransitGatewayRoute {
	result := make(map[string]types.TransitGatewayRoute)

	for _, resource := range template.Resources {
		if resource.Type == "AWS::EC2::TransitGatewayRoute" && template.ShouldHaveResource(resource) {
			rtid := strings.Replace(resource.Properties["TransitGatewayRouteTableId"].(string), "REF: ", "", 1)

			if rtid == logicalId {
				// Convert CloudFormation resource to TransitGatewayRoute
				route := TGWRouteResourceToTGWRoute(resource, params, logicalToPhysical)
				destination := GetTGWRouteDestination(route)
				// Store the route for comparison
				if destination != "" {
					result[destination] = route
				}
			}
		}
	}
	return result
}

// CompareTGWRoutes compares two Transit Gateway routes for equality
// Returns true if routes match, false if they differ
func CompareTGWRoutes(route1 types.TransitGatewayRoute, route2 types.TransitGatewayRoute, blackholeIgnore []string) bool {
	// Compare destination (CIDR or prefix list)
	if !stringPointerValueMatch(route1.DestinationCidrBlock, route2.DestinationCidrBlock) {
		return false
	}
	if !stringPointerValueMatch(route1.PrefixListId, route2.PrefixListId) {
		return false
	}

	// Compare state
	if route1.State != route2.State {
		// Check if this is a blackhole route that should be ignored
		dest := GetTGWRouteDestination(route1)
		for _, ignore := range blackholeIgnore {
			if dest == ignore && (route1.State == types.TransitGatewayRouteStateBlackhole || route2.State == types.TransitGatewayRouteStateBlackhole) {
				return true
			}
		}
		return false
	}

	// Compare attachment IDs (only for non-blackhole routes)
	if route1.State != types.TransitGatewayRouteStateBlackhole {
		target1 := GetTGWRouteTarget(route1)
		target2 := GetTGWRouteTarget(route2)
		if target1 != target2 {
			return false
		}
	}

	return true
}

// extractStringProperty extracts a string pointer from CloudFormation properties with parameter and logical ID resolution
func extractStringProperty(array map[string]any, params []cfntypes.Parameter, logicalToPhysical map[string]string, value string) *string {
	if _, ok := array[value]; !ok {
		return nil
	}
	result := ""
	switch value := array[value].(type) {
	case string:
		refvalue := strings.Replace(value, "REF: ", "", 1)
		if _, ok := logicalToPhysical[refvalue]; ok {
			result = logicalToPhysical[refvalue]
		} else {
			result = value
		}
	case map[string]any:
		// Handle Ref intrinsic function
		if refname, ok := value["Ref"].(string); ok {
			if _, ok := logicalToPhysical[refname]; ok {
				result = logicalToPhysical[refname]
			} else {
				for _, parameter := range params {
					if *parameter.ParameterKey == refname {
						if parameter.ResolvedValue != nil {
							result = *parameter.ResolvedValue
						} else {
							result = *parameter.ParameterValue
						}
					}
				}
			}
		}
		// Handle Fn::ImportValue intrinsic function
		// ImportValue returns the actual physical ID directly from the stack outputs
		if importValue, ok := value["Fn::ImportValue"].(string); ok {
			// The importValue is the export name, but we need the actual value
			// In the processed template, CloudFormation has already resolved this
			// So we look for it in logicalToPhysical map using the import name
			if physicalId, ok := logicalToPhysical[importValue]; ok {
				result = physicalId
			} else {
				// If not found, use the import value string itself as a fallback
				result = importValue
			}
		}
	}

	// Return nil if the result is empty string (property exists but couldn't be resolved)
	if result == "" {
		return nil
	}
	return &result
}
