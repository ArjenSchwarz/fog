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
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/spf13/cobra"
)

var driftFlags DriftFlags
var listAllResourcesFunc = lib.ListAllResources

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
	ctx := context.Background()
	awsConfig, err := config.DefaultAwsConfig(ctx, *settings)
	if err != nil {
		failWithError(err)
	}
	svc := awsConfig.CloudformationClient()
	resultTitle := "Drift results for stack " + driftFlags.StackName
	keys := []string{"LogicalId", "Type", "ChangeType", "Details"}

	if !driftFlags.ResultsOnly {
		driftid, err := lib.StartDriftDetection(ctx, &driftFlags.StackName, svc)
		if err != nil {
			failWithError(err)
		}
		status, err := lib.WaitForDriftDetectionToFinish(ctx, driftid, svc)
		if err != nil {
			failWithError(err)
		}
		if status != types.StackDriftDetectionStatusDetectionComplete {
			failWithError(fmt.Errorf("drift detection completed with status: %s", status))
		}
	}
	defaultDrift, err := lib.GetDefaultStackDrift(ctx, &driftFlags.StackName, svc)
	if err != nil {
		failWithError(err)
	}
	stack, err := lib.GetStack(ctx, &driftFlags.StackName, svc)
	if err != nil {
		failWithError(err)
	}
	naclResources, routetableResources, tgwRouteTableResources, logicalToPhysical, err := separateSpecialCases(ctx, defaultDrift, &driftFlags.StackName, svc)
	if err != nil {
		failWithError(err)
	}
	// Build rows incrementally
	rows := make([]map[string]any, 0)

	for _, drift := range defaultDrift {
		if drift.StackResourceDriftStatus == types.StackResourceDriftStatusInSync || !driftHasRequiredFields(drift) {
			continue
		}
		actualProperties := make(map[string]any)
		if drift.ActualProperties != nil {
			if err := json.Unmarshal([]byte(*drift.ActualProperties), &actualProperties); err != nil {
				failWithError(err)
			}
		}
		expectedProperties := make(map[string]any)
		if drift.ExpectedProperties != nil {
			if err := json.Unmarshal([]byte(*drift.ExpectedProperties), &expectedProperties); err != nil {
				failWithError(err)
			}
		}
		content := make(map[string]any)
		content["LogicalId"] = aws.ToString(drift.LogicalResourceId)
		content["Type"] = aws.ToString(drift.ResourceType)
		changetype := string(drift.StackResourceDriftStatus)
		if drift.StackResourceDriftStatus == types.StackResourceDriftStatusDeleted {
			changetype = output.StyleWarning(changetype)
		}
		content["ChangeType"] = changetype
		tagMap := getExpectedAndActualTags(expectedProperties, actualProperties)

		properties := []string{}
		handledtags := []string{}

		for _, property := range drift.PropertyDifferences {
			propertyPath := aws.ToString(property.PropertyPath)
			if propertyPath == "" {
				continue
			}
			pathsplit := strings.Split(propertyPath, "/")
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
			expectedStr := aws.ToString(property.ExpectedValue)
			if json.Valid([]byte(expectedStr)) {
				if err := json.Indent(&expected, []byte(expectedStr), "", "  "); err != nil {
					failWithError(err)
				}
			} else {
				expected.WriteString(expectedStr)
			}
			actualStr := aws.ToString(property.ActualValue)
			if json.Valid([]byte(actualStr)) {
				if err := json.Indent(&actual, []byte(actualStr), "", "  "); err != nil {
					failWithError(err)
				}
			} else {
				actual.WriteString(actualStr)
			}
			switch property.DifferenceType {
			case types.DifferenceTypeRemove:
				properties = append(properties, output.StyleWarning(fmt.Sprintf("%s: %s - %s", property.DifferenceType, aws.ToString(property.PropertyPath), expected.String())))
			case types.DifferenceTypeAdd:
				properties = append(properties, output.StylePositive(fmt.Sprintf("%s: %s - %s", property.DifferenceType, aws.ToString(property.PropertyPath), actual.String())))
			default:
				properties = append(properties, fmt.Sprintf("%s: %s - %s => %s", property.DifferenceType, aws.ToString(property.PropertyPath), aws.ToString(property.ExpectedValue), aws.ToString(property.ActualValue)))
			}
		}
		if len(properties) != 0 {
			sort.Strings(properties)
			if driftFlags.SeparateProperties {
				for _, property := range properties {
					separateContent := make(map[string]any)
					maps.Copy(separateContent, content)
					separateContent["Details"] = property
					rows = append(rows, separateContent)
				}
			} else {
				content["Details"] = properties
				rows = append(rows, content)
			}
		}
	}
	params := lib.GetParametersMap(stack.Parameters)
	template, err := lib.GetTemplateBody(ctx, &driftFlags.StackName, params, svc)
	if err != nil {
		failWithError(err)
	}
	checkNaclEntries(ctx, naclResources, template, stack.Parameters, logicalToPhysical, &rows, awsConfig)
	checkRouteTableRoutes(ctx, routetableResources, template, stack.Parameters, logicalToPhysical, &rows, awsConfig)
	checkTransitGatewayRouteTableRoutes(ctx, tgwRouteTableResources, template, stack.Parameters, logicalToPhysical, &rows, awsConfig)
	if err := detectUnmanagedResources(ctx, settings.GetStringSlice("drift.detect-unmanaged-resources"), logicalToPhysical, &rows, awsConfig); err != nil {
		failWithError(err)
	}

	// Create and render the document
	if len(rows) == 0 {
		doc := output.New().
			Table(
				resultTitle,
				[]map[string]any{
					{"Status": "No drift detected"},
				},
				output.WithKeys("Status"),
			).
			Build()
		if err := renderDocument(context.Background(), doc); err != nil {
			failWithError(err)
		}
	} else {
		doc := output.New().
			Table(
				resultTitle,
				rows,
				output.WithKeys(keys...),
			).
			Build()
		if err := renderDocument(context.Background(), doc); err != nil {
			failWithError(err)
		}
	}
}

func driftHasRequiredFields(drift types.StackResourceDrift) bool {
	return drift.LogicalResourceId != nil && drift.ResourceType != nil
}

func separateSpecialCases(ctx context.Context, defaultDrift []types.StackResourceDrift, stackName *string, svc interface {
	lib.CloudFormationDescribeStackResourcesAPI
	lib.CloudFormationListExportsAPI
}) (map[string]string, map[string]string, map[string]string, map[string]string, error) {
	naclResources := make(map[string]string)
	routetableResources := make(map[string]string)
	tgwRouteTableResources := make(map[string]string)
	logicalToPhysical := make(map[string]string)

	// Build logicalToPhysical map from ALL stack resources
	// This ensures attachments and other resources are available for template resolution
	stackResourcesResp, err := svc.DescribeStackResources(ctx, &cloudformation.DescribeStackResourcesInput{
		StackName: stackName,
	})
	if err != nil {
		return naclResources, routetableResources, tgwRouteTableResources, logicalToPhysical, fmt.Errorf("failed to describe stack resources for drift special cases: %w", err)
	}
	for _, resource := range stackResourcesResp.StackResources {
		if resource.LogicalResourceId == nil || resource.PhysicalResourceId == nil {
			continue
		}
		logicalToPhysical[*resource.LogicalResourceId] = *resource.PhysicalResourceId
	}

	// Add CloudFormation exports to the map to handle !ImportValue references
	// This allows template routes that use ImportValue to be properly resolved.
	// Paginate to ensure all exports are collected (API returns max 100 per page).
	exportsPaginator := cloudformation.NewListExportsPaginator(svc, &cloudformation.ListExportsInput{})
	for exportsPaginator.HasMorePages() {
		exportsResp, err := exportsPaginator.NextPage(ctx)
		if err != nil {
			// Non-fatal - just log and continue without remaining exports
			log.Printf("Warning: Could not list CloudFormation exports: %v", err)
			break
		}
		for _, export := range exportsResp.Exports {
			if export.Name != nil && export.Value != nil {
				logicalToPhysical[*export.Name] = *export.Value
			}
		}
	}

	// Identify special case resources from drift results
	for _, drift := range defaultDrift {
		if drift.ResourceType == nil || drift.LogicalResourceId == nil || drift.PhysicalResourceId == nil {
			continue
		}
		switch *drift.ResourceType {
		case "AWS::EC2::NetworkAcl":
			naclResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		case "AWS::EC2::RouteTable":
			routetableResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		case "AWS::EC2::TransitGatewayRouteTable":
			tgwRouteTableResources[*drift.LogicalResourceId] = *drift.PhysicalResourceId
		}
	}
	return naclResources, routetableResources, tgwRouteTableResources, logicalToPhysical, nil
}

func detectUnmanagedResources(ctx context.Context, resourceTypes []string, logicalToPhysical map[string]string, rows *[]map[string]any, awsConfig config.AWSConfig) error {
	for _, resourceType := range resourceTypes {
		allresources, err := listAllResourcesFunc(ctx, resourceType, awsConfig.CloudControlClient(), awsConfig.SSOAdminClient(), awsConfig.OrganizationsClient())
		if err != nil {
			return fmt.Errorf("failed to list unmanaged resources for %s: %w", resourceType, err)
		}
		checkIfResourcesAreManaged(allresources, logicalToPhysical, rows)
	}
	return nil
}

func checkIfResourcesAreManaged(allresources map[string]string, logicalToPhysical map[string]string, rows *[]map[string]any) {
	// Build a set of managed physical IDs for O(1) lookups.
	// logicalToPhysical maps logical IDs (keys) to physical IDs (values).
	// allresources maps physical resource identifiers (keys) to resource types (values).
	// We need to check if each physical ID from allresources exists among the
	// physical IDs (values) in logicalToPhysical.
	managedPhysicalIDs := make(map[string]struct{}, len(logicalToPhysical))
	for _, physicalID := range logicalToPhysical {
		managedPhysicalIDs[physicalID] = struct{}{}
	}

	toIgnore := settings.GetStringSlice("drift.ignore-unmanaged-resources")
	for resource, resourcetype := range allresources {
		// If the resource's physical ID isn't in the managed set, it's not managed by CloudFormation
		if _, managed := managedPhysicalIDs[resource]; !managed {
			// If the resource is in the ignore list, don't report it
			if stringInSlice(resource, toIgnore) {
				continue
			}
			content := make(map[string]any)
			content["LogicalId"] = resource
			content["Type"] = resourcetype
			content["ChangeType"] = "UNMANAGED"
			content["Details"] = "Not managed by this CloudFormation stack"
			*rows = append(*rows, content)
		}
	}
}

// checkNaclEntries verifies the NACL entries and if there are differences adds those to the provided rows slice
func checkNaclEntries(ctx context.Context, naclResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, logicalToPhysical map[string]string, rows *[]map[string]any, awsConfig config.AWSConfig) {
	// Specific check for NACLs
	for logicalId, physicalId := range naclResources {
		rulechanges := []string{}
		nacl, err := lib.GetNacl(ctx, physicalId, awsConfig.EC2Client())
		if err != nil {
			failWithError(err)
		}
		attachedRules := lib.FilterNaclEntriesByLogicalId(logicalId, template, parameters, logicalToPhysical)
		for _, entry := range nacl.Entries {
			rulenumberstring := naclEntryKey(entry)
			cfnentry, ok := attachedRules[rulenumberstring]
			// If the key exists
			if ok {
				if !lib.CompareNaclEntries(entry, cfnentry) {
					ruledetails := fmt.Sprintf("Expected: %s\nActual: %s", naclEntryToString(cfnentry), naclEntryToString(entry))
					rulechanges = append(rulechanges, ruledetails)
				}
				delete(attachedRules, rulenumberstring)
			} else {
				// 32767 is the automatically generated deny all entry at the end of every NACL
				if rulenumberstring == "I32767" || rulenumberstring == "E32767" {
					continue
				}
				ruledetails := fmt.Sprintf("Unmanaged entry: %s", naclEntryToString(entry))
				rulechanges = append(rulechanges, output.StylePositive(ruledetails))
			}
		}
		// Leftover rules only exist in CloudFormation
		for _, cfnentry := range attachedRules {
			ruledetails := fmt.Sprintf("Removed entry: %s", naclEntryToString(cfnentry))
			rulechanges = append(rulechanges, output.StyleWarning(ruledetails))

		}
		if len(rulechanges) != 0 {
			if driftFlags.SeparateProperties {
				for _, change := range rulechanges {
					content := make(map[string]any)
					content["LogicalId"] = fmt.Sprintf("Entry for NACL %s", logicalId)
					content["Type"] = "AWS::EC2::NetworkACLEntry"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					*rows = append(*rows, content)
				}
			} else {
				content := make(map[string]any)
				content["LogicalId"] = fmt.Sprintf("Entries for NACL %s", logicalId)
				content["Type"] = "AWS::EC2::NetworkACLEntry"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				*rows = append(*rows, content)
			}
		}
	}
}

// checkRouteTableRoutes verifies the routes and if there are differences adds those to the provided rows slice
func checkRouteTableRoutes(ctx context.Context, routetableResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, logicalToPhysical map[string]string, rows *[]map[string]any, awsConfig config.AWSConfig) {
	// Create a list of all AWS managed prefixes
	managedPrefixLists, err := lib.GetManagedPrefixLists(ctx, awsConfig.EC2Client())
	if err != nil {
		failWithError(err)
	}
	awsPrefixesSlice := awsManagedPrefixListIDs(managedPrefixLists)
	// Specific check for NACLs
	for logicalId, physicalId := range routetableResources {
		rulechanges := []string{}
		routetable, err := lib.GetRouteTable(ctx, physicalId, awsConfig.EC2Client())
		if err != nil {
			failWithError(err)
		}
		attachedRules := lib.FilterRoutesByLogicalId(logicalId, template, parameters, logicalToPhysical)
		for _, route := range routetable.Routes {
			ruleid := lib.GetRouteDestination(route)
			if ruleid == "" {
				continue
			}
			if route.DestinationPrefixListId != nil && (!settings.GetBool("verbose") || stringInSlice(*route.DestinationPrefixListId, awsPrefixesSlice)) {
				// If the route is for a prefixlist, don't report it by default as they're not defined in CloudFormation. Also don't report any AWS managed prefixlists
				continue
			}
			if cfnroute, ok := attachedRules[ruleid]; ok {
				if !lib.CompareRoutes(route, cfnroute, settings.GetStringSlice("drift.ignore-blackholes")) {
					ruledetails := fmt.Sprintf("Expected: %s\nActual: %s", routeToString(cfnroute), routeToString(route))
					rulechanges = append(rulechanges, ruledetails)
				}
				delete(attachedRules, ruleid)
			} else {
				// If the route was created with the table, don't report it
				if route.Origin == ec2types.RouteOriginCreateRouteTable {
					continue
				}
				ruledetails := fmt.Sprintf("Unmanaged route: %s", routeToString(route))
				rulechanges = append(rulechanges, output.StylePositive(ruledetails))
			}
		}
		// Leftover rules only exist in CloudFormation
		for routeid, cfnroute := range attachedRules {
			// routeid being empty implies it wasn't created, likely due to a condition
			if routeid == "" {
				continue
			}
			ruledetails := fmt.Sprintf("Removed route: %s", routeToString(cfnroute))
			rulechanges = append(rulechanges, output.StyleWarning(ruledetails))

		}
		if len(rulechanges) != 0 {
			if driftFlags.SeparateProperties {
				for _, change := range rulechanges {
					content := make(map[string]any)
					content["LogicalId"] = fmt.Sprintf("Route for RouteTable %s", logicalId)
					content["Type"] = "AWS::EC2::Route"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					*rows = append(*rows, content)
				}
			} else {
				content := make(map[string]any)
				content["LogicalId"] = fmt.Sprintf("Routes for RouteTable %s", logicalId)
				content["Type"] = "AWS::EC2::Route"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				*rows = append(*rows, content)
			}
		}
	}
}

// awsManagedPrefixListIDs returns the PrefixListId values for entries
// owned by "AWS". Entries with nil OwnerId or nil/empty PrefixListId are
// silently skipped so that partial SDK responses don't cause a panic.
func awsManagedPrefixListIDs(lists []ec2types.ManagedPrefixList) []string {
	result := make([]string, 0, len(lists))
	for _, pl := range lists {
		if pl.OwnerId == nil || pl.PrefixListId == nil {
			continue
		}
		if *pl.OwnerId == "AWS" && *pl.PrefixListId != "" {
			result = append(result, *pl.PrefixListId)
		}
	}
	return result
}

// checkTransitGatewayRouteTableRoutes verifies Transit Gateway routes and reports any differences
func checkTransitGatewayRouteTableRoutes(ctx context.Context, tgwRouteTableResources map[string]string, template lib.CfnTemplateBody, parameters []types.Parameter, logicalToPhysical map[string]string, rows *[]map[string]any, awsConfig config.AWSConfig) {
	// Iterate through each Transit Gateway route table
	for logicalId, physicalId := range tgwRouteTableResources {
		rulechanges := []string{}

		// Get actual routes from AWS
		routes, err := lib.GetTransitGatewayRouteTableRoutes(ctx, physicalId, awsConfig.EC2Client())
		if err != nil {
			failWithError(err)
		}

		// Get map of expected routes from template
		expectedRoutes := lib.FilterTGWRoutesByLogicalId(logicalId, template, parameters, logicalToPhysical)

		// Check each actual route against expected routes
		for _, route := range routes {
			// Filter out propagated routes (not static)
			if route.Type == ec2types.TransitGatewayRouteTypePropagated {
				continue
			}

			// Filter out routes in transient states
			if route.State != ec2types.TransitGatewayRouteStateActive && route.State != ec2types.TransitGatewayRouteStateBlackhole {
				continue
			}

			destination := lib.GetTGWRouteDestination(route)

			// Check if route exists in template
			if cfnroute, ok := expectedRoutes[destination]; ok {
				// Route exists in template, compare for modifications
				if !lib.CompareTGWRoutes(route, cfnroute, settings.GetStringSlice("drift.ignore-blackholes")) {
					ruledetails := fmt.Sprintf("Expected: %s\nActual: %s", tgwRouteToString(cfnroute), tgwRouteToString(route))
					rulechanges = append(rulechanges, ruledetails)
				}
				// Remove from expectedRoutes map to track what's been checked
				delete(expectedRoutes, destination)
			} else {
				// Route not in template, it's unmanaged
				ruledetails := fmt.Sprintf("Unmanaged route: %s", tgwRouteToString(route))
				rulechanges = append(rulechanges, output.StylePositive(ruledetails))
			}
		}

		// Check for routes that exist in template but not in AWS (removed routes)
		for destination, cfnroute := range expectedRoutes {
			// Empty destination implies the route wasn't created, likely due to a condition
			if destination == "" {
				continue
			}
			ruledetails := fmt.Sprintf("Removed route: %s", tgwRouteToString(cfnroute))
			rulechanges = append(rulechanges, output.StyleWarning(ruledetails))
		}

		// Report all changes if any were found
		if len(rulechanges) != 0 {
			if driftFlags.SeparateProperties {
				for _, change := range rulechanges {
					content := make(map[string]any)
					content["LogicalId"] = fmt.Sprintf("Route for TransitGatewayRouteTable %s", logicalId)
					content["Type"] = "AWS::EC2::TransitGatewayRoute"
					content["ChangeType"] = string(types.StackResourceDriftStatusModified)
					content["Details"] = change
					*rows = append(*rows, content)
				}
			} else {
				content := make(map[string]any)
				content["LogicalId"] = fmt.Sprintf("Routes for TransitGatewayRouteTable %s", logicalId)
				content["Type"] = "AWS::EC2::TransitGatewayRoute"
				content["ChangeType"] = string(types.StackResourceDriftStatusModified)
				content["Details"] = rulechanges
				*rows = append(*rows, content)
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

func getExpectedAndActualTags(expectedResources map[string]any, actualResources map[string]any) map[string]map[string]string {
	// if Tags exists in expectedResources, compare the list of tags with those in actualResources
	tags := make(map[string]map[string]string)
	// go through expectedResources["Tags"] and add each item in expectedTags
	if expectedTags, ok := expectedResources["Tags"].([]any); ok {
		for _, tag := range expectedTags {
			key, value, valid := extractTagKeyValue(tag)
			if !valid {
				continue
			}
			tags[key] = map[string]string{"Expected": value}
		}
	}
	// go through actualResources["Tags"] and add each item in actualTags
	if actualTags, ok := actualResources["Tags"].([]any); ok {
		for _, tag := range actualTags {
			key, value, valid := extractTagKeyValue(tag)
			if !valid {
				continue
			}
			if tags[key] == nil {
				tags[key] = map[string]string{"Expected": "", "Actual": value}
			}
			tags[key]["Actual"] = value
		}
	}
	return tags
}

// extractTagKeyValue safely extracts Key and Value strings from a tag entry.
// Returns ("", "", false) if the entry is malformed.
func extractTagKeyValue(tag any) (string, string, bool) {
	tagMap, ok := tag.(map[string]any)
	if !ok {
		return "", "", false
	}
	key, keyOk := tagMap["Key"].(string)
	value, valueOk := tagMap["Value"].(string)
	if !keyOk || !valueOk {
		return "", "", false
	}
	return key, value, true
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
			} else if separate[0] == *drift.LogicalResourceId && strings.Join(separate[1:], ":") == tag {
				return false
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
	if property.PropertyPath == nil || *property.PropertyPath == "" {
		return "", ""
	}
	pathsplit := strings.Split(*property.PropertyPath, "/")
	// Leading "/" means element [0] is always ""; the tag category sits at [1].
	// e.g. "/Tags/0/Key" → ["", "Tags", "0", "Key"] (4 elements incl. empty prefix).
	if len(pathsplit) < 2 {
		return "", ""
	}
	var expected, actual bytes.Buffer
	expectedStr := aws.ToString(property.ExpectedValue)
	actualStr := aws.ToString(property.ActualValue)
	if json.Valid([]byte(expectedStr)) {
		if err := json.Indent(&expected, []byte(expectedStr), "", "  "); err != nil {
			failWithError(err)
		}
	}
	if json.Valid([]byte(actualStr)) {
		if err := json.Indent(&actual, []byte(actualStr), "", "  "); err != nil {
			failWithError(err)
		}
	}
	switch property.DifferenceType {
	case types.DifferenceTypeRemove:
		// When expected is empty (nil ExpectedValue), no tag data is available
		// to report so we fall through to the empty return below.
		tagstructs := []tag{}
		if expected.Len() > 0 && expected.String()[0] == '[' {
			if err := json.Unmarshal(expected.Bytes(), &tagstructs); err != nil {
				failWithError(err)
			}
		} else if expected.Len() > 0 {
			tagstruct := tag{}
			if err := json.Unmarshal(expected.Bytes(), &tagstruct); err != nil {
				failWithError(err)
			}
			tagstructs = append(tagstructs, tagstruct)
		}
		for _, tagstruct := range tagstructs {
			if !shouldTagBeHandled(tagstruct.Key, *drift) {
				continue
			}
			return output.StyleWarning(fmt.Sprintf("%s: %s - %s: %s", property.DifferenceType, pathsplit[1], tagstruct.Key, tagstruct.Value)), ""
		}
		return "", ""
	case types.DifferenceTypeAdd:
		// When actual is empty (nil ActualValue), no tag data is available
		// to report so we return early.
		if actual.Len() == 0 {
			return "", ""
		}
		tagstruct := tag{}
		if err := json.Unmarshal(actual.Bytes(), &tagstruct); err != nil {
			failWithError(err)
		}
		if !shouldTagBeHandled(tagstruct.Key, *drift) {
			return "", ""
		}
		return output.StylePositive(fmt.Sprintf("%s: %s - %s: %s", property.DifferenceType, pathsplit[1], tagstruct.Key, tagstruct.Value)), ""
	default:
		tagKey := ""
		tags := map[string]string{}
		// loop over the tags in the tagMap and see if the "Expected" value matches *property.ExpectedValue
		for key, values := range tagMap {
			if values["Expected"] == expectedStr && values["Actual"] == actualStr {
				tagKey = key
				tags = values
			}
		}
		if !shouldTagBeHandled(tagKey, *drift) {
			return "", ""
		}
		// Full tag property paths have 4 segments (e.g. "/Tags/0/Key"),
		// shorter paths skip the Key/Value-specific logic.
		if len(pathsplit) >= 4 {
			if pathsplit[3] == "Key" {
				if tags["Expected"] == tags["Actual"] {
					return fmt.Sprintf("%s: Tag %s sequence change", property.DifferenceType, aws.ToString(property.ExpectedValue)), pathsplit[2]
				}
			} else if pathsplit[3] == "Value" && stringInSlice(pathsplit[2], handledtags) {
				return "", ""
			}
		}
		return fmt.Sprintf("%s: %s - %s => %s", property.DifferenceType, tagKey, tags["Expected"], tags["Actual"]), ""
	}
}

// naclEntryKey builds the map key used to match an EC2 NACL entry against the
// CloudFormation template rules. The key has the form "I<ruleNumber>" for
// ingress or "E<ruleNumber>" for egress. When Egress is nil, defaults to "I"
// (ingress); when RuleNumber is nil, defaults to "unknown" — producing a
// degenerate key like "Iunknown" that will not collide with valid CFN rules.
func naclEntryKey(entry ec2types.NetworkAclEntry) string {
	prefix := "I"
	if entry.Egress != nil && *entry.Egress {
		prefix = "E"
	}
	ruleNum := "unknown"
	if entry.RuleNumber != nil {
		ruleNum = strconv.Itoa(int(*entry.RuleNumber))
	}
	return prefix + ruleNum
}

func naclEntryToString(entry ec2types.NetworkAclEntry) string {
	direction := "ingress"
	if entry.Egress != nil && *entry.Egress {
		direction = "egress"
	}
	ports := "Ports: All"
	if entry.PortRange != nil {
		switch {
		case entry.PortRange.From == nil && entry.PortRange.To == nil:
			ports = "Ports: ?-?"
		case entry.PortRange.From == nil:
			ports = fmt.Sprintf("Ports: ?-%v", *entry.PortRange.To)
		case entry.PortRange.To == nil:
			ports = fmt.Sprintf("Ports: %v-?", *entry.PortRange.From)
		case *entry.PortRange.From == *entry.PortRange.To:
			ports = fmt.Sprintf("Port: %v", *entry.PortRange.From)
		default:
			ports = fmt.Sprintf("Ports: %v-%v", *entry.PortRange.From, *entry.PortRange.To)
		}
	}
	if entry.IcmpTypeCode != nil {
		switch {
		case entry.IcmpTypeCode.Type == nil:
			ports = "ICMP: unknown"
		case *entry.IcmpTypeCode.Type == -1:
			ports = "ICMP: All"
		case entry.IcmpTypeCode.Code == nil:
			ports = fmt.Sprintf("ICMP: %v-?", *entry.IcmpTypeCode.Type)
		default:
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
	ruleNum := "unknown"
	if entry.RuleNumber != nil {
		ruleNum = strconv.Itoa(int(*entry.RuleNumber))
	}
	protocol := aws.ToString(entry.Protocol)
	return fmt.Sprintf("%s #%v %v: %s, %s %s", direction, ruleNum, entry.RuleAction, protocol, cidr, ports)
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

func tgwRouteToString(route ec2types.TransitGatewayRoute) string {
	destination := lib.GetTGWRouteDestination(route)
	target := lib.GetTGWRouteTarget(route)
	status := ""
	if route.State == ec2types.TransitGatewayRouteStateBlackhole {
		status = " (blackhole)"
	}
	return fmt.Sprintf("%s: %s%s", destination, target, status)
}
