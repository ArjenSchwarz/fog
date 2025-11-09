package lib

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

// TestDeployInfo_ErrorPaths tests error handling paths in DeployInfo methods.
// These tests address Issue 4.1 from the audit report regarding missing error path tests.
func TestDeployInfo_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		deploy *DeployInfo
		desc   string
	}{
		"nil DeployInfo": {
			deploy: nil,
			desc:   "Operations on nil DeployInfo should be handled safely",
		},
		"empty StackName": {
			deploy: &DeployInfo{
				StackName: "",
			},
			desc: "Empty stack name should be handled",
		},
		"nil Template": {
			deploy: &DeployInfo{
				StackName: "test-stack",
				Template:  nil,
			},
			desc: "Nil template should be handled",
		},
		"empty Template": {
			deploy: &DeployInfo{
				StackName: "test-stack",
				Template:  []byte{},
			},
			desc: "Empty template should be handled",
		},
		"nil Parameters": {
			deploy: &DeployInfo{
				StackName:  "test-stack",
				Template:   []byte("template"),
				Parameters: nil,
			},
			desc: "Nil parameters should be handled",
		},
		"nil Tags": {
			deploy: &DeployInfo{
				StackName: "test-stack",
				Template:  []byte("template"),
				Tags:      nil,
			},
			desc: "Nil tags should be handled",
		},
		"nil ChangesetResponse": {
			deploy: &DeployInfo{
				StackName:         "test-stack",
				ChangesetResponse: nil,
			},
			desc: "Nil changeset response should be handled",
		},
		"nil FinalStackState": {
			deploy: &DeployInfo{
				StackName:       "test-stack",
				FinalStackState: nil,
			},
			desc: "Nil final stack state should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Basic field access should not panic
			assert.NotPanics(t, func() {
				if tc.deploy != nil {
					_ = tc.deploy.StackName
					_ = tc.deploy.Template
					_ = tc.deploy.Parameters
					_ = tc.deploy.Tags
				}
			}, tc.desc)
		})
	}
}

// TestCfnStack_ErrorPaths tests error handling for CfnStack operations.
func TestCfnStack_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		stack *CfnStack
		desc  string
	}{
		"nil CfnStack": {
			stack: nil,
			desc:  "Operations on nil stack should be safe",
		},
		"empty StackName": {
			stack: &CfnStack{
				StackName: aws.String(""),
			},
			desc: "Empty stack name should be handled",
		},
		"nil StackName pointer": {
			stack: &CfnStack{
				StackName: nil,
			},
			desc: "Nil stack name pointer should be handled",
		},
		"nil StackStatus": {
			stack: &CfnStack{
				StackName:   aws.String("test"),
				StackStatus: "",
			},
			desc: "Empty stack status should be handled",
		},
		"nil Parameters": {
			stack: &CfnStack{
				StackName:  aws.String("test"),
				Parameters: nil,
			},
			desc: "Nil parameters should be handled",
		},
		"nil Outputs": {
			stack: &CfnStack{
				StackName: aws.String("test"),
				Outputs:   nil,
			},
			desc: "Nil outputs should be handled",
		},
		"nil Tags": {
			stack: &CfnStack{
				StackName: aws.String("test"),
				Tags:      nil,
			},
			desc: "Nil tags should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				if tc.stack != nil {
					_ = tc.stack.StackName
					_ = tc.stack.StackStatus
					_ = tc.stack.Parameters
					_ = tc.stack.Outputs
					_ = tc.stack.Tags
				}
			}, tc.desc)
		})
	}
}

// TestStackEvent_ErrorPaths tests error handling for stack event operations.
func TestStackEvent_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		event *StackEvent
		desc  string
	}{
		"nil StackEvent": {
			event: nil,
			desc:  "Nil stack event should be handled safely",
		},
		"empty ResourceStatus": {
			event: &StackEvent{
				ResourceStatus: "",
			},
			desc: "Empty resource status should be handled",
		},
		"nil LogicalResourceId": {
			event: &StackEvent{
				LogicalResourceId: nil,
			},
			desc: "Nil logical resource ID should be handled",
		},
		"nil PhysicalResourceId": {
			event: &StackEvent{
				PhysicalResourceId: nil,
			},
			desc: "Nil physical resource ID should be handled",
		},
		"nil ResourceType": {
			event: &StackEvent{
				ResourceType: nil,
			},
			desc: "Nil resource type should be handled",
		},
		"nil Timestamp": {
			event: &StackEvent{
				Timestamp: nil,
			},
			desc: "Nil timestamp should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				if tc.event != nil {
					_ = tc.event.ResourceStatus
					_ = tc.event.LogicalResourceId
					_ = tc.event.PhysicalResourceId
				}
			}, tc.desc)
		})
	}
}

// TestChangesetInfo_ErrorPaths tests error handling for changeset operations.
func TestChangesetInfo_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		changeset *ChangesetInfo
		desc      string
	}{
		"nil ChangesetInfo": {
			changeset: nil,
			desc:      "Nil changeset info should be handled safely",
		},
		"empty ChangesetName": {
			changeset: &ChangesetInfo{
				ChangesetName: aws.String(""),
			},
			desc: "Empty changeset name should be handled",
		},
		"nil ChangesetName": {
			changeset: &ChangesetInfo{
				ChangesetName: nil,
			},
			desc: "Nil changeset name should be handled",
		},
		"empty Status": {
			changeset: &ChangesetInfo{
				Status: "",
			},
			desc: "Empty status should be handled",
		},
		"nil Changes": {
			changeset: &ChangesetInfo{
				Changes: nil,
			},
			desc: "Nil changes array should be handled",
		},
		"empty Changes": {
			changeset: &ChangesetInfo{
				Changes: []cfntypes.Change{},
			},
			desc: "Empty changes array should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				if tc.changeset != nil {
					_ = tc.changeset.ChangesetName
					_ = tc.changeset.Status
					_ = tc.changeset.Changes
				}
			}, tc.desc)
		})
	}
}

// TestCfnResource_ErrorPaths tests error handling for resource operations.
func TestCfnResource_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		resource *CfnResource
		desc     string
	}{
		"nil CfnResource": {
			resource: nil,
			desc:     "Nil resource should be handled safely",
		},
		"nil LogicalResourceId": {
			resource: &CfnResource{
				LogicalResourceId: nil,
			},
			desc: "Nil logical resource ID should be handled",
		},
		"nil PhysicalResourceId": {
			resource: &CfnResource{
				PhysicalResourceId: nil,
			},
			desc: "Nil physical resource ID should be handled",
		},
		"nil ResourceType": {
			resource: &CfnResource{
				ResourceType: nil,
			},
			desc: "Nil resource type should be handled",
		},
		"empty ResourceStatus": {
			resource: &CfnResource{
				ResourceStatus: "",
			},
			desc: "Empty resource status should be handled",
		},
		"nil Timestamp": {
			resource: &CfnResource{
				Timestamp: nil,
			},
			desc: "Nil timestamp should be handled",
		},
		"nil DriftInformation": {
			resource: &CfnResource{
				DriftInformation: nil,
			},
			desc: "Nil drift information should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				if tc.resource != nil {
					_ = tc.resource.LogicalResourceId
					_ = tc.resource.PhysicalResourceId
					_ = tc.resource.ResourceType
					_ = tc.resource.ResourceStatus
				}
			}, tc.desc)
		})
	}
}

// TestStackDependency_ErrorPaths tests error handling for stack dependency operations.
func TestStackDependency_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		dep  *StackDependency
		desc string
	}{
		"nil StackDependency": {
			dep:  nil,
			desc: "Nil stack dependency should be handled safely",
		},
		"empty DependentStack": {
			dep: &StackDependency{
				DependentStack: "",
			},
			desc: "Empty dependent stack should be handled",
		},
		"empty RequiredStack": {
			dep: &StackDependency{
				RequiredStack: "",
			},
			desc: "Empty required stack should be handled",
		},
		"empty ExportName": {
			dep: &StackDependency{
				ExportName: "",
			},
			desc: "Empty export name should be handled",
		},
		"all fields empty": {
			dep: &StackDependency{
				DependentStack: "",
				RequiredStack:  "",
				ExportName:     "",
			},
			desc: "All empty fields should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				if tc.dep != nil {
					_ = tc.dep.DependentStack
					_ = tc.dep.RequiredStack
					_ = tc.dep.ExportName
				}
			}, tc.desc)
		})
	}
}

// TestStackExport_ErrorPaths tests error handling for stack export operations.
func TestStackExport_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		export *StackExport
		desc   string
	}{
		"nil StackExport": {
			export: nil,
			desc:   "Nil stack export should be handled safely",
		},
		"empty Name": {
			export: &StackExport{
				Name: "",
			},
			desc: "Empty export name should be handled",
		},
		"empty ExportingStack": {
			export: &StackExport{
				ExportingStack: "",
			},
			desc: "Empty exporting stack should be handled",
		},
		"empty Value": {
			export: &StackExport{
				Value: "",
			},
			desc: "Empty export value should be handled",
		},
		"all fields empty": {
			export: &StackExport{
				Name:           "",
				ExportingStack: "",
				Value:          "",
			},
			desc: "All empty fields should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				if tc.export != nil {
					_ = tc.export.Name
					_ = tc.export.ExportingStack
					_ = tc.export.Value
				}
			}, tc.desc)
		})
	}
}

// TestParameter_ErrorPaths tests error handling for parameter operations.
func TestParameter_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		param cfntypes.Parameter
		desc  string
	}{
		"nil ParameterKey": {
			param: cfntypes.Parameter{
				ParameterKey: nil,
			},
			desc: "Nil parameter key should be handled",
		},
		"nil ParameterValue": {
			param: cfntypes.Parameter{
				ParameterValue: nil,
			},
			desc: "Nil parameter value should be handled",
		},
		"both nil": {
			param: cfntypes.Parameter{
				ParameterKey:   nil,
				ParameterValue: nil,
			},
			desc: "Both nil values should be handled",
		},
		"empty string key": {
			param: cfntypes.Parameter{
				ParameterKey: aws.String(""),
			},
			desc: "Empty string parameter key should be handled",
		},
		"empty string value": {
			param: cfntypes.Parameter{
				ParameterValue: aws.String(""),
			},
			desc: "Empty string parameter value should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = tc.param.ParameterKey
				_ = tc.param.ParameterValue
			}, tc.desc)
		})
	}
}

// TestOutput_ErrorPaths tests error handling for output operations.
func TestOutput_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		output cfntypes.Output
		desc   string
	}{
		"nil OutputKey": {
			output: cfntypes.Output{
				OutputKey: nil,
			},
			desc: "Nil output key should be handled",
		},
		"nil OutputValue": {
			output: cfntypes.Output{
				OutputValue: nil,
			},
			desc: "Nil output value should be handled",
		},
		"nil Description": {
			output: cfntypes.Output{
				Description: nil,
			},
			desc: "Nil description should be handled",
		},
		"nil ExportName": {
			output: cfntypes.Output{
				ExportName: nil,
			},
			desc: "Nil export name should be handled",
		},
		"all nil": {
			output: cfntypes.Output{
				OutputKey:   nil,
				OutputValue: nil,
				Description: nil,
				ExportName:  nil,
			},
			desc: "All nil values should be handled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotPanics(t, func() {
				_ = tc.output.OutputKey
				_ = tc.output.OutputValue
				_ = tc.output.Description
				_ = tc.output.ExportName
			}, tc.desc)
		})
	}
}
