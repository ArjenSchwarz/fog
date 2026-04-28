package lib

import (
	"reflect"
	"strings"
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

	overrides := map[string]any{"NameParam": "Overridden"}

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
			body, err := ParseTemplateString(tt.input, &overrides)
			if err != nil {
				t.Fatalf("ParseTemplateString() error = %v", err)
			}
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
		{ParameterKey: aws.String("ProtocolParam"), ParameterValue: aws.String("-1")},
		{ParameterKey: aws.String("EgressParam"), ParameterValue: aws.String("true")},
		{ParameterKey: aws.String("RuleActionParam"), ParameterValue: aws.String("deny")},
	}

	// IPv4 entry using numeric Protocol and nested maps for PortRange and ICMP
	// properties. CIDR is supplied via Ref parameter.
	resource1 := CfnTemplateResource{
		Type: "AWS::EC2::NetworkAclEntry",
		Properties: map[string]any{
			"Protocol":   6.0,
			"RuleNumber": 100.0,
			"CidrBlock":  map[string]any{"Ref": "CIDR"},
			"RuleAction": "deny",
			"Egress":     false,
			"PortRange":  map[string]any{"From": "80", "To": 443.0},
			"Icmp":       map[string]any{"Type": "1", "Code": 2.0},
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
		Properties: map[string]any{
			"Protocol":      "17",
			"RuleNumber":    110.0,
			"Ipv6CidrBlock": map[string]any{"Ref": "IPV6"},
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

	// Parameterized entry ensures Ref-based values are resolved before drift
	// comparison so protocol, egress, and rule action do not silently fall back
	// to zero values.
	resource3 := CfnTemplateResource{
		Type: "AWS::EC2::NetworkAclEntry",
		Properties: map[string]any{
			"Protocol":   map[string]any{"Ref": "ProtocolParam"},
			"RuleNumber": 120.0,
			"CidrBlock":  map[string]any{"Ref": "CIDR"},
			"RuleAction": map[string]any{"Ref": "RuleActionParam"},
			"Egress":     map[string]any{"Ref": "EgressParam"},
		},
	}

	expected3 := types.NetworkAclEntry{
		CidrBlock:  aws.String("10.0.0.0/24"),
		Egress:     aws.Bool(true),
		Protocol:   aws.String("-1"),
		RuleAction: types.RuleActionDeny,
		RuleNumber: aws.Int32(120),
	}

	got3 := NaclResourceToNaclEntry(resource3, params)
	if !reflect.DeepEqual(got3, expected3) {
		t.Errorf("NaclResourceToNaclEntry() = %#v, want %#v", got3, expected3)
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
		Properties: map[string]any{
			"DestinationCidrBlock": "10.0.0.0/16",
			"GatewayId":            map[string]any{"Ref": "GateParam"},
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
				Properties: map[string]any{
					"NetworkAclId": "REF: TestNacl",
					"Protocol":     6.0,
					"RuleNumber":   100.0,
					"CidrBlock":    map[string]any{"Ref": "CIDR"},
					"RuleAction":   "allow",
					"Egress":       false,
				},
			},
			"EgressRule": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
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
				Properties: map[string]any{
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

	results := FilterNaclEntriesByLogicalId("TestNacl", template, params, map[string]string{})

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
				Properties: map[string]any{
					"RouteTableId":         "REF: TestRouteTable",
					"DestinationCidrBlock": "0.0.0.0/0",
					"GatewayId":            map[string]any{"Ref": "GW"},
				},
			},
			"Route2": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         "REF: TestRouteTable",
					"DestinationCidrBlock": "10.0.0.0/8",
					"NatGatewayId":         "REF: MyNATGateway",
				},
			},
			"OtherRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
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

// TestFilterRoutesByLogicalId_NoDestination verifies that a route resource
// missing all destination properties is excluded from the result map, rather
// than being inserted with an empty-string key.
func TestFilterRoutesByLogicalId_NoDestination(t *testing.T) {
	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"RouteNoDestination": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId": "REF: TestRouteTable",
					"GatewayId":    "local",
				},
			},
			"RouteWithDestination": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         "REF: TestRouteTable",
					"DestinationCidrBlock": "10.0.0.0/8",
					"GatewayId":            "igw-123",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterRoutesByLogicalId(
		"TestRouteTable",
		template,
		[]cfntypes.Parameter{},
		map[string]string{},
	)

	// Only the route with a destination should be present
	if len(results) != 1 {
		t.Errorf("Expected 1 route, got %d", len(results))
	}
	if _, ok := results["10.0.0.0/8"]; !ok {
		t.Errorf("Expected route with destination 10.0.0.0/8 to be present")
	}
	// The route with no destination should not create an empty-string key
	if _, ok := results[""]; ok {
		t.Errorf("Route with no destination should not appear in results")
	}
}

// TestCfnTemplateTransform_Value ensures that the Value method returns the
// correct type based on which field is populated.
func TestCfnTemplateTransform_Value(t *testing.T) {
	tests := []struct {
		name      string
		transform CfnTemplateTransform
		want      any
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

	body, err := ParseTemplateString(template, nil)
	if err != nil {
		t.Fatalf("ParseTemplateString() error = %v", err)
	}
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

	if notificationARNs, ok := props["NotificationARNs"].([]any); !ok || len(notificationARNs) == 0 {
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

	body, err := ParseTemplateString(template, nil)
	if err != nil {
		t.Fatalf("ParseTemplateString() error = %v", err)
	}
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

// TestFilterTGWRoutesByLogicalId verifies filtering Transit Gateway routes from a template
// by logical Transit Gateway route table ID.
func TestFilterTGWRoutesByLogicalId(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("AttachParam"), ParameterValue: aws.String("tgw-attach-param")},
	}
	logicalToPhysical := map[string]string{
		"MyAttachment": "tgw-attach-logical",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"TGWRoute1": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "REF: TestTGWRouteTable",
					"DestinationCidrBlock":       "10.0.0.0/16",
					"TransitGatewayAttachmentId": "tgw-attach-12345678",
				},
			},
			"TGWRoute2": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "REF: TestTGWRouteTable",
					"DestinationPrefixListId":    "pl-12345678",
					"TransitGatewayAttachmentId": map[string]any{"Ref": "AttachParam"},
				},
			},
			"TGWRoute3": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "REF: TestTGWRouteTable",
					"DestinationCidrBlock":       "192.168.0.0/24",
					"Blackhole":                  true,
				},
			},
			"OtherTGWRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "REF: OtherTGWRouteTable",
					"DestinationCidrBlock":       "172.16.0.0/12",
					"TransitGatewayAttachmentId": "tgw-attach-other",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterTGWRoutesByLogicalId("TestTGWRouteTable", template, params, logicalToPhysical)

	if len(results) != 3 {
		t.Errorf("Expected 3 routes, got %d", len(results))
	}

	// Check that the CIDR route exists with correct properties
	if route, ok := results["10.0.0.0/16"]; ok {
		if *route.DestinationCidrBlock != "10.0.0.0/16" {
			t.Errorf("Expected CIDR 10.0.0.0/16, got %s", *route.DestinationCidrBlock)
		}
		if *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId != "tgw-attach-12345678" {
			t.Errorf("Expected attachment tgw-attach-12345678, got %s", *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId)
		}
		if route.State != types.TransitGatewayRouteStateActive {
			t.Errorf("Expected state active, got %s", route.State)
		}
	} else {
		t.Errorf("Expected CIDR route not found")
	}

	// Check that the prefix list route exists with parameter resolution
	if route, ok := results["pl-12345678"]; ok {
		if *route.PrefixListId != "pl-12345678" {
			t.Errorf("Expected prefix list pl-12345678, got %s", *route.PrefixListId)
		}
		if *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId != "tgw-attach-param" {
			t.Errorf("Expected attachment from parameter, got %s", *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId)
		}
	} else {
		t.Errorf("Expected prefix list route not found")
	}

	// Check that the blackhole route exists with correct properties
	if route, ok := results["192.168.0.0/24"]; ok {
		if *route.DestinationCidrBlock != "192.168.0.0/24" {
			t.Errorf("Expected CIDR 192.168.0.0/24, got %s", *route.DestinationCidrBlock)
		}
		if route.State != types.TransitGatewayRouteStateBlackhole {
			t.Errorf("Expected state blackhole, got %s", route.State)
		}
		if len(route.TransitGatewayAttachments) != 0 {
			t.Errorf("Expected no attachments for blackhole route, got %d", len(route.TransitGatewayAttachments))
		}
	} else {
		t.Errorf("Expected blackhole route not found")
	}

	// Ensure the other route table's route is not included
	if _, ok := results["172.16.0.0/12"]; ok {
		t.Errorf("Expected OtherTGWRouteTable route not to be included")
	}
}

// TestFilterTGWRoutesByLogicalId_RefAndImportMap verifies that FilterTGWRoutesByLogicalId
// handles route table ID properties specified as Ref maps and Fn::ImportValue maps,
// not just the "REF: " string format. This is a regression test for T-365.
func TestFilterTGWRoutesByLogicalId_RefAndImportMap(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"MyTGWRouteTable":     "tgw-rtb-physical123",
		"TGWRouteTableExport": "tgw-rtb-physical123",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			// Route using Ref map for route table ID
			"RefRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": map[string]any{"Ref": "MyTGWRouteTable"},
					"DestinationCidrBlock":       "10.0.0.0/16",
					"TransitGatewayAttachmentId": "tgw-attach-ref",
				},
			},
			// Route using Fn::ImportValue map for route table ID
			"ImportRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": map[string]any{"Fn::ImportValue": "TGWRouteTableExport"},
					"DestinationCidrBlock":       "172.16.0.0/12",
					"TransitGatewayAttachmentId": "tgw-attach-import",
				},
			},
			// Route using REF: string format (existing format, should still work)
			"RefStringRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "REF: MyTGWRouteTable",
					"DestinationCidrBlock":       "192.168.0.0/24",
					"TransitGatewayAttachmentId": "tgw-attach-refstr",
				},
			},
			// Route using plain physical ID string for route table ID
			"PhysicalIdRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "tgw-rtb-physical123",
					"DestinationCidrBlock":       "10.50.0.0/16",
					"TransitGatewayAttachmentId": "tgw-attach-physical",
				},
			},
			// Route for a different route table (should not be included)
			"OtherRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": map[string]any{"Ref": "OtherRouteTable"},
					"DestinationCidrBlock":       "10.99.0.0/16",
					"TransitGatewayAttachmentId": "tgw-attach-other",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterTGWRoutesByLogicalId("MyTGWRouteTable", template, params, logicalToPhysical)

	if len(results) != 4 {
		t.Errorf("Expected 4 routes, got %d", len(results))
	}

	// Check that the Ref map route is included
	if route, ok := results["10.0.0.0/16"]; ok {
		if *route.DestinationCidrBlock != "10.0.0.0/16" {
			t.Errorf("Expected CIDR 10.0.0.0/16, got %s", *route.DestinationCidrBlock)
		}
	} else {
		t.Error("Expected Ref map route (10.0.0.0/16) not found")
	}

	// Check that the ImportValue map route is included
	if route, ok := results["172.16.0.0/12"]; ok {
		if *route.DestinationCidrBlock != "172.16.0.0/12" {
			t.Errorf("Expected CIDR 172.16.0.0/12, got %s", *route.DestinationCidrBlock)
		}
	} else {
		t.Error("Expected ImportValue map route (172.16.0.0/12) not found")
	}

	// Check that the REF: string format route is still included
	if route, ok := results["192.168.0.0/24"]; ok {
		if *route.DestinationCidrBlock != "192.168.0.0/24" {
			t.Errorf("Expected CIDR 192.168.0.0/24, got %s", *route.DestinationCidrBlock)
		}
	} else {
		t.Error("Expected REF: string route (192.168.0.0/24) not found")
	}

	// Check that the plain physical ID string route is included
	if route, ok := results["10.50.0.0/16"]; ok {
		if *route.DestinationCidrBlock != "10.50.0.0/16" {
			t.Errorf("Expected CIDR 10.50.0.0/16, got %s", *route.DestinationCidrBlock)
		}
	} else {
		t.Error("Expected plain physical ID route (10.50.0.0/16) not found")
	}

	// Ensure the other route table's route is NOT included
	if _, ok := results["10.99.0.0/16"]; ok {
		t.Error("Expected other route table's route not to be included")
	}
}

// TestCfnTemplateParameter_UnmarshalJSON verifies that parameter constraint
// fields can be unmarshaled from both numeric and string JSON values
func TestCfnTemplateParameter_UnmarshalJSON(t *testing.T) {
	tests := map[string]struct {
		json string
		want CfnTemplateParameter
	}{
		"numeric constraints": {
			json: `{
				"Type": "String",
				"MaxLength": 100,
				"MinLength": 1,
				"MaxValue": 99.5,
				"MinValue": 0.5
			}`,
			want: CfnTemplateParameter{
				Type:      "String",
				MaxLength: 100,
				MinLength: 1,
				MaxValue:  99.5,
				MinValue:  0.5,
			},
		},
		"string constraints": {
			json: `{
				"Type": "String",
				"MaxLength": "100",
				"MinLength": "1",
				"MaxValue": "99.5",
				"MinValue": "0.5"
			}`,
			want: CfnTemplateParameter{
				Type:      "String",
				MaxLength: 100,
				MinLength: 1,
				MaxValue:  99.5,
				MinValue:  0.5,
			},
		},
		"mixed numeric and string constraints": {
			json: `{
				"Type": "String",
				"MaxLength": "100",
				"MinLength": 1,
				"MaxValue": 99.5,
				"MinValue": "0.5"
			}`,
			want: CfnTemplateParameter{
				Type:      "String",
				MaxLength: 100,
				MinLength: 1,
				MaxValue:  99.5,
				MinValue:  0.5,
			},
		},
		"parameter with default and allowed values": {
			json: `{
				"Type": "String",
				"Description": "Test parameter",
				"Default": "test-value",
				"AllowedPattern": "^[a-z-]+$",
				"AllowedValues": ["test-value", "other-value"],
				"ConstraintDescription": "Must be lowercase",
				"MaxLength": "50",
				"NoEcho": false
			}`,
			want: CfnTemplateParameter{
				Type:                  "String",
				Description:           "Test parameter",
				Default:               "test-value",
				AllowedPattern:        "^[a-z-]+$",
				AllowedValues:         []any{"test-value", "other-value"},
				ConstraintDescription: "Must be lowercase",
				MaxLength:             50,
				NoEcho:                false,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var got CfnTemplateParameter
			err := got.UnmarshalJSON([]byte(tc.json))
			if err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}

			if got.Type != tc.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tc.want.Type)
			}
			if got.MaxLength != tc.want.MaxLength {
				t.Errorf("MaxLength = %v, want %v", got.MaxLength, tc.want.MaxLength)
			}
			if got.MinLength != tc.want.MinLength {
				t.Errorf("MinLength = %v, want %v", got.MinLength, tc.want.MinLength)
			}
			if got.MaxValue != tc.want.MaxValue {
				t.Errorf("MaxValue = %v, want %v", got.MaxValue, tc.want.MaxValue)
			}
			if got.MinValue != tc.want.MinValue {
				t.Errorf("MinValue = %v, want %v", got.MinValue, tc.want.MinValue)
			}
			if got.Description != tc.want.Description {
				t.Errorf("Description = %v, want %v", got.Description, tc.want.Description)
			}
			if got.AllowedPattern != tc.want.AllowedPattern {
				t.Errorf("AllowedPattern = %v, want %v", got.AllowedPattern, tc.want.AllowedPattern)
			}
		})
	}
}

// TestParseTemplateString_AdditionalStructures verifies parsing of templates with
// various additional structures including outputs, metadata, and conditions
func TestParseTemplateString_AdditionalStructures(t *testing.T) {
	tests := map[string]struct {
		template string
		verify   func(*testing.T, CfnTemplateBody)
	}{
		"template with outputs": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Outputs": {
					"BucketName": {
						"Value": "my-bucket",
						"Description": "The bucket name",
						"Export": {"Name": "MyBucketName"}
					}
				}
			}`,
			verify: func(t *testing.T, body CfnTemplateBody) {
				if len(body.Outputs) != 1 {
					t.Errorf("Expected 1 output, got %d", len(body.Outputs))
				}
				if output, ok := body.Outputs["BucketName"]; ok {
					if output.Value != "my-bucket" {
						t.Errorf("Output value = %v, want my-bucket", output.Value)
					}
				} else {
					t.Errorf("Expected output BucketName not found")
				}
			},
		},
		"template with metadata": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Metadata": {
					"AWS::CloudFormation::Interface": {
						"ParameterGroups": [{"Label": "Network", "Parameters": ["VpcId"]}]
					}
				}
			}`,
			verify: func(t *testing.T, body CfnTemplateBody) {
				if len(body.Metadata) == 0 {
					t.Errorf("Expected metadata, got none")
				}
			},
		},
		"template with conditions": {
			template: `{
				"AWSTemplateFormatVersion": "2010-09-09",
				"Conditions": {
					"CreateResource": true,
					"SkipResource": false
				}
			}`,
			verify: func(t *testing.T, body CfnTemplateBody) {
				if len(body.Conditions) != 2 {
					t.Errorf("Expected 2 conditions, got %d", len(body.Conditions))
				}
				if !body.Conditions["CreateResource"] {
					t.Errorf("CreateResource condition = false, want true")
				}
				if body.Conditions["SkipResource"] {
					t.Errorf("SkipResource condition = true, want false")
				}
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			body, err := ParseTemplateString(tc.template, nil)
			if err != nil {
				t.Fatalf("ParseTemplateString() error = %v", err)
			}
			tc.verify(t, body)
		})
	}
}

// TestRouteResourceToRoute_ImportValue verifies that Fn::ImportValue intrinsic
// functions are correctly resolved when converting Route resources
func TestRouteResourceToRoute_ImportValue(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"CENTRAL-TRANSIT-TransitGateway": "tgw-0f8df356de84e6b47",
	}

	// Template resource using Fn::ImportValue for TransitGatewayId
	resource := CfnTemplateResource{
		Type: "AWS::EC2::Route",
		Properties: map[string]any{
			"DestinationCidrBlock": "164.53.5.66/32",
			"TransitGatewayId": map[string]any{
				"Fn::ImportValue": "CENTRAL-TRANSIT-TransitGateway",
			},
		},
	}

	expected := types.Route{
		DestinationCidrBlock: aws.String("164.53.5.66/32"),
		TransitGatewayId:     aws.String("tgw-0f8df356de84e6b47"),
		Origin:               types.RouteOriginCreateRoute,
		State:                types.RouteStateActive,
	}

	got := RouteResourceToRoute(resource, params, logicalToPhysical)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("RouteResourceToRoute() with Fn::ImportValue:\ngot  = %#v\nwant = %#v", got, expected)
	}
}

// TestTGWRouteResourceToTGWRoute verifies conversion from a CloudFormation
// Transit Gateway route resource to a TransitGatewayRoute structure for destination extraction.
func TestTGWRouteResourceToTGWRoute(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("CIDRParam"), ParameterValue: aws.String("192.168.0.0/16")},
	}
	logicalToPhysical := map[string]string{}

	tests := []struct {
		name     string
		resource CfnTemplateResource
		want     types.TransitGatewayRoute
	}{
		{
			name: "CIDR block destination",
			resource: CfnTemplateResource{
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"DestinationCidrBlock": "10.0.0.0/16",
				},
			},
			want: types.TransitGatewayRoute{
				DestinationCidrBlock: aws.String("10.0.0.0/16"),
				State:                types.TransitGatewayRouteStateActive,
				Type:                 types.TransitGatewayRouteTypeStatic,
			},
		},
		{
			name: "Prefix list destination",
			resource: CfnTemplateResource{
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"DestinationPrefixListId": "pl-12345678",
				},
			},
			want: types.TransitGatewayRoute{
				PrefixListId: aws.String("pl-12345678"),
				State:        types.TransitGatewayRouteStateActive,
				Type:         types.TransitGatewayRouteTypeStatic,
			},
		},
		{
			name: "CIDR with parameter reference",
			resource: CfnTemplateResource{
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"DestinationCidrBlock": map[string]any{"Ref": "CIDRParam"},
				},
			},
			want: types.TransitGatewayRoute{
				DestinationCidrBlock: aws.String("192.168.0.0/16"),
				State:                types.TransitGatewayRouteStateActive,
				Type:                 types.TransitGatewayRouteTypeStatic,
			},
		},
		{
			name: "Missing properties returns route with defaults",
			resource: CfnTemplateResource{
				Type:       "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{},
			},
			want: types.TransitGatewayRoute{
				State: types.TransitGatewayRouteStateActive,
				Type:  types.TransitGatewayRouteTypeStatic,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TGWRouteResourceToTGWRoute(tt.resource, params, logicalToPhysical)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TGWRouteResourceToTGWRoute() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestParseTemplateString_EmptyAndWhitespace verifies that ParseTemplateString
// returns a clear error for empty or whitespace-only template bodies instead of panicking.
func TestParseTemplateString_EmptyAndWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  string
	}{
		{"empty string", "", "template body is empty"},
		{"whitespace only spaces", "   ", "template body is empty"},
		{"whitespace only tabs and newlines", "\t\n\t\n", "template body is empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseTemplateString(tt.template, nil)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestFilterNaclEntriesByLogicalId_FnImportValue verifies that NACL entries using
// Fn::ImportValue for NetworkAclId are correctly filtered instead of panicking.
// The Fn::ImportValue import name is resolved through logicalToPhysical to get
// the physical NACL ID, which is compared against the physical ID of the logicalId.
func TestFilterNaclEntriesByLogicalId_FnImportValue(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"SharedNaclExport": "acl-shared123",
		"MyNacl":           "acl-shared123",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"ImportedNaclEntry": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": map[string]any{
						"Fn::ImportValue": "SharedNaclExport",
					},
					"Protocol":   6.0,
					"RuleNumber": 100.0,
					"CidrBlock":  "10.0.0.0/8",
					"RuleAction": "allow",
					"Egress":     false,
				},
			},
			"StringNaclEntry": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": "REF: LocalNacl",
					"Protocol":     "17",
					"RuleNumber":   200.0,
					"CidrBlock":    "0.0.0.0/0",
					"RuleAction":   "deny",
					"Egress":       true,
				},
			},
		},
		Conditions: map[string]bool{},
	}

	// Should not panic when NetworkAclId is Fn::ImportValue map.
	// "MyNacl" resolves to "acl-shared123" via logicalToPhysical,
	// and "SharedNaclExport" also resolves to "acl-shared123", so the entry matches.
	results := FilterNaclEntriesByLogicalId("MyNacl", template, params, logicalToPhysical)

	if len(results) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(results))
	}

	if entry, ok := results["I100"]; ok {
		if *entry.RuleNumber != 100 {
			t.Errorf("Expected rule number 100, got %d", *entry.RuleNumber)
		}
	} else {
		t.Errorf("Expected ingress rule I100 not found")
	}

	// String-based entry should not be included
	if _, ok := results["E200"]; ok {
		t.Errorf("Expected LocalNacl entry not to be included")
	}
}

// TestFilterRoutesByLogicalId_FnImportValue verifies that routes using
// Fn::ImportValue for RouteTableId are correctly filtered instead of panicking.
func TestFilterRoutesByLogicalId_FnImportValue(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"SharedRouteTableExport": "rtb-shared456",
		"MyRouteTable":           "rtb-shared456",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"ImportedRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId": map[string]any{
						"Fn::ImportValue": "SharedRouteTableExport",
					},
					"DestinationCidrBlock": "0.0.0.0/0",
					"GatewayId":            "igw-abc123",
				},
			},
			"StringRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         "REF: LocalRouteTable",
					"DestinationCidrBlock": "10.0.0.0/8",
					"GatewayId":            "local",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	// Should not panic when RouteTableId is Fn::ImportValue map.
	// "MyRouteTable" resolves to "rtb-shared456" via logicalToPhysical,
	// and "SharedRouteTableExport" also resolves to "rtb-shared456", so the route matches.
	results := FilterRoutesByLogicalId("MyRouteTable", template, params, logicalToPhysical)

	if len(results) != 1 {
		t.Errorf("Expected 1 route, got %d", len(results))
	}

	if route, ok := results["0.0.0.0/0"]; ok {
		if *route.DestinationCidrBlock != "0.0.0.0/0" {
			t.Errorf("Expected destination 0.0.0.0/0, got %s", *route.DestinationCidrBlock)
		}
	} else {
		t.Errorf("Expected default route not found")
	}

	// String-based route should not be included
	if _, ok := results["10.0.0.0/8"]; ok {
		t.Errorf("Expected LocalRouteTable route not to be included")
	}
}

// TestFilterRoutesByLogicalId_RefMap verifies that routes using {"Ref": "LogicalId"}
// map format for RouteTableId are correctly matched. This is a regression test for
// T-741 where resourceIdMatchesLogical only handled "REF: " strings and
// Fn::ImportValue maps, missing the Ref-map format entirely.
func TestFilterRoutesByLogicalId_RefMap(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("GW"), ParameterValue: aws.String("igw-ref123")},
	}
	logicalToPhysical := map[string]string{
		"MyRouteTable": "rtb-abc123",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"RefRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         map[string]any{"Ref": "MyRouteTable"},
					"DestinationCidrBlock": "0.0.0.0/0",
					"GatewayId":            map[string]any{"Ref": "GW"},
				},
			},
			"StringRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         "REF: MyRouteTable",
					"DestinationCidrBlock": "10.0.0.0/8",
					"GatewayId":            "local",
				},
			},
			"OtherRefRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         map[string]any{"Ref": "OtherRouteTable"},
					"DestinationCidrBlock": "192.168.0.0/16",
					"GatewayId":            "local",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterRoutesByLogicalId("MyRouteTable", template, params, logicalToPhysical)

	// Both the Ref-map route and the string-based route should match
	if len(results) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(results))
	}

	// Ref-map route should be found
	if route, ok := results["0.0.0.0/0"]; ok {
		if *route.GatewayId != "igw-ref123" {
			t.Errorf("Expected gateway igw-ref123, got %s", *route.GatewayId)
		}
	} else {
		t.Errorf("Expected Ref-map default route not found")
	}

	// String-based route should also be found
	if _, ok := results["10.0.0.0/8"]; !ok {
		t.Errorf("Expected string-based route not found")
	}

	// OtherRouteTable route should NOT be included
	if _, ok := results["192.168.0.0/16"]; ok {
		t.Errorf("Expected OtherRouteTable route not to be included")
	}
}

// TestResourceIdMatchesLogical_RefMap verifies that resourceIdMatchesLogical
// handles the {"Ref": "LogicalId"} map format. Regression test for T-741.
func TestResourceIdMatchesLogical_RefMap(t *testing.T) {
	logicalToPhysical := map[string]string{
		"MyResource": "phys-123",
	}

	tests := []struct {
		name      string
		prop      any
		logicalId string
		want      bool
	}{
		{
			name:      "Ref map matches logical ID",
			prop:      map[string]any{"Ref": "MyResource"},
			logicalId: "MyResource",
			want:      true,
		},
		{
			name:      "Ref map does not match different logical ID",
			prop:      map[string]any{"Ref": "OtherResource"},
			logicalId: "MyResource",
			want:      false,
		},
		{
			name:      "string REF still works",
			prop:      "REF: MyResource",
			logicalId: "MyResource",
			want:      true,
		},
		{
			name:      "Fn::ImportValue still works",
			prop:      map[string]any{"Fn::ImportValue": "MyResource"},
			logicalId: "MyResource",
			want:      true,
		},
		{
			name:      "nil prop returns false",
			prop:      nil,
			logicalId: "MyResource",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceIdMatchesLogical(tt.prop, tt.logicalId, logicalToPhysical)
			if got != tt.want {
				t.Errorf("resourceIdMatchesLogical(%v, %q) = %v, want %v", tt.prop, tt.logicalId, got, tt.want)
			}
		})
	}
}

// TestFilterNaclEntriesByLogicalId_RefMap verifies that NACL entries using
// {"Ref": "LogicalId"} map format for NetworkAclId are correctly matched.
// This ensures the fix for T-741 also covers NACL filtering.
func TestFilterNaclEntriesByLogicalId_RefMap(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"NaclEntry1": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": map[string]any{"Ref": "MyNacl"},
					"RuleNumber":   float64(100),
					"Protocol":     "-1",
					"RuleAction":   "allow",
					"Egress":       false,
					"CidrBlock":    "10.0.0.0/8",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterNaclEntriesByLogicalId("MyNacl", template, params, logicalToPhysical)

	if len(results) != 1 {
		t.Errorf("Expected 1 NACL entry, got %d", len(results))
	}
}

// TestFilterNaclEntriesByLogicalId_ParameterizedRuleNumber is a regression
// test for T-834. Parameterized rule numbers must resolve to their real
// values so entries keep distinct drift-check keys. In the drift path,
// ParseTemplateString is invoked with parameter overrides, which typically
// inlines Ref values as numeric strings; the {"Ref": "Param"} map form can
// also survive when parameters are not supplied. Both shapes must work.
// Previously, parameterized entries collapsed onto the same "I0" or "E0"
// key, causing drift detection to report spurious modifications.
func TestFilterNaclEntriesByLogicalId_ParameterizedRuleNumber(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("IngressRuleNumber"), ParameterValue: aws.String("150")},
		{ParameterKey: aws.String("EgressRuleNumber"), ParameterValue: aws.String("250")},
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			// Inlined numeric string — the common drift-path shape after
			// ParseTemplateString applies parameter overrides.
			"IngressRule": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": "REF: TestNacl",
					"Protocol":     6.0,
					"RuleNumber":   "150",
					"CidrBlock":    "10.0.0.0/24",
					"RuleAction":   "allow",
					"Egress":       false,
				},
			},
			// Unresolved {"Ref": "Param"} map — covers paths where the Ref
			// map survives into extraction and must be resolved via params.
			"EgressRule": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": "REF: TestNacl",
					"Protocol":     "17",
					"RuleNumber":   map[string]any{"Ref": "EgressRuleNumber"},
					"CidrBlock":    "0.0.0.0/0",
					"RuleAction":   "deny",
					"Egress":       true,
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterNaclEntriesByLogicalId("TestNacl", template, params, map[string]string{})

	if len(results) != 2 {
		t.Fatalf("Expected 2 NACL entries, got %d (map=%v)", len(results), results)
	}

	entry, ok := results["I150"]
	if !ok {
		t.Fatalf("Expected ingress rule I150 in results, got keys %v", keysOf(results))
	}
	if entry.RuleNumber == nil || *entry.RuleNumber != 150 {
		t.Errorf("Expected ingress rule number 150, got %v", entry.RuleNumber)
	}

	entry, ok = results["E250"]
	if !ok {
		t.Fatalf("Expected egress rule E250 in results, got keys %v", keysOf(results))
	}
	if entry.RuleNumber == nil || *entry.RuleNumber != 250 {
		t.Errorf("Expected egress rule number 250, got %v", entry.RuleNumber)
	}
}

// keysOf returns the keys of a map for diagnostic messages.
func keysOf(m map[string]types.NetworkAclEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestFilterTGWRoutesByLogicalId_FnImportValue verifies that TGW routes using
// Fn::ImportValue for TransitGatewayRouteTableId are correctly filtered instead of panicking.
func TestFilterTGWRoutesByLogicalId_FnImportValue(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"SharedTGWRouteTableExport": "tgw-rtb-shared789",
		"MyTGWRouteTable":           "tgw-rtb-shared789",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"ImportedTGWRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": map[string]any{
						"Fn::ImportValue": "SharedTGWRouteTableExport",
					},
					"DestinationCidrBlock":       "10.0.0.0/16",
					"TransitGatewayAttachmentId": "tgw-attach-abc",
				},
			},
			"StringTGWRoute": {
				Type: "AWS::EC2::TransitGatewayRoute",
				Properties: map[string]any{
					"TransitGatewayRouteTableId": "REF: LocalTGWRouteTable",
					"DestinationCidrBlock":       "192.168.0.0/16",
					"TransitGatewayAttachmentId": "tgw-attach-other",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	// Should not panic when TransitGatewayRouteTableId is Fn::ImportValue map.
	// "MyTGWRouteTable" resolves to "tgw-rtb-shared789" via logicalToPhysical,
	// and "SharedTGWRouteTableExport" also resolves to the same value, so the route matches.
	results := FilterTGWRoutesByLogicalId("MyTGWRouteTable", template, params, logicalToPhysical)

	if len(results) != 1 {
		t.Errorf("Expected 1 route, got %d", len(results))
	}

	if route, ok := results["10.0.0.0/16"]; ok {
		if *route.DestinationCidrBlock != "10.0.0.0/16" {
			t.Errorf("Expected CIDR 10.0.0.0/16, got %s", *route.DestinationCidrBlock)
		}
	} else {
		t.Errorf("Expected CIDR route not found")
	}

	// String-based route should not be included
	if _, ok := results["192.168.0.0/16"]; ok {
		t.Errorf("Expected LocalTGWRouteTable route not to be included")
	}
}

// TestParseTemplateString_ValidTemplatesStillWork verifies that valid JSON and YAML
// templates continue to parse correctly after adding the empty-template guard.
func TestParseTemplateString_ValidTemplatesStillWork(t *testing.T) {
	jsonTemplate := `{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "Bucket": {
      "Type": "AWS::S3::Bucket"
    }
  }
}`
	yamlTemplate := `AWSTemplateFormatVersion: "2010-09-09"
Resources:
  Bucket:
    Type: AWS::S3::Bucket
`

	tests := []struct {
		name     string
		template string
	}{
		{"JSON template", jsonTemplate},
		{"YAML template", yamlTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body, err := ParseTemplateString(tt.template, nil)
			if err != nil {
				t.Fatalf("ParseTemplateString() unexpected error = %v", err)
			}
			if body.AWSTemplateFormatVersion != "2010-09-09" {
				t.Errorf("AWSTemplateFormatVersion = %q, want %q", body.AWSTemplateFormatVersion, "2010-09-09")
			}
			if _, ok := body.Resources["Bucket"]; !ok {
				t.Errorf("expected resource 'Bucket' not found")
			}
		})
	}
}

// TestResolveParameterValue_NilPointerKey verifies that resolveParameterValue
// does not panic when a parameter entry has a nil ParameterKey.
func TestResolveParameterValue_NilPointerKey(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: nil, ParameterValue: aws.String("value1")},
		{ParameterKey: aws.String("GoodKey"), ParameterValue: aws.String("good-value")},
	}

	// Should skip the nil-key parameter and still resolve a valid one
	got := resolveParameterValue("GoodKey", params)
	if got != "good-value" {
		t.Errorf("resolveParameterValue() = %q, want %q", got, "good-value")
	}

	// Should return empty for an unmatched ref without panicking
	got = resolveParameterValue("Missing", params)
	if got != "" {
		t.Errorf("resolveParameterValue() = %q, want %q", got, "")
	}
}

// TestResolveParameterValue_NilPointerValue verifies that resolveParameterValue
// does not panic when ParameterValue is nil and ResolvedValue is also nil.
func TestResolveParameterValue_NilPointerValue(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: aws.String("EmptyParam"), ParameterValue: nil, ResolvedValue: nil},
	}
	got := resolveParameterValue("EmptyParam", params)
	if got != "" {
		t.Errorf("resolveParameterValue() = %q, want %q", got, "")
	}
}

// TestStringPointer_NilParameterKeyAndValue verifies that stringPointer
// does not panic when parameter entries have nil ParameterKey or ParameterValue fields.
func TestStringPointer_NilParameterKeyAndValue(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: nil, ParameterValue: aws.String("should-skip")},
		{ParameterKey: aws.String("GateParam"), ParameterValue: nil, ResolvedValue: nil},
		{ParameterKey: aws.String("ValidParam"), ParameterValue: aws.String("resolved-val")},
	}
	logicalToPhysical := map[string]string{}

	tests := []struct {
		name  string
		props map[string]any
		want  *string
	}{
		{
			name: "Ref to param with nil key is skipped, valid param resolves",
			props: map[string]any{
				"GatewayId": map[string]any{"Ref": "ValidParam"},
			},
			want: aws.String("resolved-val"),
		},
		{
			name: "Ref to param with nil ParameterValue returns empty",
			props: map[string]any{
				"GatewayId": map[string]any{"Ref": "GateParam"},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringPointer(tt.props, params, logicalToPhysical, "GatewayId")
			if tt.want == nil {
				if got != nil {
					t.Errorf("stringPointer() = %v, want nil", *got)
				}
			} else if got == nil || *got != *tt.want {
				t.Errorf("stringPointer() = %v, want %v", got, *tt.want)
			}
		})
	}
}

// TestRouteResourceToRoute_NilParameterFields verifies that RouteResourceToRoute
// does not panic when parameters contain nil ParameterKey or ParameterValue.
func TestRouteResourceToRoute_NilParameterFields(t *testing.T) {
	params := []cfntypes.Parameter{
		{ParameterKey: nil, ParameterValue: aws.String("should-skip")},
		{ParameterKey: aws.String("GateParam"), ParameterValue: aws.String("igw-123")},
	}
	logical := map[string]string{}

	resource := CfnTemplateResource{
		Type: "AWS::EC2::Route",
		Properties: map[string]any{
			"DestinationCidrBlock": "10.0.0.0/8",
			"GatewayId":            map[string]any{"Ref": "GateParam"},
		},
	}

	got := RouteResourceToRoute(resource, params, logical)
	if got.GatewayId == nil || *got.GatewayId != "igw-123" {
		t.Errorf("RouteResourceToRoute().GatewayId = %v, want igw-123", got.GatewayId)
	}
}

// TestFilterRoutesByLogicalId_HardcodedPhysicalId verifies that routes whose
// RouteTableId property is a literal physical ID string are matched against
// the logical ID via the logicalToPhysical map.
func TestFilterRoutesByLogicalId_HardcodedPhysicalId(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"MyRouteTable": "rtb-physical123",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"PhysicalIdRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         "rtb-physical123",
					"DestinationCidrBlock": "10.50.0.0/16",
					"GatewayId":            "igw-abc123",
				},
			},
			"UnrelatedRoute": {
				Type: "AWS::EC2::Route",
				Properties: map[string]any{
					"RouteTableId":         "rtb-other999",
					"DestinationCidrBlock": "10.99.0.0/16",
					"GatewayId":            "igw-other",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterRoutesByLogicalId("MyRouteTable", template, params, logicalToPhysical)

	if len(results) != 1 {
		t.Fatalf("Expected 1 route, got %d", len(results))
	}
	if _, ok := results["10.50.0.0/16"]; !ok {
		t.Error("Expected hardcoded physical ID route (10.50.0.0/16) not found")
	}
	if _, ok := results["10.99.0.0/16"]; ok {
		t.Error("Unrelated physical ID route (10.99.0.0/16) should not be included")
	}
}

// TestFilterNaclEntriesByLogicalId_HardcodedPhysicalId verifies that NACL
// entries whose NetworkAclId property is a literal physical ID string are
// matched against the logical ID via the logicalToPhysical map.
func TestFilterNaclEntriesByLogicalId_HardcodedPhysicalId(t *testing.T) {
	params := []cfntypes.Parameter{}
	logicalToPhysical := map[string]string{
		"MyNacl": "acl-physical123",
	}

	template := CfnTemplateBody{
		Resources: map[string]CfnTemplateResource{
			"PhysicalIdEntry": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": "acl-physical123",
					"RuleNumber":   float64(100),
					"Protocol":     "-1",
					"RuleAction":   "allow",
					"Egress":       false,
					"CidrBlock":    "10.0.0.0/8",
				},
			},
			"UnrelatedEntry": {
				Type: "AWS::EC2::NetworkAclEntry",
				Properties: map[string]any{
					"NetworkAclId": "acl-other999",
					"RuleNumber":   float64(200),
					"Protocol":     "-1",
					"RuleAction":   "allow",
					"Egress":       false,
					"CidrBlock":    "10.1.0.0/16",
				},
			},
		},
		Conditions: map[string]bool{},
	}

	results := FilterNaclEntriesByLogicalId("MyNacl", template, params, logicalToPhysical)

	if len(results) != 1 {
		t.Fatalf("Expected 1 NACL entry, got %d", len(results))
	}
	if _, ok := results["I100"]; !ok {
		t.Error("Expected hardcoded physical ID NACL entry (I100) not found")
	}
	if _, ok := results["I200"]; ok {
		t.Error("Unrelated physical ID NACL entry (I200) should not be included")
	}
}

// TestResourceIdMatchesLogical_HardcodedPhysicalId verifies that
// resourceIdMatchesLogical matches a plain physical ID string against the
// physical ID of the logical resource via the logicalToPhysical map.
func TestResourceIdMatchesLogical_HardcodedPhysicalId(t *testing.T) {
	logicalToPhysical := map[string]string{
		"MyResource": "phys-12345",
	}

	tests := []struct {
		name      string
		prop      any
		logicalId string
		want      bool
	}{
		{
			name:      "plain physical ID string matches via logicalToPhysical",
			prop:      "phys-12345",
			logicalId: "MyResource",
			want:      true,
		},
		{
			name:      "REF:-prefixed physical ID still matches after trimming",
			prop:      "REF: phys-12345",
			logicalId: "MyResource",
			want:      true,
		},
		{
			name:      "plain physical ID that does not match logical ID's physical ID",
			prop:      "phys-other",
			logicalId: "MyResource",
			want:      false,
		},
		{
			name:      "plain physical ID when logical ID has no physical mapping",
			prop:      "phys-12345",
			logicalId: "UnmappedResource",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resourceIdMatchesLogical(tt.prop, tt.logicalId, logicalToPhysical)
			if got != tt.want {
				t.Errorf("resourceIdMatchesLogical(%v, %q) = %v, want %v", tt.prop, tt.logicalId, got, tt.want)
			}
		})
	}
}
