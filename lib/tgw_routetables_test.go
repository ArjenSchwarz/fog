package lib

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type mockEC2SearchTransitGatewayRoutesAPI func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error)

func (m mockEC2SearchTransitGatewayRoutesAPI) SearchTransitGatewayRoutes(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
	return m(ctx, params, optFns...)
}

// TestGetTGWRouteDestination tests the GetTGWRouteDestination function which extracts the destination
// identifier from a Transit Gateway route. Tests cover CIDR block extraction, prefix list extraction,
// nil handling, and precedence when both CIDR and prefix list are present.
func TestGetTGWRouteDestination(t *testing.T) {
	type args struct {
		route types.TransitGatewayRoute
	}
	tests := map[string]struct {
		args args
		want string
	}{
		"Extract DestinationCidrBlock": {
			args: args{route: types.TransitGatewayRoute{
				DestinationCidrBlock: aws.String("10.0.0.0/16"),
			}},
			want: "10.0.0.0/16",
		},
		"Extract PrefixListId when CIDR is nil": {
			args: args{route: types.TransitGatewayRoute{
				PrefixListId: aws.String("pl-12345678"),
			}},
			want: "pl-12345678",
		},
		"Return empty string when both are nil": {
			args: args{route: types.TransitGatewayRoute{}},
			want: "",
		},
		"Prefer DestinationCidrBlock over PrefixListId": {
			args: args{route: types.TransitGatewayRoute{
				DestinationCidrBlock: aws.String("192.168.0.0/24"),
				PrefixListId:         aws.String("pl-87654321"),
			}},
			want: "192.168.0.0/24",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := GetTGWRouteDestination(tt.args.route); got != tt.want {
				t.Errorf("GetTGWRouteDestination() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetTGWRouteTarget tests the GetTGWRouteTarget function which extracts the target identifier
// from a Transit Gateway route. Tests validate attachment ID extraction, blackhole state handling,
// empty attachment array handling, nil pointer handling, and ECMP behavior where only the first
// attachment is used.
func TestGetTGWRouteTarget(t *testing.T) {
	type args struct {
		route types.TransitGatewayRoute
	}
	tests := map[string]struct {
		args args
		want string
	}{
		"Extract attachment ID from first attachment": {
			args: args{route: types.TransitGatewayRoute{
				State: types.TransitGatewayRouteStateActive,
				TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
					{TransitGatewayAttachmentId: aws.String("tgw-attach-12345678")},
				},
			}},
			want: "tgw-attach-12345678",
		},
		"Return blackhole for blackhole state": {
			args: args{route: types.TransitGatewayRoute{
				State: types.TransitGatewayRouteStateBlackhole,
			}},
			want: "blackhole",
		},
		"Return empty string for empty TransitGatewayAttachments array": {
			args: args{route: types.TransitGatewayRoute{
				State:                     types.TransitGatewayRouteStateActive,
				TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{},
			}},
			want: "",
		},
		"Return empty string for nil attachment ID pointer": {
			args: args{route: types.TransitGatewayRoute{
				State: types.TransitGatewayRouteStateActive,
				TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
					{TransitGatewayAttachmentId: nil},
				},
			}},
			want: "",
		},
		"Use first attachment for routes with multiple attachments (ECMP)": {
			args: args{route: types.TransitGatewayRoute{
				State: types.TransitGatewayRouteStateActive,
				TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
					{TransitGatewayAttachmentId: aws.String("tgw-attach-first")},
					{TransitGatewayAttachmentId: aws.String("tgw-attach-second")},
				},
			}},
			want: "tgw-attach-first",
		},
		"Blackhole state overrides attachment": {
			args: args{route: types.TransitGatewayRoute{
				State: types.TransitGatewayRouteStateBlackhole,
				TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
					{TransitGatewayAttachmentId: aws.String("tgw-attach-12345678")},
				},
			}},
			want: "blackhole",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := GetTGWRouteTarget(tt.args.route); got != tt.want {
				t.Errorf("GetTGWRouteTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetTransitGatewayRouteTableRoutes tests the GetTransitGatewayRouteTableRoutes function which
// retrieves all routes for a Transit Gateway route table using the AWS SearchTransitGatewayRoutes API.
// Tests cover successful route retrieval, empty route tables, API error handling, and verification
// that the correct route table ID is passed to the API call.
func TestGetTransitGatewayRouteTableRoutes(t *testing.T) {
	type args struct {
		routeTableId string
		svc          EC2SearchTransitGatewayRoutesAPI
	}
	tests := map[string]struct {
		args    args
		want    []types.TransitGatewayRoute
		wantErr bool
	}{
		"Success - retrieve routes": {
			args: args{
				routeTableId: "tgw-rtb-12345678",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					return &ec2.SearchTransitGatewayRoutesOutput{
						Routes: []types.TransitGatewayRoute{
							{
								DestinationCidrBlock: aws.String("10.0.0.0/16"),
								State:                types.TransitGatewayRouteStateActive,
								Type:                 types.TransitGatewayRouteTypeStatic,
								TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
									{TransitGatewayAttachmentId: aws.String("tgw-attach-12345678")},
								},
							},
							{
								DestinationCidrBlock: aws.String("192.168.0.0/24"),
								State:                types.TransitGatewayRouteStateBlackhole,
								Type:                 types.TransitGatewayRouteTypeStatic,
							},
						},
					}, nil
				}),
			},
			want: []types.TransitGatewayRoute{
				{
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					State:                types.TransitGatewayRouteStateActive,
					Type:                 types.TransitGatewayRouteTypeStatic,
					TransitGatewayAttachments: []types.TransitGatewayRouteAttachment{
						{TransitGatewayAttachmentId: aws.String("tgw-attach-12345678")},
					},
				},
				{
					DestinationCidrBlock: aws.String("192.168.0.0/24"),
					State:                types.TransitGatewayRouteStateBlackhole,
					Type:                 types.TransitGatewayRouteTypeStatic,
				},
			},
			wantErr: false,
		},
		"Success - empty route table": {
			args: args{
				routeTableId: "tgw-rtb-empty",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					return &ec2.SearchTransitGatewayRoutesOutput{
						Routes: []types.TransitGatewayRoute{},
					}, nil
				}),
			},
			want:    []types.TransitGatewayRoute{},
			wantErr: false,
		},
		"Error - API call fails": {
			args: args{
				routeTableId: "tgw-rtb-error",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					return nil, errors.New("route table not found")
				}),
			},
			want:    nil,
			wantErr: true,
		},
		"Verify correct route table ID passed": {
			args: args{
				routeTableId: "tgw-rtb-specific",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					if params.TransitGatewayRouteTableId == nil || *params.TransitGatewayRouteTableId != "tgw-rtb-specific" {
						return nil, errors.New("incorrect route table ID")
					}
					return &ec2.SearchTransitGatewayRoutesOutput{
						Routes: []types.TransitGatewayRoute{},
					}, nil
				}),
			},
			want:    []types.TransitGatewayRoute{},
			wantErr: false,
		},
		"Error - InvalidRouteTableID.NotFound": {
			args: args{
				routeTableId: "tgw-rtb-notfound",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					return nil, &smithy.GenericAPIError{Code: "InvalidRouteTableID.NotFound", Message: "route table not found"}
				}),
			},
			want:    nil,
			wantErr: true,
		},
		"Error - UnauthorizedOperation": {
			args: args{
				routeTableId: "tgw-rtb-unauthorized",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					return nil, &smithy.GenericAPIError{Code: "UnauthorizedOperation", Message: "insufficient IAM permissions"}
				}),
			},
			want:    nil,
			wantErr: true,
		},
		"Error - context timeout": {
			args: args{
				routeTableId: "tgw-rtb-timeout",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					return nil, context.DeadlineExceeded
				}),
			},
			want:    nil,
			wantErr: true,
		},
		"Verify context is passed and not nil": {
			args: args{
				routeTableId: "tgw-rtb-context",
				svc: mockEC2SearchTransitGatewayRoutesAPI(func(ctx context.Context, params *ec2.SearchTransitGatewayRoutesInput, optFns ...func(*ec2.Options)) (*ec2.SearchTransitGatewayRoutesOutput, error) {
					if ctx == nil {
						return nil, errors.New("context must not be nil")
					}
					return &ec2.SearchTransitGatewayRoutesOutput{
						Routes: []types.TransitGatewayRoute{},
					}, nil
				}),
			},
			want:    []types.TransitGatewayRoute{},
			wantErr: false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := GetTransitGatewayRouteTableRoutes(context.Background(), tt.args.routeTableId, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTransitGatewayRouteTableRoutes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				opts := []cmp.Option{
					cmpopts.IgnoreUnexported(types.TransitGatewayRoute{}, types.TransitGatewayRouteAttachment{}),
				}

				if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
					t.Errorf("GetTransitGatewayRouteTableRoutes() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
