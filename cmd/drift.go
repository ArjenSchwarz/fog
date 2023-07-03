/*
Copyright © 2023 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

var drift_StackName *string
var drift_resultsOnly *bool
var drift_separateProperties *bool

// driftCmd represents the drift command
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Better drift detection for a VPC",
	Long: `Enables drift detection and shows the results.

	Including checking certain values that aren't currently
	supported natively by CloudFormation drift detection.
	In particular it will show NACLs and Routes changes.`,
	Run: detectDrift,
}

func init() {
	rootCmd.AddCommand(driftCmd)
	drift_StackName = driftCmd.Flags().StringP("stackname", "n", "", "The name of the stack")
	drift_resultsOnly = driftCmd.Flags().BoolP("results-only", "r", false, "Don't trigger a new drift detection")
	drift_separateProperties = driftCmd.Flags().BoolP("separate-properties", "s", false, "Put every property on its own line")
}

func detectDrift(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	svc := awsConfig.CloudformationClient()
	resultTitle := "Drift results for stack " + *drift_StackName
	keys := []string{"LogicalId", "Type", "ChangeType", "Details"}
	outputsettings = settings.NewOutputSettings()
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "LogicalId"
	if !*drift_resultsOnly {
		driftid := lib.StartDriftDetection(drift_StackName, awsConfig.CloudformationClient())
		lib.WaitForDriftDetectionToFinish(driftid, awsConfig.CloudformationClient())
	}
	naclResources := make(map[string]string)
	routetableResources := make(map[string]string)
	logicalToPhysical := make(map[string]string)
	checkedResources := []string{}
	stack, err := lib.GetStack(drift_StackName, svc)
	if err != nil {
		panic(err)
	}
	for _, drift := range lib.GetDefaultStackDrift(drift_StackName, svc) {
		checkedResources = append(checkedResources, *drift.LogicalResourceId)
		logicalToPhysical[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		switch *drift.ResourceType {
		case "AWS::EC2::NetworkAcl":
			naclResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
			break
		case "AWS::EC2::RouteTable":
			routetableResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
			break
		}
		if drift.StackResourceDriftStatus == types.StackResourceDriftStatusInSync {
			continue
		}
		content := make(map[string]interface{})
		content["LogicalId"] = *drift.LogicalResourceId
		content["Type"] = *drift.ResourceType
		changetype := string(drift.StackResourceDriftStatus)
		if drift.StackResourceDriftStatus == types.StackResourceDriftStatusDeleted {
			changetype = outputsettings.StringWarningInline(changetype)
		}
		content["ChangeType"] = changetype
		expectedtags, actualtags := verifyTagOrder(drift.PropertyDifferences)
		properties := []string{}
		handledtags := []string{}
		for _, property := range drift.PropertyDifferences {
			pathsplit := strings.Split(*property.PropertyPath, "/")
			if stringInSlice("Tags", pathsplit) {
				tagprop, taghandled := tagDifferences(property, handledtags, expectedtags, actualtags, properties)
				if tagprop != "" {
					properties = append(properties, tagprop)
				}
				if taghandled != "" {
					handledtags = append(handledtags, taghandled)
				}
				continue
			}
			var expected, actual bytes.Buffer
			json.Indent(&expected, []byte(aws.ToString(property.ExpectedValue)), "", "  ")
			json.Indent(&actual, []byte(aws.ToString(property.ActualValue)), "", "  ")
			switch property.DifferenceType {
			case types.DifferenceTypeRemove:
				properties = append(properties, outputsettings.StringWarningInline(fmt.Sprintf("%s: %s - %s", property.DifferenceType, aws.ToString(property.PropertyPath), string(expected.Bytes()))))
				break
			case types.DifferenceTypeAdd:
				properties = append(properties, outputsettings.StringPositiveInline(fmt.Sprintf("%s: %s - %s", property.DifferenceType, aws.ToString(property.PropertyPath), string(actual.Bytes()))))
				break
			default:
				properties = append(properties, fmt.Sprintf("%s: %s - %s => %s", property.DifferenceType, aws.ToString(property.PropertyPath), aws.ToString(property.ExpectedValue), aws.ToString(property.ActualValue)))
			}
		}
		sort.Strings(properties)
		if *drift_separateProperties {
			for _, property := range properties {
				separateContent := make(map[string]interface{})
				for k, v := range content {
					separateContent[k] = v
				}
				separateContent["Details"] = property
				holder := format.OutputHolder{Contents: separateContent}
				output.AddHolder(holder)
			}
		} else {
			content["Details"] = properties
			holder := format.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
	}
	params := lib.GetParametersMap(stack.Parameters)
	template := lib.GetTemplateBody(drift_StackName, params, svc)
	checkNaclEntries(naclResources, template, stack.Parameters, &output, awsConfig)
	checkRouteTableRoutes(routetableResources, template, stack.Parameters, logicalToPhysical, &output, awsConfig)
	output.Write()
}

// checkNaclEntries verifies the NACL entries and if there are differences adds those to the provided output array
func checkNaclEntries(naclResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, output *format.OutputArray, awsConfig config.AWSConfig) {
	// Specific check for NACLs
	for logicalId, physicalId := range naclResources {
		rulechanges := []string{}
		nacl := lib.GetNacl(physicalId, awsConfig.EC2Client())
		attachedRules := lib.FilterNaclEntriesByLogicalId(logicalId, template, parameters)
		for _, entry := range nacl.Entries {
			rulenumberstring := "I"
			if *entry.Egress {
				rulenumberstring = "E"
			}
			rulenumberstring += strconv.Itoa(int(*entry.RuleNumber))
			cfnentry, ok := attachedRules[rulenumberstring]
			// If the key exists
			if ok {
				if !lib.CompareNaclEntries(entry, cfnentry) {
					ruledetails := fmt.Sprintf("Expected: %s%sActual: %s", naclEntryToString(cfnentry), outputsettings.GetSeparator(), naclEntryToString(entry))
					rulechanges = append(rulechanges, ruledetails)
				}
				delete(attachedRules, rulenumberstring)
			} else {
				// 32767 is the automatically generated deny all entry at the end of every NACL
				if rulenumberstring == "I32767" || rulenumberstring == "E32767" {
					continue
				}
				ruledetails := fmt.Sprintf("Unmanaged entry: %s", naclEntryToString(entry))
				rulechanges = append(rulechanges, outputsettings.StringPositiveInline(ruledetails))
			}
		}
		// Leftover rules only exist in CloudFormation
		for _, cfnentry := range attachedRules {
			ruledetails := fmt.Sprintf("Removed entry: %s", naclEntryToString(cfnentry))
			rulechanges = append(rulechanges, outputsettings.StringWarningInline(ruledetails))

		}
		if len(rulechanges) != 0 {
			if *drift_separateProperties {
				for _, change := range rulechanges {
					content := make(map[string]interface{})
					content["LogicalId"] = fmt.Sprintf("Entry for NACL %s", logicalId)
					content["Type"] = "AWS::EC2::NetworkACLEntry"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					holder := format.OutputHolder{Contents: content}
					output.AddHolder(holder)
				}
			} else {
				content := make(map[string]interface{})
				content["LogicalId"] = fmt.Sprintf("Entries for NACL %s", logicalId)
				content["Type"] = "AWS::EC2::NetworkACLEntry"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}
	}
}

// checkRouteTableRoutes verifies the routes and if there are differences adds those to the provided output array
func checkRouteTableRoutes(routetableResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, logicalToPhysical map[string]string, output *format.OutputArray, awsConfig config.AWSConfig) {
	// Specific check for NACLs
	for logicalId, physicalId := range routetableResources {
		rulechanges := []string{}
		routetable := lib.GetRouteTable(physicalId, awsConfig.EC2Client())
		attachedRules := lib.FilterRoutesByLogicalId(logicalId, template, parameters, logicalToPhysical)
		// fmt.Print(attachedRules)
		for _, route := range routetable.Routes {
			ruleid := lib.GetRouteDestination(route)
			// fmt.Printf("Route: %s - %s\n", ruleid, routeToString(route))
			if cfnroute, ok := attachedRules[ruleid]; ok {
				if !lib.CompareRoutes(route, cfnroute) {
					ruledetails := fmt.Sprintf("Expected: %s%sActual: %s", routeToString(cfnroute), outputsettings.GetSeparator(), routeToString(route))
					rulechanges = append(rulechanges, ruledetails)
				}
				delete(attachedRules, ruleid)
			} else {
				// If the route was created with the table, don't report it
				if route.Origin == ec2types.RouteOriginCreateRouteTable {
					continue
				}
				// If the route is for the S3 prefixlist or the dynamodb prefixlist, don't report it
				if route.DestinationPrefixListId != nil && (*route.DestinationPrefixListId == "pl-6ca54005" || *route.DestinationPrefixListId == "pl-62a5400b") {
					continue
				}
				ruledetails := fmt.Sprintf("Unmanaged route: %s", routeToString(route))
				rulechanges = append(rulechanges, outputsettings.StringPositiveInline(ruledetails))
			}
		}
		// Leftover rules only exist in CloudFormation
		for routeid, cfnroute := range attachedRules {
			// routeid being empty implies it wasn't created, likely due to a condition
			if routeid == "" {
				continue
			}
			ruledetails := fmt.Sprintf("Removed route: %s", routeToString(cfnroute))
			rulechanges = append(rulechanges, outputsettings.StringWarningInline(ruledetails))

		}
		if len(rulechanges) != 0 {
			if *drift_separateProperties {
				for _, change := range rulechanges {
					content := make(map[string]interface{})
					content["LogicalId"] = fmt.Sprintf("Route for RouteTable %s", logicalId)
					content["Type"] = "AWS::EC2::Route"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					holder := format.OutputHolder{Contents: content}
					output.AddHolder(holder)
				}
			} else {
				content := make(map[string]interface{})
				content["LogicalId"] = fmt.Sprintf("Routes for RouteTable %s", logicalId)
				content["Type"] = "AWS::EC2::Route"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}
	}
}

func verifyTagOrder(properties []types.PropertyDifference) (map[string]string, map[string]string) {
	type tagprop struct {
		ID       string
		Type     string
		Expected string
		Actual   string
	}
	var tags []tagprop
	for _, property := range properties {
		if strings.HasPrefix(aws.ToString(property.PropertyPath), "/Tags/") {
			pathsplit := strings.Split(*property.PropertyPath, "/")
			if len(pathsplit) == 4 {
				tags = append(tags, tagprop{ID: pathsplit[2], Type: pathsplit[3], Expected: *property.ExpectedValue, Actual: *property.ActualValue})
			}
		}
	}
	expectedmap := make(map[string]string)
	actualmap := make(map[string]string)
	for _, tagkeys := range tags {
		if tagkeys.Type == "Key" {
			for _, tagvalues := range tags {
				if tagkeys.ID == tagvalues.ID {
					expectedmap[tagkeys.Expected] = tagvalues.Expected
					actualmap[tagkeys.Actual] = tagvalues.Actual
				}
			}
		}
	}
	return expectedmap, actualmap
}

func tagDifferences(property types.PropertyDifference, handledtags []string, expectedtags map[string]string, actualtags map[string]string, properties []string) (string, string) {
	type tag struct {
		Key   string
		Value string
	}
	pathsplit := strings.Split(*property.PropertyPath, "/")
	var expected, actual bytes.Buffer
	json.Indent(&expected, []byte(aws.ToString(property.ExpectedValue)), "", "  ")
	json.Indent(&actual, []byte(aws.ToString(property.ActualValue)), "", "  ")
	switch property.DifferenceType {
	case types.DifferenceTypeRemove:
		tagstructs := []tag{}
		json.Unmarshal(expected.Bytes(), &tagstructs)
		for _, tagstruct := range tagstructs {
			return outputsettings.StringWarningInline(fmt.Sprintf("%s: %s - %s: %s", property.DifferenceType, pathsplit[1], tagstruct.Key, tagstruct.Value)), ""
		}
		return "", ""
	case types.DifferenceTypeAdd:
		tagstruct := tag{}
		json.Unmarshal(actual.Bytes(), &tagstruct)
		return outputsettings.StringPositiveInline(fmt.Sprintf("%s: %s - %s: %s", property.DifferenceType, pathsplit[1], tagstruct.Key, tagstruct.Value)), ""
	default:
		if pathsplit[3] == "Key" {
			if actualtags[*property.ExpectedValue] == expectedtags[*property.ExpectedValue] {
				return fmt.Sprintf("%s: Tag %s sequence change", property.DifferenceType, aws.ToString(property.ExpectedValue)), pathsplit[2]
			}
		} else if pathsplit[3] == "Value" && stringInSlice(pathsplit[2], handledtags) {
			return "", ""
		}
		return fmt.Sprintf("%s: %s - %s => %s", property.DifferenceType, aws.ToString(property.PropertyPath), aws.ToString(property.ExpectedValue), aws.ToString(property.ActualValue)), ""
	}
}

func naclEntryToString(entry ec2types.NetworkAclEntry) string {
	direction := "ingress"
	if *entry.Egress {
		direction = "egress"
	}
	ports := "Ports: All"
	if entry.PortRange != nil {
		if *entry.PortRange.From == *entry.PortRange.To {
			ports = fmt.Sprintf("Port: %v", *entry.PortRange.From)
		} else {
			ports = fmt.Sprintf("Ports: %v-%v", *entry.PortRange.From, *entry.PortRange.To)
		}
	}
	var cidr string
	if entry.CidrBlock != nil {
		cidr = *entry.CidrBlock
	}
	if entry.Ipv6CidrBlock != nil {
		cidr = *entry.Ipv6CidrBlock
	}
	return fmt.Sprintf("%s #%v %v: %s, %s %s", direction, *entry.RuleNumber, entry.RuleAction, *entry.Protocol, cidr, ports)
}

func routeToString(route ec2types.Route) string {
	destination := ""
	if route.DestinationCidrBlock != nil {
		destination = *route.DestinationCidrBlock
	} else if route.DestinationPrefixListId != nil {
		destination = *route.DestinationPrefixListId
	} else {
		destination = *route.DestinationIpv6CidrBlock
	}
	target := ""
	if route.CarrierGatewayId != nil {
		target = *route.CarrierGatewayId
	} else if route.CoreNetworkArn != nil {
		target = *route.CoreNetworkArn
	} else if route.EgressOnlyInternetGatewayId != nil {
		target = *route.EgressOnlyInternetGatewayId
	} else if route.GatewayId != nil {
		target = *route.GatewayId
	} else if route.InstanceId != nil {
		target = *route.InstanceId
		// InstanceOwnerId
	} else if route.LocalGatewayId != nil {
		target = *route.LocalGatewayId
	} else if route.NatGatewayId != nil {
		target = *route.NatGatewayId
	} else if route.NetworkInterfaceId != nil {
		target = *route.NetworkInterfaceId
	} else if route.TransitGatewayId != nil {
		target = *route.TransitGatewayId
	} else if route.VpcPeeringConnectionId != nil {
		target = *route.VpcPeeringConnectionId
	}
	status := ""
	if route.State == ec2types.RouteStateBlackhole {
		status = fmt.Sprintf(" (%s)", string(route.State))
	}
	return fmt.Sprintf("%s: %s %s", destination, target, status)
}
