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
	"log"
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

var driftFlags DriftFlags

// driftCmd represents the drift command
var driftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Better drift detection for a VPC",
	Long: `Enables drift detection and shows the results.

Including checking certain values that aren't currently
supported natively by CloudFormation drift detection.
In particular it will show NACLs and Routes changes.

Due to limitations in CloudFormation, prefix lists in routes don't
show up by default as they can't be managed using CloudFormation and
therefore we can't see if they've drifted.
If you wish these to be shown, you can use the --verbose flag. This
will still exclude AWS managed prefix lists, as these are automatically
assigned.`,
	Run: detectDrift,
}

func init() {
	stackCmd.AddCommand(driftCmd)
	driftFlags.RegisterFlags(driftCmd)
}

func detectDrift(cmd *cobra.Command, args []string) {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	svc := awsConfig.CloudformationClient()
	resultTitle := "Drift results for stack " + driftFlags.StackName
	keys := []string{"LogicalId", "Type", "ChangeType", "Details"}
	outputsettings = settings.NewOutputSettings()
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = resultTitle
	output.Settings.SortKey = "LogicalId"
	if !driftFlags.ResultsOnly {
		driftid := lib.StartDriftDetection(&driftFlags.StackName, awsConfig.CloudformationClient())
		lib.WaitForDriftDetectionToFinish(driftid, awsConfig.CloudformationClient())
	}
	defaultDrift := lib.GetDefaultStackDrift(&driftFlags.StackName, svc)
	naclResources, routetableResources, logicalToPhysical := separateSpecialCases(defaultDrift)
	checkedResources := []string{}
	stack, err := lib.GetStack(&driftFlags.StackName, svc)
	if err != nil {
		failWithError(err)
	}

	for _, drift := range defaultDrift {
		//TODO: verify if checkedResources is needed
		// Store the result of append
		checkedResources = append(checkedResources, *drift.LogicalResourceId)
		// Use checkedResources to avoid the SA4010 warning
		_ = checkedResources
		if drift.StackResourceDriftStatus == types.StackResourceDriftStatusInSync {
			continue
		}
		actualProperties := make(map[string]interface{})
		if drift.ActualProperties != nil {
			if err := json.Unmarshal([]byte(*drift.ActualProperties), &actualProperties); err != nil {
				failWithError(err)
			}
		}
		expectedProperties := make(map[string]interface{})
		if drift.ExpectedProperties != nil {
			if err := json.Unmarshal([]byte(*drift.ExpectedProperties), &expectedProperties); err != nil {
				failWithError(err)
			}
		}
		content := make(map[string]interface{})
		content["LogicalId"] = *drift.LogicalResourceId
		content["Type"] = *drift.ResourceType
		changetype := string(drift.StackResourceDriftStatus)
		if drift.StackResourceDriftStatus == types.StackResourceDriftStatusDeleted {
			changetype = outputsettings.StringWarningInline(changetype)
		}
		content["ChangeType"] = changetype
		tagMap := getExpectedAndActualTags(expectedProperties, actualProperties)

		properties := []string{}
		handledtags := []string{}

		for _, property := range drift.PropertyDifferences {
			pathsplit := strings.Split(*property.PropertyPath, "/")
			if stringInSlice("Tags", pathsplit) {
				tagprop, taghandled := tagDifferences(property, handledtags, tagMap, properties, &drift)
				if tagprop != "" {
					properties = append(properties, tagprop)
				}
				if taghandled != "" {
					handledtags = append(handledtags, taghandled)
				}
				continue
			}
			var expected, actual bytes.Buffer
			if err := json.Indent(&expected, []byte(aws.ToString(property.ExpectedValue)), "", "  "); err != nil {
				failWithError(err)
			}
			if err := json.Indent(&actual, []byte(aws.ToString(property.ActualValue)), "", "  "); err != nil {
				failWithError(err)
			}
			switch property.DifferenceType {
			case types.DifferenceTypeRemove:
				properties = append(properties, outputsettings.StringWarningInline(fmt.Sprintf("%s: %s - %s", property.DifferenceType, aws.ToString(property.PropertyPath), expected.String())))
			case types.DifferenceTypeAdd:
				properties = append(properties, outputsettings.StringPositiveInline(fmt.Sprintf("%s: %s - %s", property.DifferenceType, aws.ToString(property.PropertyPath), actual.String())))
			default:
				properties = append(properties, fmt.Sprintf("%s: %s - %s => %s", property.DifferenceType, aws.ToString(property.PropertyPath), aws.ToString(property.ExpectedValue), aws.ToString(property.ActualValue)))
			}
		}
		if len(properties) != 0 {
			sort.Strings(properties)
			if driftFlags.SeparateProperties {
				for _, property := range properties {
					separateContent := make(map[string]interface{})
					for k, v := range content {
						separateContent[k] = v
					}
					separateContent["Details"] = property
					output.AddContents(separateContent)
				}
			} else {
				content["Details"] = properties
				output.AddContents(content)
			}
		}
	}
	params := lib.GetParametersMap(stack.Parameters)
	template := lib.GetTemplateBody(&driftFlags.StackName, params, svc)
	checkNaclEntries(naclResources, template, stack.Parameters, &output, awsConfig)
	checkRouteTableRoutes(routetableResources, template, stack.Parameters, logicalToPhysical, &output, awsConfig)
	for _, resourcetype := range settings.GetStringSlice("drift.detect-unmanaged-resources") {
		allresources, err := lib.ListAllResources(resourcetype, awsConfig.CloudControlClient(), awsConfig.SSOAdminClient(), awsConfig.OrganizationsClient())
		if err != nil {
			log.Fatal(err)
		}
		checkIfResourcesAreManaged(allresources, logicalToPhysical, &output)
	}
	output.Write()
}

func separateSpecialCases(defaultDrift []types.StackResourceDrift) (map[string]string, map[string]string, map[string]string) {
	naclResources := make(map[string]string)
	routetableResources := make(map[string]string)
	logicalToPhysical := make(map[string]string)
	for _, drift := range defaultDrift {
		logicalToPhysical[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		switch *drift.ResourceType {
		case "AWS::EC2::NetworkAcl":
			naclResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		case "AWS::EC2::RouteTable":
			routetableResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		}
	}
	return naclResources, routetableResources, logicalToPhysical
}

func checkIfResourcesAreManaged(allresources map[string]string, logicalToPhysical map[string]string, output *format.OutputArray) {
	toIgnore := settings.GetStringSlice("drift.ignore-unmanaged-resources")
	for resource, resourcetype := range allresources {
		// If the resource isn't in the logicalToPhysical map, it's not managed by CloudFormation
		if !stringValueInMap(resource, logicalToPhysical) {
			// If the resource is in the ignore list, don't report it
			if stringInSlice(resource, toIgnore) {
				continue
			}
			content := make(map[string]interface{})
			content["LogicalId"] = resource
			content["Type"] = resourcetype
			content["ChangeType"] = "UNMANAGED"
			content["Details"] = fmt.Sprintf("Not managed by this CloudFormation stack")
			output.AddContents(content)
		}
	}
}

// checkNaclEntries verifies the NACL entries and if there are differences adds those to the provided output array
func checkNaclEntries(naclResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, output *format.OutputArray, awsConfig config.AWSConfig) {
	// Specific check for NACLs
	for logicalId, physicalId := range naclResources {
		rulechanges := []string{}
		nacl, err := lib.GetNacl(physicalId, awsConfig.EC2Client())
		if err != nil {
			failWithError(err)
		}
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
			if driftFlags.SeparateProperties {
				for _, change := range rulechanges {
					content := make(map[string]interface{})
					content["LogicalId"] = fmt.Sprintf("Entry for NACL %s", logicalId)
					content["Type"] = "AWS::EC2::NetworkACLEntry"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					output.AddContents(content)
				}
			} else {
				content := make(map[string]interface{})
				content["LogicalId"] = fmt.Sprintf("Entries for NACL %s", logicalId)
				content["Type"] = "AWS::EC2::NetworkACLEntry"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				output.AddContents(content)
			}
		}
	}
}

// checkRouteTableRoutes verifies the routes and if there are differences adds those to the provided output array
func checkRouteTableRoutes(routetableResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, logicalToPhysical map[string]string, output *format.OutputArray, awsConfig config.AWSConfig) {
	// Create a list of all AWS managed prefixes
	managedPrefixLists := lib.GetManagedPrefixLists(awsConfig.EC2Client())
	awsPrefixesSlice := make([]string, 0)
	for _, prefixlist := range managedPrefixLists {
		if *prefixlist.OwnerId == "AWS" {
			awsPrefixesSlice = append(awsPrefixesSlice, *prefixlist.PrefixListId)
		}
	}
	// Specific check for NACLs
	for logicalId, physicalId := range routetableResources {
		rulechanges := []string{}
		routetable, err := lib.GetRouteTable(physicalId, awsConfig.EC2Client())
		if err != nil {
			failWithError(err)
		}
		attachedRules := lib.FilterRoutesByLogicalId(logicalId, template, parameters, logicalToPhysical)
		for _, route := range routetable.Routes {
			ruleid := lib.GetRouteDestination(route)
			if route.DestinationPrefixListId != nil && (!settings.GetBool("verbose") || stringInSlice(*route.DestinationPrefixListId, awsPrefixesSlice)) {
				// If the route is for a prefixlist, don't report it by default as they're not defined in CloudFormation. Also don't report any AWS managed prefixlists
				continue
			}
			if cfnroute, ok := attachedRules[ruleid]; ok {
				if !lib.CompareRoutes(route, cfnroute, settings.GetStringSlice("drift.ignore-blackholes")) {
					ruledetails := fmt.Sprintf("Expected: %s%sActual: %s", routeToString(cfnroute), outputsettings.GetSeparator(), routeToString(route))
					rulechanges = append(rulechanges, ruledetails)
				}
				delete(attachedRules, ruleid)
			} else {
				// If the route was created with the table, don't report it
				if route.Origin == ec2types.RouteOriginCreateRouteTable {
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
			if driftFlags.SeparateProperties {
				for _, change := range rulechanges {
					content := make(map[string]interface{})
					content["LogicalId"] = fmt.Sprintf("Route for RouteTable %s", logicalId)
					content["Type"] = "AWS::EC2::Route"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					output.AddContents(content)
				}
			} else {
				content := make(map[string]interface{})
				content["LogicalId"] = fmt.Sprintf("Routes for RouteTable %s", logicalId)
				content["Type"] = "AWS::EC2::Route"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				output.AddContents(content)
			}
		}
	}
}

// func verifyTagOrder(properties []types.PropertyDifference) (map[string]string, map[string]string) {
// 	type tagprop struct {
// 		ID       string
// 		Type     string
// 		Expected string
// 		Actual   string
// 	}
// 	var tags []tagprop
// 	for _, property := range properties {
// 		if strings.HasPrefix(aws.ToString(property.PropertyPath), "/Tags/") {
// 			pathsplit := strings.Split(*property.PropertyPath, "/")
// 			if len(pathsplit) == 4 {
// 				tags = append(tags, tagprop{ID: pathsplit[2], Type: pathsplit[3], Expected: *property.ExpectedValue, Actual: *property.ActualValue})
// 			}
// 		}
// 	}
// 	expectedmap := make(map[string]string)
// 	actualmap := make(map[string]string)
// 	for _, tagkeys := range tags {
// 		if tagkeys.Type == "Key" {
// 			for _, tagvalues := range tags {
// 				if tagkeys.ID == tagvalues.ID {
// 					expectedmap[tagkeys.Expected] = tagvalues.Expected
// 					actualmap[tagkeys.Actual] = tagvalues.Actual
// 				}
// 			}
// 		}
// 	}
// 	return expectedmap, actualmap
// }

func getExpectedAndActualTags(expectedResources map[string]interface{}, actualResources map[string]interface{}) map[string]map[string]string {
	// if Tags exists in expectedResources, compare the list of tags with those in actualResources
	tags := make(map[string]map[string]string)
	// go through expectedResources["Tags"] and add each item in expectedTags
	if expectedResources["Tags"] != nil {
		for _, tag := range expectedResources["Tags"].([]interface{}) {
			tagMap := tag.(map[string]interface{})
			tags[tagMap["Key"].(string)] = map[string]string{"Expected": tagMap["Value"].(string)}
		}
	}
	// go through actualResources["Tags"] and add each item in actualTags
	if actualResources["Tags"] != nil {
		for _, tag := range actualResources["Tags"].([]interface{}) {
			tagMap := tag.(map[string]interface{})
			if tags[tagMap["Key"].(string)] == nil {
				tags[tagMap["Key"].(string)] = map[string]string{"Expected": "", "Actual": tagMap["Value"].(string)}
			}
			tags[tagMap["Key"].(string)]["Actual"] = tagMap["Value"].(string)
		}
	}
	return tags
}

func shouldTagBeHandled(tag string, drift types.StackResourceDrift) bool {
	ignoredSlice := strings.Split(driftFlags.IgnoreTags, ",")
	ignoredSlice = append(ignoredSlice, settings.GetStringSlice("drift.ignore-tags")...)
	// check high-level tags
	if stringInSlice(tag, ignoredSlice) {
		return false
	}
	// check tags per resource id and resource type
	for _, ignoredtag := range ignoredSlice {
		ignoredtag = strings.ReplaceAll(ignoredtag, "::", "YSPACERY")
		separate := strings.Split(ignoredtag, ":")
		if strings.Contains(ignoredtag, ":") {
			// it's a service
			if strings.Contains(ignoredtag, "YSPACERY") {
				service := strings.ReplaceAll(separate[0], "YSPACERY", "::")
				if service == *drift.ResourceType && strings.Join(separate[1:], ":") == tag {
					return false
				}
				// it's a logicalID
			} else {
				if separate[0] == *drift.LogicalResourceId && strings.Join(separate[1:], ":") == tag {
					return false
				}

			}

		}

	}
	return true
}

func tagDifferences(property types.PropertyDifference, handledtags []string, tagMap map[string]map[string]string, properties []string, drift *types.StackResourceDrift) (string, string) {
	type tag struct {
		Key   string
		Value string
	}
	pathsplit := strings.Split(*property.PropertyPath, "/")
	var expected, actual bytes.Buffer
	if err := json.Indent(&expected, []byte(aws.ToString(property.ExpectedValue)), "", "  "); err != nil {
		failWithError(err)
	}
	if err := json.Indent(&actual, []byte(aws.ToString(property.ActualValue)), "", "  "); err != nil {
		failWithError(err)
	}
	switch property.DifferenceType {
	case types.DifferenceTypeRemove:
		tagstructs := []tag{}
		if expected.String()[0] == '[' {
			if err := json.Unmarshal(expected.Bytes(), &tagstructs); err != nil {
				failWithError(err)
			}
		} else {
			tagstruct := tag{}
			if err := json.Unmarshal(expected.Bytes(), &tagstruct); err != nil {
				failWithError(err)
			}
			tagstructs = append(tagstructs, tagstruct)
		}
		for _, tagstruct := range tagstructs {
			return outputsettings.StringWarningInline(fmt.Sprintf("%s: %s - %s: %s", property.DifferenceType, pathsplit[1], tagstruct.Key, tagstruct.Value)), ""
		}
		return "", ""
	case types.DifferenceTypeAdd:
		tagstruct := tag{}
		if err := json.Unmarshal(actual.Bytes(), &tagstruct); err != nil {
			failWithError(err)
		}
		return outputsettings.StringPositiveInline(fmt.Sprintf("%s: %s - %s: %s", property.DifferenceType, pathsplit[1], tagstruct.Key, tagstruct.Value)), ""
	default:
		tagKey := ""
		tags := map[string]string{}
		//loop over the tags in the tagMap and see if the "Expected" value matches *property.ExpectedValue
		for key, values := range tagMap {
			if values["Expected"] == *property.ExpectedValue && values["Actual"] == *property.ActualValue {
				tagKey = key
				tags = values
			}
		}
		if !shouldTagBeHandled(tagKey, *drift) {
			return "", ""
		}
		if pathsplit[3] == "Key" {
			if tags["Expected"] == tags["Actual"] {
				return fmt.Sprintf("%s: Tag %s sequence change", property.DifferenceType, aws.ToString(property.ExpectedValue)), pathsplit[2]
			}
		} else if pathsplit[3] == "Value" && stringInSlice(pathsplit[2], handledtags) {
			return "", ""
		}
		return fmt.Sprintf("%s: %s - %s => %s", property.DifferenceType, tagKey, tags["Expected"], tags["Actual"]), ""
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
	if entry.IcmpTypeCode != nil {
		if *entry.IcmpTypeCode.Type == -1 {
			ports = "ICMP: All"
		} else {
			ports = fmt.Sprintf("ICMP: %v-%v", *entry.IcmpTypeCode.Type, *entry.IcmpTypeCode.Code)
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
	destination := lib.GetRouteDestination(route)
	target := lib.GetRouteTarget(route)
	status := ""
	if route.State == ec2types.RouteStateBlackhole {
		status = fmt.Sprintf(" (%s)", string(route.State))
	}
	return fmt.Sprintf("%s: %s %s", destination, target, status)
}
