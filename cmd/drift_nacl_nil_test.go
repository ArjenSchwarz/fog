package cmd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// TestNaclEntryToString_NilEgress verifies that naclEntryToString does not
// panic when the Egress field is nil.
func TestNaclEntryToString_NilEgress(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		RuleNumber: aws.Int32(100),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		// Egress intentionally nil
	}
	result := naclEntryToString(entry)
	if result == "" {
		t.Error("expected non-empty string for entry with nil Egress")
	}
}

// TestNaclEntryToString_NilRuleNumber verifies that naclEntryToString does not
// panic when RuleNumber is nil.
func TestNaclEntryToString_NilRuleNumber(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(false),
		Protocol:   aws.String("6"),
		RuleAction: ec2types.RuleActionAllow,
		CidrBlock:  aws.String("10.0.0.0/8"),
		// RuleNumber intentionally nil
	}
	result := naclEntryToString(entry)
	if result == "" {
		t.Error("expected non-empty string for entry with nil RuleNumber")
	}
}

// TestNaclEntryToString_NilProtocol verifies that naclEntryToString does not
// panic when Protocol is nil.
func TestNaclEntryToString_NilProtocol(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress:     aws.Bool(true),
		RuleNumber: aws.Int32(200),
		RuleAction: ec2types.RuleActionDeny,
		CidrBlock:  aws.String("0.0.0.0/0"),
		// Protocol intentionally nil
	}
	result := naclEntryToString(entry)
	if result == "" {
		t.Error("expected non-empty string for entry with nil Protocol")
	}
}

// TestNaclEntryToString_NilPortRangeFields verifies that naclEntryToString
// handles a PortRange struct whose From or To fields are nil.
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
	if result == "" {
		t.Error("expected non-empty string for entry with nil PortRange fields")
	}
}

// TestNaclEntryToString_NilIcmpTypeCodeFields verifies that naclEntryToString
// handles an IcmpTypeCode struct whose Type or Code fields are nil.
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
	if result == "" {
		t.Error("expected non-empty string for entry with nil IcmpTypeCode fields")
	}
}

// TestNaclEntryToString_AllNilFields verifies that naclEntryToString does not
// panic when every pointer field is nil.
func TestNaclEntryToString_AllNilFields(t *testing.T) {
	entry := ec2types.NetworkAclEntry{}
	result := naclEntryToString(entry)
	if result == "" {
		t.Error("expected non-empty string for entry with all nil fields")
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
// does not panic when Egress is nil. We test this via naclEntryKey.
func TestCheckNaclEntryKey_NilEgress(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		RuleNumber: aws.Int32(100),
	}
	result := naclEntryKey(entry)
	if result == "" {
		t.Error("expected non-empty key for entry with nil Egress")
	}
}

// TestCheckNaclEntryKey_NilRuleNumber verifies that building a NACL entry key
// does not panic when RuleNumber is nil.
func TestCheckNaclEntryKey_NilRuleNumber(t *testing.T) {
	entry := ec2types.NetworkAclEntry{
		Egress: aws.Bool(true),
	}
	result := naclEntryKey(entry)
	if result == "" {
		t.Error("expected non-empty key for entry with nil RuleNumber")
	}
}

// TestCheckNaclEntryKey_BothNil verifies that building a NACL entry key
// does not panic when both Egress and RuleNumber are nil.
func TestCheckNaclEntryKey_BothNil(t *testing.T) {
	entry := ec2types.NetworkAclEntry{}
	result := naclEntryKey(entry)
	if result == "" {
		t.Error("expected non-empty key for entry with all nil fields")
	}
}
