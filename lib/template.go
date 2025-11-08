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
	MaxLength             int     `json:"-"`
	MinLength             int     `json:"-"`
	MaxValue              float64 `json:"-"`
	MinValue              float64 `json:"-"`
	NoEcho                bool    `json:"NoEcho,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshalling for CfnTemplateParameter
func (p *CfnTemplateParameter) UnmarshalJSON(b []byte) error {
	// Create a temporary struct with all fields as any type
	type Alias CfnTemplateParameter
	aux := &struct {
		MaxLength any `json:"MaxLength,omitempty"`
		MinLength any `json:"MinLength,omitempty"`
		MaxValue  any `json:"MaxValue,omitempty"`
		MinValue  any `json:"MinValue,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}

	// Convert MaxLength from string or number to int
	if aux.MaxLength != nil {
		switch v := aux.MaxLength.(type) {
		case float64:
			p.MaxLength = int(v)
		case string:
			if val, err := strconv.Atoi(v); err == nil {
				p.MaxLength = val
			}
		}
	}

	// Convert MinLength from string or number to int
	if aux.MinLength != nil {
		switch v := aux.MinLength.(type) {
		case float64:
			p.MinLength = int(v)
		case string:
			if val, err := strconv.Atoi(v); err == nil {
				p.MinLength = val
			}
		}
	}

	// Convert MaxValue from string or number to float64
	if aux.MaxValue != nil {
		switch v := aux.MaxValue.(type) {
		case float64:
			p.MaxValue = v
		case string:
			if val, err := strconv.ParseFloat(v, 64); err == nil {
				p.MaxValue = val
			}
		}
	}

	// Convert MinValue from string or number to float64
	if aux.MinValue != nil {
		switch v := aux.MinValue.(type) {
		case float64:
			p.MinValue = v
		case string:
			if val, err := strconv.ParseFloat(v, 64); err == nil {
				p.MinValue = val
			}
		}
	}

	return nil
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

// customImportValueHandler handles Fn::ImportValue intrinsic functions by preserving them
// in a map format that can be processed later with the logicalToPhysical map
func customImportValueHandler(name string, input any, template any) any {
	// Return the import value in a format that stringPointer can recognize
	return map[string]any{
		"Fn::ImportValue": input,
	}
}

// ParseTemplateString parses a CloudFormation template string into a CfnTemplateBody
func ParseTemplateString(template string, parameters *map[string]any) CfnTemplateBody {
	parsedTemplate := CfnTemplateBody{}
	override := map[string]intrinsics.IntrinsicHandler{}
	override["Ref"] = customRefHandler
	override["Fn::ImportValue"] = customImportValueHandler
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
	protocol := extractProtocol(resource.Properties)
	rulenr := extractRuleNumber(resource.Properties)
	cidrblock := extractCidrBlock(resource.Properties, "CidrBlock", params)
	ipv6cidrblock := ""
	if cidrblock == "" {
		ipv6cidrblock = extractCidrBlock(resource.Properties, "Ipv6CidrBlock", params)
	}
	ruleaction := extractRuleAction(resource.Properties)
	egress := extractEgressFlag(resource.Properties)

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
	if portRange := extractPortRange(resource.Properties); portRange != nil {
		result.PortRange = portRange
	}
	if icmpTypeCode := extractIcmpTypeCode(resource.Properties); icmpTypeCode != nil {
		result.IcmpTypeCode = icmpTypeCode
	}

	return result
}

// extractRuleNumber safely extracts the rule number from NACL properties
func extractRuleNumber(properties map[string]any) int32 {
	if ruleNum, ok := properties["RuleNumber"].(float64); ok {
		return int32(ruleNum)
	}
	return 0
}

// extractEgressFlag safely extracts the egress flag from NACL properties
func extractEgressFlag(properties map[string]any) bool {
	if egress, ok := properties["Egress"].(bool); ok {
		return egress
	}
	return false
}

// extractProtocol extracts the protocol from NACL properties
func extractProtocol(properties map[string]any) string {
	switch value := properties["Protocol"].(type) {
	case string:
		return value
	case float64:
		return strconv.Itoa(int(value))
	default:
		return ""
	}
}

// extractCidrBlock extracts a CIDR block from properties, resolving parameter references
func extractCidrBlock(properties map[string]any, key string, params []cfntypes.Parameter) string {
	if properties[key] == nil {
		return ""
	}

	switch value := properties[key].(type) {
	case string:
		return value
	case map[string]any:
		if refname, ok := value["Ref"].(string); ok {
			return resolveParameterValue(refname, params)
		}
	}
	return ""
}

// resolveParameterValue resolves a parameter reference to its actual value
func resolveParameterValue(refname string, params []cfntypes.Parameter) string {
	for _, parameter := range params {
		if *parameter.ParameterKey == refname {
			if parameter.ResolvedValue != nil {
				return *parameter.ResolvedValue
			}
			if parameter.ParameterValue != nil {
				return *parameter.ParameterValue
			}
		}
	}
	return ""
}

// extractRuleAction extracts the rule action from NACL properties
func extractRuleAction(properties map[string]any) types.RuleAction {
	if ruleactionprop, ok := properties["RuleAction"].(string); ok {
		if ruleactionprop == string(types.RuleActionDeny) {
			return types.RuleActionDeny
		}
	}
	return types.RuleActionAllow
}

// extractPortRange extracts port range from NACL properties
func extractPortRange(properties map[string]any) *types.PortRange {
	if properties["PortRange"] == nil {
		return nil
	}

	ports := properties["PortRange"].(map[string]any)
	fromport := extractInt32Value(ports["From"])
	toport := extractInt32Value(ports["To"])

	return &types.PortRange{
		From: &fromport,
		To:   &toport,
	}
}

// extractIcmpTypeCode extracts ICMP type code from NACL properties
func extractIcmpTypeCode(properties map[string]any) *types.IcmpTypeCode {
	if properties["Icmp"] == nil {
		return nil
	}

	icmptypecodedata := properties["Icmp"].(map[string]any)
	icmpcode := extractInt32Value(icmptypecodedata["Code"])
	icmptype := extractInt32Value(icmptypecodedata["Type"])

	return &types.IcmpTypeCode{
		Code: &icmpcode,
		Type: &icmptype,
	}
}

// extractInt32Value extracts an int32 value from either a float64 or string
func extractInt32Value(value any) int32 {
	switch v := value.(type) {
	case float64:
		return int32(v)
	case string:
		if intVal, err := strconv.Atoi(v); err == nil {
			return int32(intVal)
		}
	}
	return 0
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
		// Handle Ref intrinsic function
		if refname, ok := value["Ref"].(string); ok {
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
		// Handle Fn::ImportValue intrinsic function
		// ImportValue returns the actual physical ID directly from the stack outputs
		if importValue, ok := value["Fn::ImportValue"].(string); ok {
			// The importValue is the export name, but we need the actual value
			// In the processed template, CloudFormation has already resolved this
			// So we look for it in logicalToPhysical map using the import name
			if physicalId, ok := logicalToPhysical[importValue]; ok {
				result = physicalId
			} else {
				// If not found, use the import value string itself as a fallback
				result = importValue
			}
		}
	}

	// Return nil if the result is empty string (property exists but couldn't be resolved)
	if result == "" {
		return nil
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
