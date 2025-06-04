package lib

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type mockEC2DescribeNaclsAPI func(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error)

func (m mockEC2DescribeNaclsAPI) DescribeNetworkAcls(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
	return m(ctx, params, optFns...)
}

type mockEC2DescribeManagedPrefixListsAPI func(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error)

func (m mockEC2DescribeManagedPrefixListsAPI) DescribeManagedPrefixLists(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error) {
	return m(ctx, params, optFns...)
}

// type mockEC2DescribeRouteTablesAPI func(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error)

// func (m mockEC2DescribeRouteTablesAPI) DescribeRouteTables(ctx context.Context, params *ec2.DescribeRouteTablesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeRouteTablesOutput, error) {
// 	return m(ctx, params, optFns...)
// }

func TestGetNacl(t *testing.T) {
	type args struct {
		naclid string
		svc    EC2DescribeNaclsAPI
	}
	tests := []struct {
		name    string
		args    args
		want    types.NetworkAcl
		wantErr bool
	}{
		{"Test Get Nacl Success", args{"naclid", mockEC2DescribeNaclsAPI(func(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
			return &ec2.DescribeNetworkAclsOutput{
					NetworkAcls: []types.NetworkAcl{
						{
							Associations: []types.NetworkAclAssociation{
								{
									NetworkAclAssociationId: aws.String("NetworkAclAssociationId"),
									NetworkAclId:            aws.String("NetworkAclId"),
									SubnetId:                aws.String("SubnetId"),
								},
							},
							Entries: []types.NetworkAclEntry{
								{
									CidrBlock:     aws.String("Cidr"),
									Egress:        aws.Bool(true),
									IcmpTypeCode:  &types.IcmpTypeCode{Code: aws.Int32(12), Type: aws.Int32(12)},
									Ipv6CidrBlock: aws.String("Ipv6CidrBlock"),
									PortRange:     &types.PortRange{From: aws.Int32(12), To: aws.Int32(12)},
									Protocol:      aws.String("Protocol"),
									RuleAction:    types.RuleActionAllow,
									RuleNumber:    aws.Int32(12),
								},
							},
							IsDefault:    aws.Bool(true),
							NetworkAclId: aws.String("NetworkAclId"),
							OwnerId:      aws.String("OwnerId"),
							Tags: []types.Tag{
								{
									Key:   aws.String("Key"),
									Value: aws.String("Value"),
								},
							},
							VpcId: aws.String("VpcId"),
						},
					},
				},
				nil
		})}, types.NetworkAcl{
			Associations: []types.NetworkAclAssociation{
				{
					NetworkAclAssociationId: aws.String("NetworkAclAssociationId"),
					NetworkAclId:            aws.String("NetworkAclId"),
					SubnetId:                aws.String("SubnetId"),
				},
			},
			Entries: []types.NetworkAclEntry{
				{
					CidrBlock:     aws.String("Cidr"),
					Egress:        aws.Bool(true),
					IcmpTypeCode:  &types.IcmpTypeCode{Code: aws.Int32(12), Type: aws.Int32(12)},
					Ipv6CidrBlock: aws.String("Ipv6CidrBlock"),
					PortRange:     &types.PortRange{From: aws.Int32(12), To: aws.Int32(12)},
					Protocol:      aws.String("Protocol"),
					RuleAction:    types.RuleActionAllow,
					RuleNumber:    aws.Int32(12),
				},
			},
			IsDefault:    aws.Bool(true),
			NetworkAclId: aws.String("NetworkAclId"),
			OwnerId:      aws.String("OwnerId"),
			Tags: []types.Tag{
				{
					Key:   aws.String("Key"),
					Value: aws.String("Value"),
				},
			},
			VpcId: aws.String("VpcId"),
		}, false},
		{"Test Get Nacl Error", args{"naclid", mockEC2DescribeNaclsAPI(func(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
			return nil, errors.New("error")
		})}, types.NetworkAcl{}, true},
		{"Test Get Nacl No Match", args{"naclid", mockEC2DescribeNaclsAPI(func(ctx context.Context, params *ec2.DescribeNetworkAclsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeNetworkAclsOutput, error) {
			return nil, errors.New("No match")
		})}, types.NetworkAcl{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetNacl(tt.args.naclid, tt.args.svc)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetNacl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNacl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareNaclEntries(t *testing.T) {
	type args struct {
		nacl1 types.NetworkAclEntry
		nacl2 types.NetworkAclEntry
	}
	fullnacl := types.NetworkAclEntry{
		CidrBlock:     aws.String("Cidr"),
		Egress:        aws.Bool(true),
		IcmpTypeCode:  &types.IcmpTypeCode{Code: aws.Int32(12), Type: aws.Int32(-1)},
		Ipv6CidrBlock: aws.String("ipv6"),
		PortRange:     &types.PortRange{From: aws.Int32(443), To: aws.Int32(444)},
		Protocol:      aws.String("protocol"),
		RuleAction:    types.RuleActionAllow,
		RuleNumber:    aws.Int32(1),
	}
	fullnaclcopy := fullnacl
	// full tests for strings are covered in stringpointervaluematch
	cidr := fullnacl
	cidr.CidrBlock = aws.String("Different")
	ipv6 := fullnacl
	ipv6.Ipv6CidrBlock = aws.String("Different")
	protocol := fullnacl
	protocol.Protocol = aws.String("Different")
	ruleaction := fullnacl
	ruleaction.RuleAction = types.RuleActionDeny
	// include nil tests
	egressnil := fullnacl
	egressnil.Egress = nil
	egressdiff := fullnacl
	egressdiff.Egress = aws.Bool(false)
	rulenumbernil := fullnacl
	rulenumbernil.RuleNumber = nil
	rulenumberdiff := fullnacl
	rulenumberdiff.RuleNumber = aws.Int32(42)
	icmpnil := fullnacl
	icmpnil.IcmpTypeCode = nil
	icmpdifftype := fullnacl
	icmpdifftype.IcmpTypeCode = &types.IcmpTypeCode{Code: aws.Int32(12), Type: aws.Int32(42)}
	icmpdiffcode := fullnacl
	icmpdiffcode.IcmpTypeCode = &types.IcmpTypeCode{Code: aws.Int32(42), Type: aws.Int32(-1)}
	portnil := fullnacl
	portnil.PortRange = nil
	portdifffrom := fullnacl
	portdifffrom.PortRange = &types.PortRange{From: aws.Int32(42), To: aws.Int32(444)}
	portdiffto := fullnacl
	portdiffto.PortRange = &types.PortRange{From: aws.Int32(443), To: aws.Int32(42)}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Empty NACLs match", args{nacl1: types.NetworkAclEntry{}, nacl2: types.NetworkAclEntry{}}, true},
		{"Full NACLs match", args{nacl1: fullnacl, nacl2: fullnaclcopy}, true},
		{"Cidr mismatch fails", args{nacl1: fullnacl, nacl2: cidr}, false},
		{"Ipv6 mismatch fails", args{nacl1: fullnacl, nacl2: ipv6}, false},
		{"Protocol mismatch fails", args{nacl1: fullnacl, nacl2: protocol}, false},
		{"Ruleaction mismatch fails", args{nacl1: fullnacl, nacl2: ruleaction}, false},
		{"Egress = nil mismatch fails", args{nacl1: fullnacl, nacl2: egressnil}, false},
		{"Egress different mismatch fails", args{nacl1: fullnacl, nacl2: egressdiff}, false},
		{"Rulenr = nil mismatch fails", args{nacl1: fullnacl, nacl2: rulenumbernil}, false},
		{"Rulenr different mismatch fails", args{nacl1: fullnacl, nacl2: rulenumberdiff}, false},
		{"ICMP = nil mismatch fails", args{nacl1: fullnacl, nacl2: icmpnil}, false},
		{"ICMP Code different mismatch fails", args{nacl1: fullnacl, nacl2: icmpdiffcode}, false},
		{"ICMP Type different mismatch fails", args{nacl1: fullnacl, nacl2: icmpdifftype}, false},
		{"PortRange = nil mismatch fails", args{nacl1: fullnacl, nacl2: portnil}, false},
		{"PortRange From different mismatch fails", args{nacl1: fullnacl, nacl2: portdifffrom}, false},
		{"PortRange To different mismatch fails", args{nacl1: fullnacl, nacl2: portdiffto}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareNaclEntries(tt.args.nacl1, tt.args.nacl2); got != tt.want {
				t.Errorf("CompareNaclEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareRoutes(t *testing.T) {
	// Comparisons are tested with stringPointerValueMatch
	// This will just go through every option
	type args struct {
		route1 types.Route
		route2 types.Route
	}
	route1 := types.Route{
		DestinationCidrBlock: aws.String("Randomblock"),
	}
	route2 := route1
	fullRoute := types.Route{
		CarrierGatewayId:            aws.String("Carrier"),
		CoreNetworkArn:              aws.String("Core"),
		DestinationCidrBlock:        aws.String("Cidr"),
		DestinationIpv6CidrBlock:    aws.String("IPv6Cidr"),
		DestinationPrefixListId:     aws.String("Prefix"),
		EgressOnlyInternetGatewayId: aws.String("Egress"),
		GatewayId:                   aws.String("Gateway"),
		InstanceId:                  aws.String("Instance"),
		InstanceOwnerId:             aws.String("InstanceOwner"),
		LocalGatewayId:              aws.String("Local"),
		NatGatewayId:                aws.String("NAT"),
		NetworkInterfaceId:          aws.String("ENI"),
		TransitGatewayId:            aws.String("Transit"),
		VpcPeeringConnectionId:      aws.String("Peer"),
		State:                       types.RouteStateActive,
		Origin:                      types.RouteOriginCreateRoute,
	}
	fullRouteCopy := fullRoute
	carrier := fullRoute
	carrier.CarrierGatewayId = aws.String("Different")
	core := fullRoute
	core.CoreNetworkArn = aws.String("Different")
	cidr := fullRoute
	cidr.DestinationCidrBlock = aws.String("Different")
	ipv6 := fullRoute
	ipv6.DestinationIpv6CidrBlock = aws.String("Different")
	prefix := fullRoute
	prefix.DestinationPrefixListId = aws.String("Different")
	egress := fullRoute
	egress.EgressOnlyInternetGatewayId = aws.String("Different")
	gateway := fullRoute
	gateway.GatewayId = aws.String("Different")
	instance := fullRoute
	instance.InstanceId = aws.String("Different")
	owner := fullRoute
	owner.InstanceOwnerId = aws.String("Different")
	local := fullRoute
	local.LocalGatewayId = aws.String("Different")
	nat := fullRoute
	nat.NatGatewayId = aws.String("Different")
	eni := fullRoute
	eni.NetworkInterfaceId = aws.String("Different")
	tgw := fullRoute
	tgw.TransitGatewayId = aws.String("Different")
	peer := fullRoute
	peer.VpcPeeringConnectionId = aws.String("Different")
	state := fullRoute
	state.State = types.RouteStateBlackhole
	origin := fullRoute
	origin.Origin = types.RouteOriginCreateRouteTable
	noppeer := fullRoute
	noppeer.VpcPeeringConnectionId = nil
	noppeer.State = types.RouteStateBlackhole
	noppeerActive := noppeer
	noppeerActive.State = types.RouteStateActive
	noppeerBlackhole := noppeer
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Match with empty routes", args{route1: types.Route{}, route2: types.Route{}}, true},
		{"Match with limited filled in", args{route1: route1, route2: route2}, true},
		{"Match fully filled in", args{route1: fullRoute, route2: fullRouteCopy}, true},
		{"Different value CarrierGatewayId", args{route1: fullRoute, route2: carrier}, false},
		{"Different value CoreNetworkArn", args{route1: fullRoute, route2: core}, false},
		{"Different value DestinationCidrBlock", args{route1: fullRoute, route2: cidr}, false},
		{"Different value DestinationIpv6CidrBlock", args{route1: fullRoute, route2: ipv6}, false},
		{"Different value DestinationPrefixListId", args{route1: fullRoute, route2: prefix}, false},
		{"Different value EgressOnlyInternetGatewayId", args{route1: fullRoute, route2: egress}, false},
		{"Different value GatewayId", args{route1: fullRoute, route2: gateway}, false},
		{"Different value InstanceId", args{route1: fullRoute, route2: instance}, false},
		{"Different value InstanceOwnerId", args{route1: fullRoute, route2: owner}, false},
		{"Different value LocalGatewayId", args{route1: fullRoute, route2: local}, false},
		{"Different value NatGatewayId", args{route1: fullRoute, route2: nat}, false},
		{"Different value NetworkInterfaceId", args{route1: fullRoute, route2: eni}, false},
		{"Different value TransitGatewayId", args{route1: fullRoute, route2: tgw}, false},
		{"Different value VpcPeeringConnectionId", args{route1: fullRoute, route2: peer}, false},
		{"Different value State", args{route1: fullRoute, route2: state}, false},
		{"Different value Origin", args{route1: fullRoute, route2: origin}, false},
		{"Nil VpcPeeringConnectionId with Blackhole state", args{route1: noppeer, route2: noppeerActive}, false},
		{"Nil VpcPeeringConnectionId with same state", args{route1: noppeer, route2: noppeerBlackhole}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareRoutes(tt.args.route1, tt.args.route2, []string{}); got != tt.want {
				t.Errorf("CompareRoutes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRouteDestination(t *testing.T) {
	type args struct {
		route types.Route
	}
	ipv4route := types.Route{
		DestinationCidrBlock: aws.String("10.0.0.0/8"),
	}
	plroute := types.Route{
		DestinationPrefixListId: aws.String("pl-randomlist"),
	}
	ipv6route := types.Route{
		DestinationIpv6CidrBlock: aws.String("fakeblock"),
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"If IPv4 set return that", args{route: ipv4route}, "10.0.0.0/8"},
		{"If Prefixlist set return that", args{route: plroute}, "pl-randomlist"},
		{"If IPv6 set return that", args{route: ipv6route}, "fakeblock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRouteDestination(tt.args.route); got != tt.want {
				t.Errorf("GetRouteDestination() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stringPointerValueMatch(t *testing.T) {
	type args struct {
		pointer1 *string
		pointer2 *string
	}
	value1 := "value1"
	othervalue1 := "value1"
	value2 := "value2"
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"Match when both are nil", args{}, true},
		{"Match when values are the same", args{pointer1: &value1, pointer2: &othervalue1}, true},
		{"No match when pointer 1 is nil", args{pointer2: &value1}, false},
		{"No match when pointer 2 is nil", args{pointer1: &value1}, false},
		{"No match when values are different", args{pointer1: &value1, pointer2: &value2}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringPointerValueMatch(tt.args.pointer1, tt.args.pointer2); got != tt.want {
				t.Errorf("stringPointerValueMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetManagedPrefixLists(t *testing.T) {
	type args struct {
		svc EC2DescribeManagedPrefixListsAPI
	}
	pl1 := types.ManagedPrefixList{
		PrefixListArn: aws.String("arn:aws:ec2:pl/test1"),
		PrefixListId:  aws.String("pl-test1"),
	}
	pl2 := types.ManagedPrefixList{
		PrefixListArn: aws.String("arn:aws:ec2:pl/test2"),
		PrefixListId:  aws.String("pl-test2"),
	}

	tests := []struct {
		name string
		args args
		want []types.ManagedPrefixList
	}{
		{
			name: "return prefix lists",
			args: args{svc: mockEC2DescribeManagedPrefixListsAPI(func(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error) {
				return &ec2.DescribeManagedPrefixListsOutput{PrefixLists: []types.ManagedPrefixList{pl1, pl2}}, nil
			})},
			want: []types.ManagedPrefixList{pl1, pl2},
		},
		{
			name: "no prefix lists",
			args: args{svc: mockEC2DescribeManagedPrefixListsAPI(func(ctx context.Context, params *ec2.DescribeManagedPrefixListsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeManagedPrefixListsOutput, error) {
				return &ec2.DescribeManagedPrefixListsOutput{}, nil
			})},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetManagedPrefixLists(tt.args.svc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetManagedPrefixLists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRouteTarget(t *testing.T) {
	type args struct {
		route types.Route
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"If CarrierGatewayId set return that", args{route: types.Route{CarrierGatewayId: aws.String("carriergateway")}}, "carriergateway"},
		{"If CoreNetworkArn set return that", args{route: types.Route{CoreNetworkArn: aws.String("corenetworkarn")}}, "corenetworkarn"},
		{"If EgressOnlyInternetGatewayId set return that", args{route: types.Route{EgressOnlyInternetGatewayId: aws.String("egressonlyinternetgatewayid")}}, "egressonlyinternetgatewayid"},
		{"If GatewayId set return that", args{route: types.Route{GatewayId: aws.String("gateway")}}, "gateway"},
		{"If InstanceId set return that", args{route: types.Route{InstanceId: aws.String("instance")}}, "instance"},
		{"If LocalGatewayId set return that", args{route: types.Route{LocalGatewayId: aws.String("local")}}, "local"},
		{"If NatGatewayId set return that", args{route: types.Route{NatGatewayId: aws.String("nat")}}, "nat"},
		{"If NetworkInterfaceId set return that", args{route: types.Route{NetworkInterfaceId: aws.String("eni")}}, "eni"},
		{"If TransitGatewayId set return that", args{route: types.Route{TransitGatewayId: aws.String("tgw")}}, "tgw"},
		{"If VpcPeeringConnectionId set return that", args{route: types.Route{VpcPeeringConnectionId: aws.String("peer")}}, "peer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRouteTarget(tt.args.route); got != tt.want {
				t.Errorf("GetRouteTarget() = %v, want %v", got, tt.want)
			}
		})
	}
}
