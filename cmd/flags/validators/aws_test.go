package validators

import (
	"context"
	"testing"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
)

func TestAWSRegionRule(t *testing.T) {
	rule := NewAWSRegionRule("region", func(cmdflags.FlagValidator) string { return "" })
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for empty region: %v", err)
	}

	rule = NewAWSRegionRule("region", func(cmdflags.FlagValidator) string { return "us-west-2" })
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error for valid region: %v", err)
	}

	rule = NewAWSRegionRule("region", func(cmdflags.FlagValidator) string { return "invalid" })
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected region error")
	}
}
