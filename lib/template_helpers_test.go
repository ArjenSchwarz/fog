package lib

import (
	"bytes"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestExtractRuleNumber tests the safe extraction of rule numbers from NACL properties
func TestExtractRuleNumber(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		want       int32
	}{
		{
			name:       "valid float64 rule number",
			properties: map[string]any{"RuleNumber": float64(100)},
			want:       100,
		},
		{
			name:       "missing rule number",
			properties: map[string]any{},
			want:       0,
		},
		{
			name:       "invalid type",
			properties: map[string]any{"RuleNumber": "not a number"},
			want:       0,
		},
		{
			name:       "negative number",
			properties: map[string]any{"RuleNumber": float64(-1)},
			want:       -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRuleNumber(tt.properties)
			if got != tt.want {
				t.Errorf("extractRuleNumber() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractEgressFlag tests the safe extraction of egress flags from NACL properties
func TestExtractEgressFlag(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		want       bool
	}{
		{
			name:       "egress true",
			properties: map[string]any{"Egress": true},
			want:       true,
		},
		{
			name:       "egress false",
			properties: map[string]any{"Egress": false},
			want:       false,
		},
		{
			name:       "missing egress",
			properties: map[string]any{},
			want:       false,
		},
		{
			name:       "invalid type",
			properties: map[string]any{"Egress": "true"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEgressFlag(tt.properties)
			if got != tt.want {
				t.Errorf("extractEgressFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractProtocol tests protocol extraction from NACL properties
func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		want       string
	}{
		{
			name:       "string protocol",
			properties: map[string]any{"Protocol": "tcp"},
			want:       "tcp",
		},
		{
			name:       "numeric protocol as float64",
			properties: map[string]any{"Protocol": float64(6)},
			want:       "6",
		},
		{
			name:       "missing protocol",
			properties: map[string]any{},
			want:       "",
		},
		{
			name:       "invalid type",
			properties: map[string]any{"Protocol": true},
			want:       "",
		},
		{
			name:       "all traffic protocol (-1)",
			properties: map[string]any{"Protocol": float64(-1)},
			want:       "-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProtocol(tt.properties)
			if got != tt.want {
				t.Errorf("extractProtocol() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestResolveParameterValue tests parameter reference resolution
func TestResolveParameterValue(t *testing.T) {
	params := []cfntypes.Parameter{
		{
			ParameterKey:   aws.String("VpcCidr"),
			ParameterValue: aws.String("10.0.0.0/16"),
		},
		{
			ParameterKey:    aws.String("SubnetCidr"),
			ResolvedValue:   aws.String("10.0.1.0/24"),
			ParameterValue:  aws.String("10.0.2.0/24"),
		},
	}

	tests := []struct {
		name    string
		refname string
		want    string
	}{
		{
			name:    "resolve with ParameterValue",
			refname: "VpcCidr",
			want:    "10.0.0.0/16",
		},
		{
			name:    "resolve with ResolvedValue takes precedence",
			refname: "SubnetCidr",
			want:    "10.0.1.0/24",
		},
		{
			name:    "non-existent parameter",
			refname: "NonExistent",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveParameterValue(tt.refname, params)
			if got != tt.want {
				t.Errorf("resolveParameterValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractCidrBlock tests CIDR block extraction with parameter resolution
func TestExtractCidrBlock(t *testing.T) {
	params := []cfntypes.Parameter{
		{
			ParameterKey:   aws.String("VpcCidr"),
			ParameterValue: aws.String("10.0.0.0/16"),
		},
	}

	tests := []struct {
		name       string
		properties map[string]any
		key        string
		want       string
	}{
		{
			name:       "direct string value",
			properties: map[string]any{"CidrBlock": "192.168.1.0/24"},
			key:        "CidrBlock",
			want:       "192.168.1.0/24",
		},
		{
			name:       "parameter reference",
			properties: map[string]any{"CidrBlock": map[string]any{"Ref": "VpcCidr"}},
			key:        "CidrBlock",
			want:       "10.0.0.0/16",
		},
		{
			name:       "missing property",
			properties: map[string]any{},
			key:        "CidrBlock",
			want:       "",
		},
		{
			name:       "nil value",
			properties: map[string]any{"CidrBlock": nil},
			key:        "CidrBlock",
			want:       "",
		},
		{
			name:       "invalid ref format",
			properties: map[string]any{"CidrBlock": map[string]any{"NotRef": "value"}},
			key:        "CidrBlock",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCidrBlock(tt.properties, tt.key, params)
			if got != tt.want {
				t.Errorf("extractCidrBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractRuleAction tests rule action extraction
func TestExtractRuleAction(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		want       types.RuleAction
	}{
		{
			name:       "allow action",
			properties: map[string]any{"RuleAction": "allow"},
			want:       types.RuleActionAllow,
		},
		{
			name:       "deny action",
			properties: map[string]any{"RuleAction": "deny"},
			want:       types.RuleActionDeny,
		},
		{
			name:       "missing action defaults to allow",
			properties: map[string]any{},
			want:       types.RuleActionAllow,
		},
		{
			name:       "invalid type defaults to allow",
			properties: map[string]any{"RuleAction": 123},
			want:       types.RuleActionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRuleAction(tt.properties)
			if got != tt.want {
				t.Errorf("extractRuleAction() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestExtractPortRange tests port range extraction with validation
func TestExtractPortRange(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		wantNil    bool
		wantFrom   int32
		wantTo     int32
	}{
		{
			name: "valid port range",
			properties: map[string]any{
				"PortRange": map[string]any{
					"From": float64(80),
					"To":   float64(443),
				},
			},
			wantNil:  false,
			wantFrom: 80,
			wantTo:   443,
		},
		{
			name: "string port values",
			properties: map[string]any{
				"PortRange": map[string]any{
					"From": "22",
					"To":   "22",
				},
			},
			wantNil:  false,
			wantFrom: 22,
			wantTo:   22,
		},
		{
			name:       "missing PortRange",
			properties: map[string]any{},
			wantNil:    true,
		},
		{
			name:       "nil PortRange",
			properties: map[string]any{"PortRange": nil},
			wantNil:    true,
		},
		{
			name:       "invalid type for PortRange",
			properties: map[string]any{"PortRange": "not a map"},
			wantNil:    true,
		},
		{
			name: "both nil From and To",
			properties: map[string]any{
				"PortRange": map[string]any{
					"From": nil,
					"To":   nil,
				},
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPortRange(tt.properties)
			if tt.wantNil {
				if got != nil {
					t.Errorf("extractPortRange() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatal("extractPortRange() = nil, want non-nil")
				}
				if *got.From != tt.wantFrom {
					t.Errorf("extractPortRange().From = %v, want %v", *got.From, tt.wantFrom)
				}
				if *got.To != tt.wantTo {
					t.Errorf("extractPortRange().To = %v, want %v", *got.To, tt.wantTo)
				}
			}
		})
	}
}

// TestExtractIcmpTypeCode tests ICMP type code extraction
func TestExtractIcmpTypeCode(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]any
		wantNil    bool
		wantCode   int32
		wantType   int32
	}{
		{
			name: "valid ICMP type code",
			properties: map[string]any{
				"Icmp": map[string]any{
					"Code": float64(0),
					"Type": float64(8),
				},
			},
			wantNil:  false,
			wantCode: 0,
			wantType: 8,
		},
		{
			name: "string values",
			properties: map[string]any{
				"Icmp": map[string]any{
					"Code": "3",
					"Type": "11",
				},
			},
			wantNil:  false,
			wantCode: 3,
			wantType: 11,
		},
		{
			name:       "missing Icmp",
			properties: map[string]any{},
			wantNil:    true,
		},
		{
			name:       "nil Icmp",
			properties: map[string]any{"Icmp": nil},
			wantNil:    true,
		},
		{
			name:       "invalid type for Icmp",
			properties: map[string]any{"Icmp": "not a map"},
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIcmpTypeCode(tt.properties)
			if tt.wantNil {
				if got != nil {
					t.Errorf("extractIcmpTypeCode() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Fatal("extractIcmpTypeCode() = nil, want non-nil")
				}
				if *got.Code != tt.wantCode {
					t.Errorf("extractIcmpTypeCode().Code = %v, want %v", *got.Code, tt.wantCode)
				}
				if *got.Type != tt.wantType {
					t.Errorf("extractIcmpTypeCode().Type = %v, want %v", *got.Type, tt.wantType)
				}
			}
		})
	}
}

// TestExtractInt32Value tests int32 value extraction with error logging
func TestExtractInt32Value(t *testing.T) {
	tests := []struct {
		name          string
		value         any
		want          int32
		expectWarning bool
	}{
		{
			name:          "float64 value",
			value:         float64(42),
			want:          42,
			expectWarning: false,
		},
		{
			name:          "string value",
			value:         "123",
			want:          123,
			expectWarning: false,
		},
		{
			name:          "nil value",
			value:         nil,
			want:          0,
			expectWarning: false,
		},
		{
			name:          "invalid string",
			value:         "not-a-number",
			want:          0,
			expectWarning: true,
		},
		{
			name:          "unexpected type",
			value:         true,
			want:          0,
			expectWarning: true,
		},
		{
			name:          "negative number",
			value:         float64(-1),
			want:          -1,
			expectWarning: false,
		},
		{
			name:          "zero",
			value:         float64(0),
			want:          0,
			expectWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			got := extractInt32Value(tt.value)

			w.Close()
			os.Stderr = oldStderr

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if got != tt.want {
				t.Errorf("extractInt32Value() = %v, want %v", got, tt.want)
			}

			if tt.expectWarning && output == "" {
				t.Error("expected warning to stderr, got none")
			}
			if !tt.expectWarning && output != "" {
				t.Errorf("unexpected warning to stderr: %s", output)
			}
		})
	}
}
