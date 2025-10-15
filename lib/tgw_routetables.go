package lib

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// GetTransitGatewayRouteTableRoutes returns all routes for a Transit Gateway route table
func GetTransitGatewayRouteTableRoutes(
	ctx context.Context,
	routeTableId string,
	svc EC2SearchTransitGatewayRoutesAPI,
) ([]types.TransitGatewayRoute, error) {
	input := ec2.SearchTransitGatewayRoutesInput{
		TransitGatewayRouteTableId: &routeTableId,
	}
	result, err := svc.SearchTransitGatewayRoutes(ctx, &input)
	if err != nil {
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
