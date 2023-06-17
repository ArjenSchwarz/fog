package lib

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func GetNacl(naclid string, svc *ec2.Client) types.NetworkAcl {
	naclids := []string{naclid}
	input := ec2.DescribeNetworkAclsInput{
		NetworkAclIds: naclids,
	}
	result, err := svc.DescribeNetworkAcls(context.TODO(), &input)
	if err != nil {
		panic(err)
	}
	return result.NetworkAcls[0]
}

func GetRouteTable(routetableId string, svc *ec2.Client) types.RouteTable {
	routetableids := []string{routetableId}
	input := ec2.DescribeRouteTablesInput{
		RouteTableIds: routetableids,
	}
	result, err := svc.DescribeRouteTables(context.TODO(), &input)
	if err != nil {
		panic(err)
	}
	return result.RouteTables[0]
}

func CompareNaclEntries(nacl1 types.NetworkAclEntry, nacl2 types.NetworkAclEntry) bool {
	if !stringPointerValueMatch(nacl1.CidrBlock, nacl2.CidrBlock) {
		return false
	}
	if nacl1.Egress == nil && nacl2.Egress != nil ||
		nacl1.Egress != nil && nacl2.Egress == nil ||
		(nacl1.Egress != nil && nacl2.Egress != nil && *nacl1.Egress != *nacl2.Egress) {
		return false
	}
	if nacl1.IcmpTypeCode == nil && nacl2.IcmpTypeCode != nil ||
		nacl1.IcmpTypeCode != nil && nacl2.IcmpTypeCode == nil {
		return false
	}
	if nacl1.IcmpTypeCode != nil && nacl2.IcmpTypeCode != nil {
		if *nacl1.IcmpTypeCode.Code != *nacl2.IcmpTypeCode.Code {
			return false
		}
		if *nacl1.IcmpTypeCode.Type != *nacl2.IcmpTypeCode.Type {
			return false
		}
	}
	if !stringPointerValueMatch(nacl1.Ipv6CidrBlock, nacl2.Ipv6CidrBlock) {
		return false
	}
	if nacl1.PortRange == nil && nacl2.PortRange != nil ||
		nacl1.PortRange != nil && nacl2.PortRange == nil {
		return false
	}
	if nacl1.PortRange != nil && nacl2.PortRange != nil {
		if *nacl1.PortRange.From != *nacl2.PortRange.From {
			return false
		}
		if *nacl1.PortRange.To != *nacl2.PortRange.To {
			return false
		}
	}
	if !stringPointerValueMatch(nacl1.Protocol, nacl2.Protocol) {
		return false
	}
	if nacl1.RuleAction != nacl2.RuleAction {
		return false
	}
	if nacl1.RuleNumber == nil && nacl2.RuleNumber != nil ||
		nacl1.RuleNumber != nil && nacl2.RuleNumber == nil ||
		(nacl1.RuleNumber != nil && nacl2.RuleNumber != nil && *nacl1.RuleNumber != *nacl2.RuleNumber) {
		return false
	}
	return true
}

func CompareRoutes(route1 types.Route, route2 types.Route) bool {
	if !stringPointerValueMatch(route1.CarrierGatewayId, route2.CarrierGatewayId) {
		return false
	}
	if !stringPointerValueMatch(route1.CoreNetworkArn, route2.CoreNetworkArn) {
		return false
	}
	if !stringPointerValueMatch(route1.DestinationCidrBlock, route2.DestinationCidrBlock) {
		return false
	}
	if !stringPointerValueMatch(route1.DestinationIpv6CidrBlock, route2.DestinationIpv6CidrBlock) {
		return false
	}
	if !stringPointerValueMatch(route1.DestinationPrefixListId, route2.DestinationPrefixListId) {
		return false
	}
	if !stringPointerValueMatch(route1.EgressOnlyInternetGatewayId, route2.EgressOnlyInternetGatewayId) {
		return false
	}
	if !stringPointerValueMatch(route1.GatewayId, route2.GatewayId) {
		return false
	}
	if !stringPointerValueMatch(route1.InstanceId, route2.InstanceId) {
		return false
	}
	if !stringPointerValueMatch(route1.InstanceOwnerId, route2.InstanceOwnerId) {
		return false
	}
	if !stringPointerValueMatch(route1.LocalGatewayId, route2.LocalGatewayId) {
		return false
	}
	if !stringPointerValueMatch(route1.NatGatewayId, route2.NatGatewayId) {
		return false
	}
	if !stringPointerValueMatch(route1.NetworkInterfaceId, route2.NetworkInterfaceId) {
		return false
	}
	if !stringPointerValueMatch(route1.TransitGatewayId, route2.TransitGatewayId) {
		return false
	}
	if !stringPointerValueMatch(route1.VpcPeeringConnectionId, route2.VpcPeeringConnectionId) {
		return false
	}
	if string(route1.Origin) != string(route2.Origin) {
		return false
	}
	if string(route1.State) != string(route2.State) {
		return false
	}
	return true
}

func GetRouteDestination(route types.Route) string {
	var result string
	if route.DestinationCidrBlock != nil {
		result = *route.DestinationCidrBlock
	} else if route.DestinationPrefixListId != nil {
		result = *route.DestinationPrefixListId
	} else {
		result = *route.DestinationIpv6CidrBlock
	}
	return result
}

func stringPointerValueMatch(pointer1 *string, pointer2 *string) bool {
	// if both nil, they match
	if pointer1 == nil && pointer2 == nil {
		return true
	}
	// if only 1 is nil, they don't match
	if pointer1 == nil || pointer2 == nil {
		return false
	}
	// otherwise the values need to match
	return *pointer1 == *pointer2
}
