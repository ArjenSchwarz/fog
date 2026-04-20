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

// ErrTGWRoutesTruncated is returned when SearchTransitGatewayRoutes reports
// AdditionalRoutesAvailable=true even after narrowing the query by route type.
// The EC2 API caps responses at 1000 routes and does not support a continuation
// token, so when both the static and propagated splits still overflow there is
// no further partitioning available and the result set cannot be retrieved in
// full.
var ErrTGWRoutesTruncated = errors.New("transit gateway route results truncated: more than 1000 routes for a single type filter")

// GetTransitGatewayRouteTableRoutes returns all routes for a Transit Gateway route table.
//
// SearchTransitGatewayRoutes caps its response at 1000 routes and indicates
// truncation via AdditionalRoutesAvailable rather than a continuation token.
// When the initial state-filtered call reports additional routes available, we
// narrow the query by route type (static and propagated) to retrieve the full
// result set as the union of both type-filtered calls. If either narrowed call
// still reports additional routes available, we return an error wrapping
// ErrTGWRoutesTruncated rather than silently returning partial data.
func GetTransitGatewayRouteTableRoutes(
	ctx context.Context,
	routeTableId string,
	svc EC2SearchTransitGatewayRoutesAPI,
) ([]types.TransitGatewayRoute, error) {
	// Add a timeout budget for the API call(s). A single query can take up to
	// 30s; the narrowing path may issue up to three sequential calls (one
	// initial, two narrowed by type), so the budget is scaled accordingly.
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	stateFilter := types.Filter{
		Name:   aws.String("state"),
		Values: []string{"active", "blackhole"},
	}

	result, err := searchTGWRoutes(ctx, svc, routeTableId, []types.Filter{stateFilter})
	if err != nil {
		return nil, err
	}

	if !additionalRoutesAvailable(result) {
		return result.Routes, nil
	}

	// The single-filter response was truncated. Split by route type to cover
	// the full result set. The union of "static" and "propagated" equals the
	// entire set of routes in the table. The initial (truncated) result is
	// discarded because the narrowed calls return complete, independent sets.
	var combined []types.TransitGatewayRoute
	for _, routeType := range []string{"static", "propagated"} {
		typeFilter := types.Filter{
			Name:   aws.String("type"),
			Values: []string{routeType},
		}
		narrowed, err := searchTGWRoutes(ctx, svc, routeTableId, []types.Filter{stateFilter, typeFilter})
		if err != nil {
			return nil, err
		}
		if additionalRoutesAvailable(narrowed) {
			return nil, fmt.Errorf("route table %s type=%s: %w", routeTableId, routeType, ErrTGWRoutesTruncated)
		}
		combined = append(combined, narrowed.Routes...)
	}
	return combined, nil
}

// searchTGWRoutes issues a single SearchTransitGatewayRoutes call and maps
// AWS errors to friendlier wrapped errors. It is factored out so the narrowing
// retry can reuse the same error-handling logic.
func searchTGWRoutes(
	ctx context.Context,
	svc EC2SearchTransitGatewayRoutesAPI,
	routeTableId string,
	filters []types.Filter,
) (*ec2.SearchTransitGatewayRoutesOutput, error) {
	input := ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: aws.String(routeTableId),
		Filters:                    filters,
	}

	result, err := svc.SearchTransitGatewayRoutes(ctx, &input)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("API call timed out: %w", err)
		}

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

	return result, nil
}

// additionalRoutesAvailable reports whether the API signalled that more routes
// than fit in the response were available for the query.
func additionalRoutesAvailable(result *ec2.SearchTransitGatewayRoutesOutput) bool {
	return result != nil && result.AdditionalRoutesAvailable != nil && *result.AdditionalRoutesAvailable
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

	// Look up the physical ID for the target logical ID
	physicalId := logicalToPhysical[logicalId]

	for _, resource := range template.Resources {
		if resource.Type == "AWS::EC2::TransitGatewayRoute" && template.ShouldHaveResource(resource) {
			if !tgwRouteMatchesRouteTable(resource, logicalId, physicalId, logicalToPhysical) {
				continue
			}
			// Convert CloudFormation resource to TransitGatewayRoute
			route := TGWRouteResourceToTGWRoute(resource, params, logicalToPhysical)
			destination := GetTGWRouteDestination(route)
			// Store the route for comparison
			if destination != "" {
				result[destination] = route
			}
		}
	}
	return result
}

// tgwRouteMatchesRouteTable checks whether a TGW route resource's TransitGatewayRouteTableId
// matches the given logical ID. Handles all property formats: "REF: " strings, Ref maps,
// Fn::ImportValue maps, and plain physical ID strings.
//
// The logicalToPhysical map serves a dual purpose: it maps both CloudFormation logical IDs
// to their physical resource IDs, and stack export names to their exported values. This
// allows Fn::ImportValue resolution by looking up the export name in the same map.
func tgwRouteMatchesRouteTable(resource CfnTemplateResource, logicalId string, physicalId string, logicalToPhysical map[string]string) bool {
	prop := resource.Properties["TransitGatewayRouteTableId"]
	if prop == nil {
		return false
	}

	switch value := prop.(type) {
	case string:
		// Handle "REF: LogicalId" format
		rtid := strings.TrimPrefix(value, "REF: ")
		if rtid == logicalId {
			return true
		}
		// Handle plain physical ID string
		if physicalId != "" && rtid == physicalId {
			return true
		}
	case map[string]any:
		// Handle {"Ref": "LogicalId"} format
		if refName, ok := value["Ref"].(string); ok {
			if refName == logicalId {
				return true
			}
		}
		// Handle {"Fn::ImportValue": "ExportName"} format.
		// When physicalId is empty (the target logical ID has no entry in logicalToPhysical),
		// ImportValue routes cannot be matched and are silently excluded. This is acceptable
		// for drift detection because without a known physical ID there is nothing to compare against.
		if importName, ok := value["Fn::ImportValue"].(string); ok {
			if resolvedId, ok := logicalToPhysical[importName]; ok {
				if physicalId != "" && resolvedId == physicalId {
					return true
				}
			}
		}
	}
	return false
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
		refvalue := strings.TrimPrefix(value, "REF: ")
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
					if parameter.ParameterKey == nil {
						continue
					}
					if *parameter.ParameterKey == refname {
						if parameter.ResolvedValue != nil {
							result = *parameter.ResolvedValue
						} else if parameter.ParameterValue != nil {
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
