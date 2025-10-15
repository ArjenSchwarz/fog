package lib

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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
