package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/awslabs/goformation/v7/intrinsics"
)

// StackDeploymentFile represents a CloudFormation stack deployment configuration file
type StackDeploymentFile struct {
	TemplateFilePath string            `json:"template-file-path"`
	Parameters       map[string]string `json:"parameters"`
	Tags             map[string]string `json:"tags"`
}

// CfnTemplateBody represents the structure of a CloudFormation template
type CfnTemplateBody struct {
	AWSTemplateFormatVersion string                          `json:"AWSTemplateFormatVersion"`
	Description              string                          `json:"Description"`
	Metadata                 map[string]any                  `json:"Metadata"`
	Transform                *CfnTemplateTransform           `json:"Transform"`
	Mappings                 map[string]any                  `json:"Mappings"`
	Rules                    map[string]CfnTemplateRule      `json:"Rules"`
	Parameters               map[string]CfnTemplateParameter `json:"Parameters"`
	Resources                map[string]CfnTemplateResource  `json:"Resources"`
	Conditions               map[string]bool                 `json:"Conditions"`
	Outputs                  map[string]CfnTemplateOutput    `json:"Outputs"`
}

// CfnTemplateParameter represents a CloudFormation template parameter definition
type CfnTemplateParameter struct {
	Type                  string  `json:"Type"`
	Description           string  `json:"Description,omitempty"`
	Default               any     `json:"Default,omitempty"`
	AllowedPattern        string  `json:"AllowedPattern,omitempty"`
	AllowedValues         []any   `json:"AllowedValues,omitempty"`
	ConstraintDescription string  `json:"ConstraintDescription,omitempty"`
	MaxLength             int     `json:"MaxLength,omitempty"`
	MinLength             int     `json:"MinLength,omitempty"`
	MaxValue              float64 `json:"MaxValue,omitempty"`
	MinValue              float64 `json:"MinValue,omitempty"`
	NoEcho                bool    `json:"NoEcho,omitempty"`
}

// CfnTemplateResource represents a resource definition in a CloudFormation template
type CfnTemplateResource struct {
	Type       string         `json:"Type"`
	Condition  string         `json:"Condition"`
	Properties map[string]any `json:"Properties"`
	Metadata   map[string]any `json:"Metadata"`
}

// CfnTemplateCondition represents a condition in a CloudFormation template
type CfnTemplateCondition struct {
	Not    []any `json:"Fn::Not"`
	Equals []any `json:"Fn::Equals"`
}

// CfnTemplateOutput represents an output definition in a CloudFormation template
type CfnTemplateOutput struct {
	Value       string `json:"Value"`
	Description string `json:"Description"`
	Export      struct {
		Name string `json:"Name"`
	} `json:"Export"`
}

// CfnTemplateRule represents a rule definition in a CloudFormation template
type CfnTemplateRule struct {
	RuleCondition string                     `json:"Condition"`
	Assertions    []CfnTemplateRuleAssertion `json:"Assertions"`
}

// CfnTemplateRuleAssertion represents a rule assertion in a CloudFormation template
type CfnTemplateRuleAssertion struct {
	Assert            any    `json:"Assert"`
	AssertDescription string `json:"AssertDescription"`
}

// CfnTemplateTransform represents a transform in a CloudFormation template
type CfnTemplateTransform struct {
	String *string

	StringArray *[]string
}

// Value returns the underlying value of the transform (either a string or string array)
func (t CfnTemplateTransform) Value() any {
	if t.String != nil {
		return *t.String
	}

	if t.StringArray != nil {
		return *t.StringArray
	}

	return nil
}

// UnmarshalJSON implements custom JSON unmarshalling for CfnTemplateTransform
func (t *CfnTemplateTransform) UnmarshalJSON(b []byte) error {
	var typecheck any
	if err := json.Unmarshal(b, &typecheck); err != nil {
		return err
	}

	switch val := typecheck.(type) {

	case string:
		t.String = &val

	case []string:
		t.StringArray = &val

	case []any:
		var strslice []string
		for _, i := range val {
			if str, ok := i.(string); ok {
				strslice = append(strslice, str)
			}
		}
		t.StringArray = &strslice
	}

	return nil
}

// GetTemplateBody retrieves and parses a CloudFormation template from a stack
func GetTemplateBody(stackname *string, parameters *map[string]any, svc CloudFormationGetTemplateAPI) CfnTemplateBody {
	input := cloudformation.GetTemplateInput{
		StackName: stackname,
	}
	result, err := svc.GetTemplate(context.TODO(), &input)
	if err != nil {
		panic(err)
	}

	return ParseTemplateString(*result.TemplateBody, parameters)
}

// customRefHandler is a simple example of an intrinsic function handler function
// that refuses to resolve any intrinsic functions, and just returns a basic string.
func customRefHandler(name string, input any, template any) any {

	// Dang son, this has got more nest than a bald eagle
	// Check the input is a string
	if name, ok := input.(string); ok {

		switch name {

		case "AWS::AccountId":
			return "123456789012"
		case "AWS::NotificationARNs": //
			return []string{"arn:aws:sns:us-east-1:123456789012:MyTopic"}
		case "AWS::NoValue":
			return nil
		case "AWS::Region":
			return "ap-southeast-2"
		case "AWS::StackId":
			return "arn:aws:cloudformation:us-east-1:123456789012:stack/MyStack/1c2fa620-982a-11e3-aff7-50e2416294e0"
		case "AWS::StackName":
			return "YOUR_STACK_NAME"

		default:

			// This isn't a pseudo 'Ref' paramater, so we need to look inside the CloudFormation template
			// to see if we can resolve the reference. This implementation just looks at the Parameters section
			// to see if there is a parameter matching the name, and if so, return the default value.

			// Check the template is a map
			if template, ok := template.(map[string]any); ok {
				// Check there is a parameters section
				if uparameters, ok := template["Parameters"]; ok {
					// Check the parameters section is a map
					if parameters, ok := uparameters.(map[string]any); ok {
						// Check there is a parameter with the same name as the Ref
						if uparameter, ok := parameters[name]; ok {
							// Check the parameter is a map
							if parameter, ok := uparameter.(map[string]any); ok {
								// Check the parameter has a default
								if def, ok := parameter["Default"]; ok {
									return def
								}
							}
						}
					}
				}
			}
		}

	}
	return fmt.Sprintf("REF: %s", input)
}

// ParseTemplateString parses a CloudFormation template string into a CfnTemplateBody
func ParseTemplateString(template string, parameters *map[string]any) CfnTemplateBody {
	parsedTemplate := CfnTemplateBody{}
	override := map[string]intrinsics.IntrinsicHandler{}
	override["Ref"] = customRefHandler
	options := intrinsics.ProcessorOptions{
		IntrinsicHandlerOverrides: override,
	}
	if parameters != nil {
		options.ParameterOverrides = *parameters
	}
	var intrinsified []byte
	var err error
	// Use goformation intrinsics to convert to JSON and deal with intrinsics
	if template[0] == '{' {
		intrinsified, err = intrinsics.ProcessJSON([]byte(template), &options)
	} else {
		intrinsified, err = intrinsics.ProcessYAML([]byte(template), &options)
	}
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(intrinsified, &parsedTemplate); err != nil {
		panic(err)
	}
	return parsedTemplate
}

// FilterNaclEntriesByLogicalId filters Network ACL entries from a template by logical ID
func FilterNaclEntriesByLogicalId(logicalId string, template CfnTemplateBody, params []cfntypes.Parameter) map[string]types.NetworkAclEntry {
	result := make(map[string]types.NetworkAclEntry)
	for _, resource := range template.Resources {
		if resource.Type == "AWS::EC2::NetworkAclEntry" && template.ShouldHaveResource(resource) {
			aclid := strings.Replace(resource.Properties["NetworkAclId"].(string), "REF: ", "", 1)
			convresource := NaclResourceToNaclEntry(resource, params)
			if aclid == logicalId {
				rulenumberstring := "I"
				if *convresource.Egress {
					rulenumberstring = "E"
				}
				rulenumberstring += strconv.Itoa(int(*convresource.RuleNumber))
				result[rulenumberstring] = convresource
			}
		}
	}
	return result
}

// FilterRoutesByLogicalId filters routes from a template by logical ID
func FilterRoutesByLogicalId(logicalId string, template CfnTemplateBody, params []cfntypes.Parameter, logicalToPhysical map[string]string) map[string]types.Route {
	result := make(map[string]types.Route)
	for _, resource := range template.Resources {
		if resource.Type == "AWS::EC2::Route" && template.ShouldHaveResource(resource) {
			rtid := strings.Replace(resource.Properties["RouteTableId"].(string), "REF: ", "", 1)
			convresource := RouteResourceToRoute(resource, params, logicalToPhysical)
			if rtid == logicalId {
				result[GetRouteDestination(convresource)] = convresource
			}
		}
	}
	return result
}

// NaclResourceToNaclEntry converts a CloudFormation NACL resource to an EC2 NetworkAclEntry
func NaclResourceToNaclEntry(resource CfnTemplateResource, params []cfntypes.Parameter) types.NetworkAclEntry {
	protocol := ""
	switch value := resource.Properties["Protocol"].(type) {
	case string:
		protocol = value
		// break statement removed as it's redundant at the end of a case
	case float64:
		protocol = strconv.Itoa(int(value))
	}
	rulenr := int32(resource.Properties["RuleNumber"].(float64))
	cidrblock := ""
	switch value := resource.Properties["CidrBlock"].(type) {
	case string:
		cidrblock = value
	case map[string]any:
		refname := value["Ref"].(string)
		for _, parameter := range params {
			if *parameter.ParameterKey == refname {
				if parameter.ResolvedValue != nil {
					cidrblock = *parameter.ResolvedValue
				} else {
					cidrblock = *parameter.ParameterValue
				}
			}
		}
	}
	ipv6cidrblock := ""
	if cidrblock == "" {
		switch value := resource.Properties["Ipv6CidrBlock"].(type) {
		case string:
			ipv6cidrblock = value
		case map[string]any:
			refname := value["Ref"].(string)
			for _, parameter := range params {
				if *parameter.ParameterKey == refname {
					if parameter.ResolvedValue != nil {
						ipv6cidrblock = *parameter.ResolvedValue
					} else {
						ipv6cidrblock = *parameter.ParameterValue
					}
				}
			}
		}
	}
	ruleaction := types.RuleActionAllow
	ruleactionprop := resource.Properties["RuleAction"].(string)
	if ruleactionprop == string(types.RuleActionDeny) {
		ruleaction = types.RuleActionDeny
	}
	egress := resource.Properties["Egress"].(bool)
	result := types.NetworkAclEntry{
		Egress:     &egress,
		Protocol:   &protocol,
		RuleAction: ruleaction,
		RuleNumber: &rulenr,
	}
	if cidrblock != "" {
		result.CidrBlock = &cidrblock
	}
	if ipv6cidrblock != "" {
		result.Ipv6CidrBlock = &ipv6cidrblock
	}
	if resource.Properties["PortRange"] != nil {
		ports := resource.Properties["PortRange"].(map[string]any)
		var fromport, toport int32
		switch value := ports["From"].(type) {
		case float64:
			fromport = int32(value)
		case string:
			fromporta, _ := strconv.Atoi(value)
			fromport = int32(fromporta)
		}
		switch value := ports["To"].(type) {
		case float64:
			toport = int32(value)
		case string:
			toporta, _ := strconv.Atoi(value)
			toport = int32(toporta)
		}
		portrange := types.PortRange{
			From: &fromport,
			To:   &toport,
		}
		result.PortRange = &portrange
	}
	// In CloudFormation the IcmpTypeCode is just called Icmp
	if resource.Properties["Icmp"] != nil {
		icmptypecodedata := resource.Properties["Icmp"].(map[string]any)
		var icmptype, icmpcode int32
		switch value := icmptypecodedata["Code"].(type) {
		case float64:
			icmpcode = int32(value)
		case string:
			icmpcodea, _ := strconv.Atoi(value)
			icmpcode = int32(icmpcodea)
		}
		switch value := icmptypecodedata["Type"].(type) {
		case float64:
			icmptype = int32(value)
		case string:
			icmptypea, _ := strconv.Atoi(value)
			icmptype = int32(icmptypea)
		}
		icmptypecode := types.IcmpTypeCode{
			Code: &icmpcode,
			Type: &icmptype,
		}
		result.IcmpTypeCode = &icmptypecode
	}
	return result
}

// RouteResourceToRoute converts a CloudFormation route resource to an EC2 Route
func RouteResourceToRoute(resource CfnTemplateResource, params []cfntypes.Parameter, logicalToPhysical map[string]string) types.Route {
	prop := resource.Properties
	result := types.Route{
		CarrierGatewayId:            stringPointer(prop, params, logicalToPhysical, "CarrierGatewayId"),
		CoreNetworkArn:              stringPointer(prop, params, logicalToPhysical, "CoreNetworkArn"),
		DestinationCidrBlock:        stringPointer(prop, params, logicalToPhysical, "DestinationCidrBlock"),
		DestinationIpv6CidrBlock:    stringPointer(prop, params, logicalToPhysical, "DestinationIpv6CidrBlock"),
		DestinationPrefixListId:     stringPointer(prop, params, logicalToPhysical, "DestinationPrefixListId"),
		EgressOnlyInternetGatewayId: stringPointer(prop, params, logicalToPhysical, "EgressOnlyInternetGatewayId"),
		GatewayId:                   stringPointer(prop, params, logicalToPhysical, "GatewayId"),
		InstanceId:                  stringPointer(prop, params, logicalToPhysical, "InstanceId"),
		InstanceOwnerId:             stringPointer(prop, params, logicalToPhysical, "InstanceOwnerId"),
		LocalGatewayId:              stringPointer(prop, params, logicalToPhysical, "LocalGatewayId"),
		NatGatewayId:                stringPointer(prop, params, logicalToPhysical, "NatGatewayId"),
		NetworkInterfaceId:          stringPointer(prop, params, logicalToPhysical, "NetworkInterfaceId"),
		Origin:                      types.RouteOriginCreateRoute, // Always expect it to be created
		State:                       types.RouteStateActive,       // Always expect it to be active
		TransitGatewayId:            stringPointer(prop, params, logicalToPhysical, "TransitGatewayId"),
		VpcPeeringConnectionId:      stringPointer(prop, params, logicalToPhysical, "VpcPeeringConnectionId"),
	}
	return result
}

func stringPointer(array map[string]any, params []cfntypes.Parameter, logicalToPhysical map[string]string, value string) *string {
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
		// break statement removed as it's redundant at the end of a case
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

// ShouldHaveResource checks if a resource should exist based on its condition
func (body *CfnTemplateBody) ShouldHaveResource(resource CfnTemplateResource) bool {
	if resource.Condition != "" {
		return body.Conditions[resource.Condition]
	}
	return true
}

// func getParsedValueOfCondition(input interface{}, params []cfntypes.Parameter) string {
// 	result := ""
// 	switch value := input.(type) {
// 	case string:
// 		result = value
// 		// break statement removed as it's redundant at the end of a case
// 	case map[string]interface{}:
// 		refname := value["Ref"].(string)
// 		for _, parameter := range params {
// 			if *parameter.ParameterKey == refname {
// 				if parameter.ResolvedValue != nil {
// 					result = *parameter.ResolvedValue
// 				} else {
// 					result = *parameter.ParameterValue
// 				}
// 			}
// 		}
// 	}
// 	return result
// }
