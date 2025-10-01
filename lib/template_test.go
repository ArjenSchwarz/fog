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

// TestFilterNaclEntriesByLogicalId verifies that NACL entries are correctly
// filtered by logical ID and converted to NetworkAclEntry structures.
func TestFilterNaclEntriesByLogicalId(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("CIDR"), ParameterValue: aws.String("10.0.0.0/24")},
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"IngressRule": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]interface{}{
					"NetworkAclId": "REF: TestNacl",
					"Protocol":     6.0,
					"RuleNumber":   100.0,
					"CidrBlock":    map[string]interface{}{"Ref": "CIDR"},
					"RuleAction":   "allow",
					"Egress":       false,
				},
			},
			"EgressRule": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]interface{}{
					"NetworkAclId": "REF: TestNacl",
					"Protocol":     "17",
					"RuleNumber":   110.0,
					"CidrBlock":    "0.0.0.0/0",
					"RuleAction":   "deny",
					"Egress":       true,
				},
			},
			"OtherNacl": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]interface{}{
					"NetworkAclId": "REF: OtherNacl",
					"Protocol":     "6",
					"RuleNumber":   200.0,
					"CidrBlock":    "192.168.0.0/16",
					"RuleAction":   "allow",
					"Egress":       false,
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterNaclEntriesByLogicalId("TestNacl", template, params)

	if len(results) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(results))
	}

	// Check ingress rule (rule number 100)
	if entry, ok := results["I100"]; ok {
		if *entry.Egress {
			t.Errorf("Expected ingress rule, got egress")
		}
		if *entry.RuleNumber != 100 {
			t.Errorf("Expected rule number 100, got %d", *entry.RuleNumber)
		}
		if *entry.CidrBlock != "10.0.0.0/24" {
			t.Errorf("Expected CIDR 10.0.0.0/24, got %s", *entry.CidrBlock)
		}
	} else {
		t.Errorf("Expected ingress rule I100 not found")
	}

	// Check egress rule (rule number 110)
	if entry, ok := results["E110"]; ok {
		if !*entry.Egress {
			t.Errorf("Expected egress rule, got ingress")
		}
		if *entry.RuleNumber != 110 {
			t.Errorf("Expected rule number 110, got %d", *entry.RuleNumber)
		}
	} else {
		t.Errorf("Expected egress rule E110 not found")
	}

	// Ensure the OtherNacl entry is not included
	if _, ok := results["I200"]; ok {
		t.Errorf("Expected OtherNacl entry not to be included")
	}
}

// TestFilterRoutesByLogicalId verifies that routes are correctly filtered by
// logical route table ID and converted to Route structures.
func TestFilterRoutesByLogicalId(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("GW"), ParameterValue: aws.String("igw-123")},
	}
	logicalToPhysical := map[string]string{
		"MyNATGateway": "nat-456",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"Route1": {
				Type: "AWS::EC2::Route",
				Properties: map[string]interface{}{
					"RouteTableId":         "REF: TestRouteTable",
					"DestinationCidrBlock": "0.0.0.0/0",
					"GatewayId":            map[string]interface{}{"Ref": "GW"},
				},
			},
			"Route2": {
				Type: "AWS::EC2::Route",
				Properties: map[string]interface{}{
					"RouteTableId":         "REF: TestRouteTable",
					"DestinationCidrBlock": "10.0.0.0/8",
					"NatGatewayId":         "REF: MyNATGateway",
				},
			},
			"OtherRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]interface{}{
					"RouteTableId":         "REF: OtherRouteTable",
					"DestinationCidrBlock": "192.168.0.0/16",
					"GatewayId":            "local",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterRoutesByLogicalId("TestRouteTable", template, params, logicalToPhysical)

	if len(results) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(results))
	}

	// Check that the default route exists with the correct gateway
	if route, ok := results["0.0.0.0/0"]; ok {
		if *route.DestinationCidrBlock != "0.0.0.0/0" {
			t.Errorf("Expected destination 0.0.0.0/0, got %s", *route.DestinationCidrBlock)
		}
		if *route.GatewayId != "igw-123" {
			t.Errorf("Expected gateway igw-123, got %s", *route.GatewayId)
		}
	} else {
		t.Errorf("Expected default route not found")
	}

	// Check that the NAT gateway route exists
	if route, ok := results["10.0.0.0/8"]; ok {
		if *route.NatGatewayId != "nat-456" {
			t.Errorf("Expected NAT gateway nat-456, got %s", *route.NatGatewayId)
		}
	} else {
		t.Errorf("Expected NAT gateway route not found")
	}

	// Ensure the other route table's route is not included
	if _, ok := results["192.168.0.0/16"]; ok {
		t.Errorf("Expected OtherRouteTable route not to be included")
	}
}

// TestCfnTemplateTransform_Value ensures that the Value method returns the
// correct type based on which field is populated.
func TestCfnTemplateTransform_Value(t *testing.T) {
	tests := []struct {
		name      string
		transform CfnTemplateTransform
		want      interface{}
	}{
		{
			name:      "String value",
			transform: CfnTemplateTransform{String: aws.String("test-string")},
			want:      "test-string",
		},
		{
			name:      "StringArray value",
			transform: CfnTemplateTransform{StringArray: &[]string{"item1", "item2"}},
			want:      []string{"item1", "item2"},
		},
		{
			name:      "Nil value",
			transform: CfnTemplateTransform{},
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.transform.Value()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCfnTemplateTransform_UnmarshalJSON validates JSON unmarshaling for
// CfnTemplateTransform, which can be either a string or an array of strings.
func TestCfnTemplateTransform_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    CfnTemplateTransform
		wantErr bool
	}{
		{
			name: "String value",
			json: `"AWS::Serverless-2016-10-31"`,
			want: CfnTemplateTransform{String: aws.String("AWS::Serverless-2016-10-31")},
		},
		{
			name: "String array",
			json: `["AWS::Serverless-2016-10-31", "AWS::Include"]`,
			want: CfnTemplateTransform{StringArray: &[]string{"AWS::Serverless-2016-10-31", "AWS::Include"}},
		},
		{
			name: "Mixed interface array",
			json: `["Transform1", "Transform2"]`,
			want: CfnTemplateTransform{StringArray: &[]string{"Transform1", "Transform2"}},
		},
		{
			name:    "Invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got CfnTemplateTransform
			err := got.UnmarshalJSON([]byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotVal := got.Value()
				wantVal := tt.want.Value()
				if !reflect.DeepEqual(gotVal, wantVal) {
					t.Errorf("UnmarshalJSON() got = %v, want %v", gotVal, wantVal)
				}
			}
		})
	}
}

// TestParseTemplateString_PseudoParameters tests that pseudo-parameters like
// AWS::AccountId, AWS::Region, etc. are resolved correctly via customRefHandler
func TestParseTemplateString_PseudoParameters(t *testing.T) {
	template := `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "TestResource": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "AccountId": {"Ref": "AWS::AccountId"},
        "Region": {"Ref": "AWS::Region"},
        "StackName": {"Ref": "AWS::StackName"},
        "StackId": {"Ref": "AWS::StackId"},
        "NotificationARNs": {"Ref": "AWS::NotificationARNs"},
        "NoValue": {"Ref": "AWS::NoValue"},
        "UnknownRef": {"Ref": "SomeUnknownRef"}
      }
    }
  }
}`

	body := ParseTemplateString(template, nil)
	props := body.Resources["TestResource"].Properties

	// Test that pseudo-parameters are resolved correctly
	if accountId, ok := props["AccountId"].(string); !ok || accountId != "123456789012" {
		t.Errorf("AccountId = %v, want 123456789012", props["AccountId"])
	}

	if region, ok := props["Region"].(string); !ok || region != "ap-southeast-2" {
		t.Errorf("Region = %v, want ap-southeast-2", props["Region"])
	}

	if stackName, ok := props["StackName"].(string); !ok || stackName != "YOUR_STACK_NAME" {
		t.Errorf("StackName = %v, want YOUR_STACK_NAME", props["StackName"])
	}

	if stackId, ok := props["StackId"].(string); !ok || stackId == "" {
		t.Errorf("StackId = %v, want non-empty ARN", props["StackId"])
	}

	if notificationARNs, ok := props["NotificationARNs"].([]interface{}); !ok || len(notificationARNs) == 0 {
		t.Errorf("NotificationARNs = %v, want non-empty array", props["NotificationARNs"])
	}

	// NoValue should be nil
	if props["NoValue"] != nil {
		t.Errorf("NoValue = %v, want nil", props["NoValue"])
	}

	// Unknown references should get a "REF: " prefix
	if unknownRef, ok := props["UnknownRef"].(string); !ok || unknownRef != "REF: SomeUnknownRef" {
		t.Errorf("UnknownRef = %v, want REF: SomeUnknownRef", props["UnknownRef"])
	}
}

// TestParseTemplateString_ParameterDefaults tests that parameter defaults
// are resolved correctly when referenced in the template
func TestParseTemplateString_ParameterDefaults(t *testing.T) {
	template := `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Parameters": {
    "ParamWithDefault": {
      "Type": "String",
      "Default": "MyDefaultValue"
    },
    "ParamNoDefault": {
      "Type": "String"
    }
  },
  "Resources": {
    "TestResource": {
      "Type": "AWS::S3::Bucket",
      "Properties": {
        "WithDefault": {"Ref": "ParamWithDefault"},
        "WithoutDefault": {"Ref": "ParamNoDefault"}
      }
    }
  }
}`

	body := ParseTemplateString(template, nil)
	props := body.Resources["TestResource"].Properties

	// Parameter with default should resolve to the default value
	if withDefault, ok := props["WithDefault"].(string); !ok || withDefault != "MyDefaultValue" {
		t.Errorf("WithDefault = %v, want MyDefaultValue", props["WithDefault"])
	}

	// Parameter without default should get "REF: " prefix
	if withoutDefault, ok := props["WithoutDefault"].(string); !ok || withoutDefault != "REF: ParamNoDefault" {
		t.Errorf("WithoutDefault = %v, want REF: ParamNoDefault", props["WithoutDefault"])
	}
}
