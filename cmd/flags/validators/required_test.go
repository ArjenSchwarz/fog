package validators

import (
	"context"
	"testing"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
)

func TestRequiredFieldRule(t *testing.T) {
	rule := NewRequiredFieldRule("name", func(cmdflags.FlagValidator) interface{} { return "" })
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected error for empty field")
	}

	rule = NewRequiredFieldRule("name", func(cmdflags.FlagValidator) interface{} { return "value" })
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
