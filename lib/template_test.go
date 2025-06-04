package lib

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestParseTemplateString ensures that ParseTemplateString correctly parses
// both JSON and YAML CloudFormation templates and applies parameter overrides.
func TestParseTemplateString(t *testing.T) {
	jsonTemplate := `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Parameters": {
    "NameParam": {"Type": "String", "Default": "DefaultValue"}
  },
  "Resources": {
    "Bucket": {
      "Type": "AWS::S3::Bucket",
      "Properties": {"BucketName": {"Ref": "NameParam"}}
    }
  }
}`

	yamlTemplate := `AWSTemplateFormatVersion: "2010-09-09"
Parameters:
  NameParam:
    Type: String
    Default: DefaultValue
Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref NameParam
`

	overrides := map[string]interface{}{"NameParam": "Overridden"}

	// Provide the template body in both JSON and YAML formats to ensure
	// parsing logic handles each correctly and that parameter overrides are
	// applied.
	tests := []struct {
		name  string
		input string
	}{
		{"JSON", jsonTemplate},
		{"YAML", yamlTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := ParseTemplateString(tt.input, &overrides)
			bucket := body.Resources["Bucket"].Properties["BucketName"].(string)
			if bucket != "Overridden" {
				t.Errorf("BucketName = %s, want Overridden", bucket)
			}
		})
	}
}

// TestNaclResourceToNaclEntry validates conversion from a template resource to a
// NetworkAclEntry structure. The first case checks IPv4 properties with a port
// range and ICMP values, while the second covers IPv6 properties and string
// protocol values.
func TestNaclResourceToNaclEntry(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("CIDR"), ParameterValue: aws.String("10.0.0.0/24")},
		{ParameterKey: aws.String("IPV6"), ParameterValue: aws.String("::/0")},
	}

	// IPv4 entry using numeric Protocol and nested maps for PortRange and ICMP
	// properties. CIDR is supplied via Ref parameter.
	resource1 := CfnTemplateResource{
		Type: "AWS::EC2::NetworkAclEntry",
		Properties: map[string]interface{}{
			"Protocol":   6.0,
			"RuleNumber": 100.0,
			"CidrBlock":  map[string]interface{}{"Ref": "CIDR"},
			"RuleAction": "deny",
			"Egress":     false,
			"PortRange":  map[string]interface{}{"From": "80", "To": 443.0},
			"Icmp":       map[string]interface{}{"Type": "1", "Code": 2.0},
		},
	}

	expected1 := types.NetworkAclEntry{
		CidrBlock:    aws.String("10.0.0.0/24"),
		Egress:       aws.Bool(false),
		IcmpTypeCode: &types.IcmpTypeCode{Type: aws.Int32(1), Code: aws.Int32(2)},
		PortRange:    &types.PortRange{From: aws.Int32(80), To: aws.Int32(443)},
		Protocol:     aws.String("6"),
		RuleAction:   types.RuleActionDeny,
		RuleNumber:   aws.Int32(100),
	}

	got1 := NaclResourceToNaclEntry(resource1, params)
	if !reflect.DeepEqual(got1, expected1) {
		t.Errorf("NaclResourceToNaclEntry() = %#v, want %#v", got1, expected1)
	}

	// IPv6 entry using string Protocol and no port or ICMP data.
	resource2 := CfnTemplateResource{
		Type: "AWS::EC2::NetworkAclEntry",
		Properties: map[string]interface{}{
			"Protocol":      "17",
			"RuleNumber":    110.0,
			"Ipv6CidrBlock": map[string]interface{}{"Ref": "IPV6"},
			"RuleAction":    "allow",
			"Egress":        true,
		},
	}

	expected2 := types.NetworkAclEntry{
		Egress:        aws.Bool(true),
		Ipv6CidrBlock: aws.String("::/0"),
		Protocol:      aws.String("17"),
		RuleAction:    types.RuleActionAllow,
		RuleNumber:    aws.Int32(110),
	}

	got2 := NaclResourceToNaclEntry(resource2, params)
	if !reflect.DeepEqual(got2, expected2) {
		t.Errorf("NaclResourceToNaclEntry() = %#v, want %#v", got2, expected2)
	}
}

// TestRouteResourceToRoute verifies that references and logical IDs are
// correctly resolved when converting a Route resource to its SDK type.
func TestRouteResourceToRoute(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("GateParam"), ParameterValue: aws.String("vgw-1")},
	}
	logical := map[string]string{"TGW": "tgw-123"}

	// Template resource with a mix of literal values, Ref lookups and a
	// logical ID reference in the TransitGatewayId property.
	resource := CfnTemplateResource{
		Type: "AWS::EC2::Route",
		Properties: map[string]interface{}{
			"DestinationCidrBlock": "10.0.0.0/16",
			"GatewayId":            map[string]interface{}{"Ref": "GateParam"},
			"TransitGatewayId":     "REF: TGW",
		},
	}

	expected := types.Route{
		DestinationCidrBlock: aws.String("10.0.0.0/16"),
		GatewayId:            aws.String("vgw-1"),
		TransitGatewayId:     aws.String("tgw-123"),
		Origin:               types.RouteOriginCreateRoute,
		State:                types.RouteStateActive,
	}

	got := RouteResourceToRoute(resource, params, logical)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("RouteResourceToRoute() = %#v, want %#v", got, expected)
	}
}

// TestCfnTemplateBody_ShouldHaveResource exercises the conditional logic for
// including a resource based on defined conditions.
func TestCfnTemplateBody_ShouldHaveResource(t *testing.T) {
	body := CfnTemplateBody{
		Conditions: map[string]bool{"Create": true, "Skip": false},
	}

	// Each case supplies a resource with a different Condition value and
	// expects ShouldHaveResource to honour the template's Conditions map.
	tests := []struct {
		name string
		res  CfnTemplateResource
		want bool
	}{
		{"No condition", CfnTemplateResource{}, true},
		{"Condition true", CfnTemplateResource{Condition: "Create"}, true},
		{"Condition false", CfnTemplateResource{Condition: "Skip"}, false},
		{"Condition missing", CfnTemplateResource{Condition: "Unknown"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := body.ShouldHaveResource(tt.res); got != tt.want {
				t.Errorf("ShouldHaveResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
