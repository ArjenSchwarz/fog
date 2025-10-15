package lib

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

	result := types.TransitGatewayRoute{
		DestinationCidrBlock: destCidr,
		PrefixListId:         prefixList,
		Type:                 types.TransitGatewayRouteTypeStatic,
	}

	// Handle Blackhole property
	blackhole := false
	if prop["Blackhole"] != nil {
		if val, ok := prop["Blackhole"].(bool); ok {
			blackhole = val
		}
	}

	// Set state based on Blackhole property
	if blackhole {
		result.State = types.TransitGatewayRouteStateBlackhole
	} else {
		result.State = types.TransitGatewayRouteStateActive
		// Only set attachment if not blackhole
		attachmentId := extractStringProperty(prop, params, logicalToPhysical, "TransitGatewayAttachmentId")
		if attachmentId != nil && *attachmentId != "" {
			result.TransitGatewayAttachments = []types.TransitGatewayRouteAttachment{
				{TransitGatewayAttachmentId: attachmentId},
			}
		}
	}

	return result
}

// CompareTGWRoutes compares two Transit Gateway routes for equality
func CompareTGWRoutes(route1 types.TransitGatewayRoute, route2 types.TransitGatewayRoute, blackholeIgnore []string) bool {
	// Compare DestinationCidrBlock
	if !stringPointerValueMatch(route1.DestinationCidrBlock, route2.DestinationCidrBlock) {
		return false
	}

	// Compare PrefixListId
	if !stringPointerValueMatch(route1.PrefixListId, route2.PrefixListId) {
		return false
	}

	// Extract and compare attachment IDs from TransitGatewayAttachments[0]
	attachment1 := ""
	if len(route1.TransitGatewayAttachments) > 0 && route1.TransitGatewayAttachments[0].TransitGatewayAttachmentId != nil {
		attachment1 = *route1.TransitGatewayAttachments[0].TransitGatewayAttachmentId
	}
	attachment2 := ""
	if len(route2.TransitGatewayAttachments) > 0 && route2.TransitGatewayAttachments[0].TransitGatewayAttachmentId != nil {
		attachment2 = *route2.TransitGatewayAttachments[0].TransitGatewayAttachmentId
	}

	if attachment1 != attachment2 {
		return false
	}

	// Compare State fields with blackhole ignore list handling
	if string(route1.State) != string(route2.State) {
		// If route1 is blackhole and attachment is in ignore list, consider it a match
		if route1.State == types.TransitGatewayRouteStateBlackhole && attachment1 != "" && slices.Contains(blackholeIgnore, attachment1) {
			return true
		}
		return false
	}

	return true
}

// FilterTGWRoutesByLogicalId filters Transit Gateway routes from a template by logical route table ID
func FilterTGWRoutesByLogicalId(logicalId string, template CfnTemplateBody, params []cfntypes.Parameter, logicalToPhysical map[string]string) map[string]types.TransitGatewayRoute {
	result := make(map[string]types.TransitGatewayRoute)
	for _, resource := range template.Resources {
		if resource.Type == "AWS::EC2::TransitGatewayRoute" && template.ShouldHaveResource(resource) {
			rtid := strings.Replace(resource.Properties["TransitGatewayRouteTableId"].(string), "REF: ", "", 1)
			convresource := TGWRouteResourceToTGWRoute(resource, params, logicalToPhysical)
			if rtid == logicalId {
				result[GetTGWRouteDestination(convresource)] = convresource
			}
		}
	}
	return result
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
		refname := value["Ref"].(string)
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

	return &result
}
