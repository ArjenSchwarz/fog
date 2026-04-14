package cmd

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestNaclEntryToString_NilEgress verifies that naclEntryToString does not
// panic when the Egress field is nil, and defaults to "ingress".
func TestNaclEntryToString_NilEgress(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		RuleNumber: aws.Int32(100),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		// Egress intentionally nil — should default to ingress
	}
	result := naclEntryToString(entry)
	expected := "ingress #100 allow: 6, 10.0.0.0/8 Ports: All"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestNaclEntryToString_NilRuleNumber verifies that naclEntryToString does not
// panic when RuleNumber is nil, and shows "unknown" in place of the number.
func TestNaclEntryToString_NilRuleNumber(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(false),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		// RuleNumber intentionally nil
	}
	result := naclEntryToString(entry)
	if !strings.HasPrefix(result, "ingress #unknown") {
		t.Errorf("expected prefix %q, got %q", "ingress #unknown", result)
	}
}

// TestNaclEntryToString_NilProtocol verifies that naclEntryToString does not
// panic when Protocol is nil, and renders an empty protocol.
func TestNaclEntryToString_NilProtocol(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(true),
		RuleNumber: aws.Int32(200),
		RuleAction: ec2types.RuleActionDeny,
		CidrBlock:  aws.String("0.0.0.0/0"),
		// Protocol intentionally nil
	}
	result := naclEntryToString(entry)
	expected := "egress #200 deny: , 0.0.0.0/0 Ports: All"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestNaclEntryToString_NilPortRangeFields verifies that naclEntryToString
// handles a PortRange struct whose From and To fields are both nil,
// rendering "?" placeholders instead of misleading zero values.
func TestNaclEntryToString_NilPortRangeFields(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(false),
		RuleNumber: aws.Int32(100),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		PortRange:  &ec2types.PortRange{
			// From and To intentionally nil
		},
	}
	result := naclEntryToString(entry)
	if !strings.Contains(result, "Ports: ?-?") {
		t.Errorf("expected output to contain %q, got %q", "Ports: ?-?", result)
	}
}

// TestNaclEntryToString_PartialPortRange verifies that when only one side of
// PortRange is nil, the nil side renders as "?" rather than a misleading "0".
func TestNaclEntryToString_PartialPortRange(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(false),
		RuleNumber: aws.Int32(100),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		PortRange: &ec2types.PortRange{
			From: aws.Int32(80),
			// To intentionally nil
		},
	}
	result := naclEntryToString(entry)
	if !strings.Contains(result, "Ports: 80-?") {
		t.Errorf("expected output to contain %q, got %q", "Ports: 80-?", result)
	}
}

// TestNaclEntryToString_NilIcmpTypeCodeFields verifies that naclEntryToString
// handles an IcmpTypeCode struct whose Type and Code fields are both nil.
func TestNaclEntryToString_NilIcmpTypeCodeFields(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:       aws.Bool(false),
		RuleNumber:   aws.Int32(100),
		Protocol:     aws.String("1"),
		RuleAction:   ec2types.RuleActionAllow,
		CidrBlock:    aws.String("10.0.0.0/8"),
		IcmpTypeCode: &ec2types.IcmpTypeCode{
			// Type and Code intentionally nil
		},
	}
	result := naclEntryToString(entry)
	if !strings.Contains(result, "ICMP: unknown") {
		t.Errorf("expected output to contain %q, got %q", "ICMP: unknown", result)
	}
}

// TestNaclEntryToString_IcmpCodeNilTypeSet verifies that when IcmpTypeCode.Type
// is set but Code is nil, the output shows "?" for the missing code rather than
// a misleading "0".
func TestNaclEntryToString_IcmpCodeNilTypeSet(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(false),
		RuleNumber: aws.Int32(100),
		Protocol:   aws.String("1"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		IcmpTypeCode: &ec2types.IcmpTypeCode{
			Type: aws.Int32(5),
			// Code intentionally nil
		},
	}
	result := naclEntryToString(entry)
	if !strings.Contains(result, "ICMP: 5-?") {
		t.Errorf("expected output to contain %q, got %q", "ICMP: 5-?", result)
	}
}

// TestNaclEntryToString_AllNilFields verifies that naclEntryToString does not
// panic when every pointer field is nil.
func TestNaclEntryToString_AllNilFields(t *testing.T) {
	entry := ec2types.NetworkAclEntry{}
	result := naclEntryToString(entry)
	if !strings.HasPrefix(result, "ingress #unknown") {
		t.Errorf("expected prefix %q, got %q", "ingress #unknown", result)
	}
}

// TestNaclEntryToString_FullyPopulated ensures a complete entry still renders
// correctly (sanity check that guarding doesn't break the happy path).
func TestNaclEntryToString_FullyPopulated(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(true),
		RuleNumber: aws.Int32(100),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		PortRange: &ec2types.PortRange{
			From: aws.Int32(443),
			To:   aws.Int32(443),
		},
	}
	result := naclEntryToString(entry)
	expected := "egress #100 allow: 6, 10.0.0.0/8 Port: 443"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestCheckNaclEntryKey_NilEgress verifies that building a NACL entry key
// defaults to "I" (ingress) when Egress is nil.
func TestCheckNaclEntryKey_NilEgress(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		RuleNumber: aws.Int32(100),
	}
	result := naclEntryKey(entry)
	expected := "I100"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestCheckNaclEntryKey_NilRuleNumber verifies that building a NACL entry key
// defaults to "unknown" when RuleNumber is nil.
func TestCheckNaclEntryKey_NilRuleNumber(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress: aws.Bool(true),
	}
	result := naclEntryKey(entry)
	expected := "Eunknown"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestCheckNaclEntryKey_BothNil verifies that building a NACL entry key
// does not panic when both Egress and RuleNumber are nil.
func TestCheckNaclEntryKey_BothNil(t *testing.T) {
	entry := ec2types.NetworkAclEntry{}
	result := naclEntryKey(entry)
	expected := "Iunknown"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
