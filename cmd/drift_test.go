package cmd

import (
	"context"
	"testing"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/viper"
)

// TestDrift_V2BuilderPattern tests the v2 Builder pattern for drift command
func TestDrift_V2BuilderPattern(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Create sample drift detection data
	driftData := []map[string]any{
		{
			"LogicalId":  "VPC",
			"Type":       "AWS::EC2::VPC",
			"ChangeType": output.StyleWarning("DELETED"),
			"Details":    "VPC has been deleted in AWS",
		},
		{
			"LogicalId":  "PublicSubnet",
			"Type":       "AWS::EC2::Subnet",
			"ChangeType": "MODIFIED",
			"Details":    []string{"CIDR Block changed", "Tags modified"},
		},
	}

	tests := map[string]struct {
		columnOrder []string
	}{
		"drift_table_columns": {
			columnOrder: []string{"LogicalId", "Type", "ChangeType", "Details"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Build document using v2 Builder pattern
			doc := output.New().
				Table(
					"Drift results for stack test-stack",
					driftData,
					output.WithKeys(tc.columnOrder...),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Verify rendering doesn't error
			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render drift output: %v", err)
			}
		})
	}
}

// TestDrift_V2InlineStyling tests inline styling in drift output
func TestDrift_V2InlineStyling(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		changeType string
		expected   string
	}{
		"deleted_warning": {
			changeType: output.StyleWarning("DELETED"),
			expected:   "DELETED",
		},
		"created_positive": {
			changeType: output.StylePositive("CREATE_IN_PROGRESS"),
			expected:   "CREATE_IN_PROGRESS",
		},
		"modified_plain": {
			changeType: "MODIFIED",
			expected:   "MODIFIED",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data with styled change type
			data := []map[string]any{
				{
					"LogicalId":  "Resource1",
					"Type":       "AWS::S3::Bucket",
					"ChangeType": tc.changeType,
					"Details":    "Resource property changed",
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Drift Detection Results",
					data,
					output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render
			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render styled output: %v", err)
			}
		})
	}
}

// TestDrift_V2ArrayHandling tests array handling for property differences
func TestDrift_V2ArrayHandling(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		details []string
	}{
		"no_properties": {
			details: []string{},
		},
		"single_property": {
			details: []string{"Property1 changed"},
		},
		"multiple_properties": {
			details: []string{
				"CIDR Block: 10.0.0.0/16 => 10.0.0.0/24",
				"Tags: removed key1",
				"Availability Zone: us-east-1a => us-east-1b",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data with array of property differences
			data := []map[string]any{
				{
					"LogicalId":  "TestResource",
					"Type":       "AWS::EC2::Subnet",
					"ChangeType": "MODIFIED",
					"Details":    tc.details,
				},
			}

			// Build document - v2 should handle arrays automatically
			doc := output.New().
				Table(
					"Property Differences",
					data,
					output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Render
			out := output.NewOutput(
				output.WithFormat(output.Table()),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render array output: %v", err)
			}
		})
	}
}

// TestDrift_V2NACLDifferences tests NACL entry differences handling
func TestDrift_V2NACLDifferences(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// NACL entry difference data
	naclData := []map[string]any{
		{
			"LogicalId":  "Entry for NACL nacl-12345",
			"Type":       "AWS::EC2::NetworkACLEntry",
			"ChangeType": "MODIFIED",
			"Details": []string{
				output.StyleWarning("Removed entry: Ingress #100"),
				output.StylePositive("Unmanaged entry: Egress #32767"),
			},
		},
	}

	// Build document
	doc := output.New().
		Table(
			"NACL Drift Detection",
			naclData,
			output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Render
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render NACL output: %v", err)
	}
}

// TestDrift_V2RouteTableDifferences tests Route Table differences handling
func TestDrift_V2RouteTableDifferences(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Route table difference data
	routeData := []map[string]any{
		{
			"LogicalId":  "Route for RouteTable rtb-12345",
			"Type":       "AWS::EC2::Route",
			"ChangeType": "MODIFIED",
			"Details": []string{
				output.StyleWarning("Removed route: 0.0.0.0/0 => igw-12345"),
				output.StylePositive("Unmanaged route: 10.0.0.0/8 => vpce-12345"),
			},
		},
	}

	// Build document
	doc := output.New().
		Table(
			"Route Table Drift Detection",
			routeData,
			output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Render
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render route table output: %v", err)
	}
}

// TestDrift_V2TransitGatewayDifferences tests Transit Gateway route differences
func TestDrift_V2TransitGatewayDifferences(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// TGW route difference data
	tgwData := []map[string]any{
		{
			"LogicalId":  "Route for TransitGatewayRouteTable tgw-rtb-12345",
			"Type":       "AWS::EC2::TransitGatewayRoute",
			"ChangeType": "MODIFIED",
			"Details": []string{
				output.StyleWarning("Removed route: 192.168.0.0/16 (active)"),
				output.StylePositive("Unmanaged route: 172.16.0.0/12 (blackhole)"),
			},
		},
	}

	// Build document
	doc := output.New().
		Table(
			"Transit Gateway Drift Detection",
			tgwData,
			output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Render
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render TGW output: %v", err)
	}
}

// TestDrift_V2MultilineProperties tests multi-line property value rendering
func TestDrift_V2MultilineProperties(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Multi-line property data
	multilineData := []map[string]any{
		{
			"LogicalId":  "IAMRole",
			"Type":       "AWS::IAM::Role",
			"ChangeType": "MODIFIED",
			"Details": []string{
				"Expected:\n{\n  \"Version\": \"2012-10-17\",\n  \"Statement\": []\n}\nActual:\n{\n  \"Version\": \"2012-10-17\",\n  \"Statement\": [{\"Effect\": \"Allow\"}]\n}",
			},
		},
	}

	// Build document
	doc := output.New().
		Table(
			"Multi-line Properties",
			multilineData,
			output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Render
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render multi-line properties: %v", err)
	}
}

// TestDrift_V2OutputFormats tests drift output in different formats
func TestDrift_V2OutputFormats(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	tests := map[string]struct {
		format output.Format
	}{
		"table_format": {
			format: output.Table(),
		},
		"csv_format": {
			format: output.CSV(),
		},
		"json_format": {
			format: output.JSON(),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Cannot use t.Parallel() because viper uses global state

			// Setup viper
			viper.Set("output", "table")
			viper.Set("table.style", "Default")
			viper.Set("table.max-column-width", 50)

			// Test data
			data := []map[string]any{
				{
					"LogicalId":  "Resource1",
					"Type":       "AWS::EC2::VPC",
					"ChangeType": "MODIFIED",
					"Details":    []string{"CIDR Block changed"},
				},
			}

			// Build document
			doc := output.New().
				Table(
					"Drift Detection",
					data,
					output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
				).
				Build()

			if doc == nil {
				t.Fatal("Built document should not be nil")
			}

			// Create output with specific format
			out := output.NewOutput(
				output.WithFormat(tc.format),
				output.WithWriter(output.NewStdoutWriter()),
			)

			err := out.Render(context.Background(), doc)
			if err != nil {
				t.Fatalf("Failed to render in format %s: %v", tc.format.Name, err)
			}
		})
	}
}

// TestDrift_V2ComplexScenario tests a complex drift scenario with multiple differences
func TestDrift_V2ComplexScenario(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because viper uses global state

	// Setup viper
	viper.Set("output", "table")
	viper.Set("table.style", "Default")
	viper.Set("table.max-column-width", 50)

	// Complex drift scenario
	complexData := []map[string]any{
		{
			"LogicalId":  "VPC",
			"Type":       "AWS::EC2::VPC",
			"ChangeType": output.StyleWarning("DELETED"),
			"Details":    "VPC has been deleted manually in AWS account",
		},
		{
			"LogicalId":  "Subnet1",
			"Type":       "AWS::EC2::Subnet",
			"ChangeType": "MODIFIED",
			"Details": []string{
				"CIDR Block: 10.0.1.0/24 => 10.0.1.0/25",
				"AvailabilityZone: us-east-1a => us-east-1b",
			},
		},
		{
			"LogicalId":  "SecurityGroup",
			"Type":       "AWS::EC2::SecurityGroup",
			"ChangeType": "MODIFIED",
			"Details": []string{
				output.StylePositive("Added inbound rule: TCP 443 from 0.0.0.0/0"),
				output.StyleWarning("Removed inbound rule: TCP 22 from 10.0.0.0/8"),
			},
		},
	}

	// Build document
	doc := output.New().
		Table(
			"Complex Drift Detection Scenario",
			complexData,
			output.WithKeys("LogicalId", "Type", "ChangeType", "Details"),
		).
		Build()

	if doc == nil {
		t.Fatal("Built document should not be nil")
	}

	// Render
	out := output.NewOutput(
		output.WithFormat(output.Table()),
		output.WithWriter(output.NewStdoutWriter()),
	)

	err := out.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to render complex scenario: %v", err)
	}
}

// TestShouldTagBeHandled tests the shouldTagBeHandled function
func TestShouldTagBeHandled(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because driftFlags uses global state

	tests := map[string]struct {
		tag          string
		ignoreTags   string
		resourceType string
		logicalID    string
		want         bool
	}{
		"tag_not_ignored": {
			tag:          "Environment",
			ignoreTags:   "Owner,CostCenter",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         true,
		},
		"tag_ignored_globally": {
			tag:          "Owner",
			ignoreTags:   "Owner,CostCenter",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         false,
		},
		"tag_ignored_by_resource_type": {
			tag:          "Environment",
			ignoreTags:   "AWS::EC2::VPC:Environment",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         false,
		},
		"tag_not_ignored_different_resource_type": {
			tag:          "Environment",
			ignoreTags:   "AWS::EC2::Subnet:Environment",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         true,
		},
		"tag_ignored_by_logical_id": {
			tag:          "Environment",
			ignoreTags:   "MyVPC:Environment",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         false,
		},
		"tag_not_ignored_different_logical_id": {
			tag:          "Environment",
			ignoreTags:   "OtherVPC:Environment",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         true,
		},
		"empty_ignore_tags": {
			tag:          "Environment",
			ignoreTags:   "",
			resourceType: "AWS::EC2::VPC",
			logicalID:    "MyVPC",
			want:         true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			driftFlags = DriftFlags{IgnoreTags: tc.ignoreTags}
			viper.Reset()

			drift := types.StackResourceDrift{
				ResourceType:      &tc.resourceType,
				LogicalResourceId: &tc.logicalID,
			}

			got := shouldTagBeHandled(tc.tag, drift)
			if got != tc.want {
				t.Errorf("shouldTagBeHandled(%q) = %v, want %v", tc.tag, got, tc.want)
			}
		})
	}
}

// TestTagDifferences_IgnoreTags tests that tagDifferences respects ignore-tags for all difference types
func TestTagDifferences_IgnoreTags(t *testing.T) {
	// NOTE: Cannot use t.Parallel() because driftFlags uses global state

	resourceType := "AWS::EC2::VPC"
	logicalID := "MyVPC"

	tests := map[string]struct {
		differenceType types.DifferenceType
		propertyPath   string
		expectedValue  string
		actualValue    string
		ignoreTags     string
		tagMap         map[string]map[string]string
		wantOutput     bool
	}{
		"add_tag_not_ignored": {
			differenceType: types.DifferenceTypeAdd,
			propertyPath:   "/Tags/0",
			expectedValue:  "null",
			actualValue:    `{"Key":"Environment","Value":"Production"}`,
			ignoreTags:     "Owner",
			wantOutput:     true,
		},
		"add_tag_ignored": {
			differenceType: types.DifferenceTypeAdd,
			propertyPath:   "/Tags/0",
			expectedValue:  "null",
			actualValue:    `{"Key":"Owner","Value":"TeamA"}`,
			ignoreTags:     "Owner",
			wantOutput:     false,
		},
		"remove_tag_not_ignored": {
			differenceType: types.DifferenceTypeRemove,
			propertyPath:   "/Tags/0",
			expectedValue:  `{"Key":"Environment","Value":"Production"}`,
			actualValue:    "null",
			ignoreTags:     "Owner",
			wantOutput:     true,
		},
		"remove_tag_ignored": {
			differenceType: types.DifferenceTypeRemove,
			propertyPath:   "/Tags/0",
			expectedValue:  `{"Key":"Owner","Value":"TeamA"}`,
			actualValue:    "null",
			ignoreTags:     "Owner",
			wantOutput:     false,
		},
		"remove_multiple_tags_one_ignored": {
			differenceType: types.DifferenceTypeRemove,
			propertyPath:   "/Tags/0",
			expectedValue:  `[{"Key":"Owner","Value":"TeamA"},{"Key":"Environment","Value":"Prod"}]`,
			actualValue:    "null",
			ignoreTags:     "Owner",
			wantOutput:     true, // Environment is not ignored, so it should output
		},
		"remove_multiple_tags_all_ignored": {
			differenceType: types.DifferenceTypeRemove,
			propertyPath:   "/Tags/0",
			expectedValue:  `[{"Key":"Owner","Value":"TeamA"},{"Key":"CostCenter","Value":"123"}]`,
			actualValue:    "null",
			ignoreTags:     "Owner,CostCenter",
			wantOutput:     false,
		},
		"modify_tag_not_ignored": {
			differenceType: types.DifferenceTypeNotEqual,
			propertyPath:   "/Tags/0/Value",
			expectedValue:  `"Staging"`,
			actualValue:    `"Production"`,
			ignoreTags:     "Owner",
			tagMap: map[string]map[string]string{
				"Environment": {"Expected": `"Staging"`, "Actual": `"Production"`},
			},
			wantOutput: true,
		},
		"modify_tag_ignored": {
			differenceType: types.DifferenceTypeNotEqual,
			propertyPath:   "/Tags/0/Value",
			expectedValue:  `"TeamA"`,
			actualValue:    `"TeamB"`,
			ignoreTags:     "Owner",
			tagMap: map[string]map[string]string{
				"Owner": {"Expected": `"TeamA"`, "Actual": `"TeamB"`},
			},
			wantOutput: false,
		},
		"add_tag_ignored_by_resource_type": {
			differenceType: types.DifferenceTypeAdd,
			propertyPath:   "/Tags/0",
			expectedValue:  "null",
			actualValue:    `{"Key":"Environment","Value":"Production"}`,
			ignoreTags:     "AWS::EC2::VPC:Environment",
			wantOutput:     false,
		},
		"add_tag_not_ignored_different_resource_type": {
			differenceType: types.DifferenceTypeAdd,
			propertyPath:   "/Tags/0",
			expectedValue:  "null",
			actualValue:    `{"Key":"Environment","Value":"Production"}`,
			ignoreTags:     "AWS::EC2::Subnet:Environment",
			wantOutput:     true,
		},
		"remove_tag_ignored_by_logical_id": {
			differenceType: types.DifferenceTypeRemove,
			propertyPath:   "/Tags/0",
			expectedValue:  `{"Key":"Environment","Value":"Production"}`,
			actualValue:    "null",
			ignoreTags:     "MyVPC:Environment",
			wantOutput:     false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Reset global state
			driftFlags = DriftFlags{IgnoreTags: tc.ignoreTags}
			viper.Reset()

			drift := types.StackResourceDrift{
				ResourceType:      &resourceType,
				LogicalResourceId: &logicalID,
			}

			property := types.PropertyDifference{
				DifferenceType: tc.differenceType,
				PropertyPath:   &tc.propertyPath,
				ExpectedValue:  &tc.expectedValue,
				ActualValue:    &tc.actualValue,
			}

			tagMap := tc.tagMap
			if tagMap == nil {
				tagMap = map[string]map[string]string{}
			}

			result, _ := tagDifferences(property, []string{}, tagMap, []string{}, &drift)

			hasOutput := result != ""
			if hasOutput != tc.wantOutput {
				t.Errorf("tagDifferences() returned %q, wantOutput=%v (got output=%v)", result, tc.wantOutput, hasOutput)
			}
		})
	}
}
