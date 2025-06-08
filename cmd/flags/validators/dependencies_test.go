package validators

import (
	"context"
	"testing"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
)

func TestDependencyRule(t *testing.T) {
	getTrigger := func(cmdflags.FlagValidator) interface{} { return "" }
	getDep := func(cmdflags.FlagValidator, string) interface{} { return "" }
	rule := NewDependencyRule("a", []string{"b"}, getTrigger, getDep)
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error when trigger empty: %v", err)
	}

	getTrigger = func(cmdflags.FlagValidator) interface{} { return "val" }
	getDep = func(cmdflags.FlagValidator, string) interface{} { return "" }
	rule = NewDependencyRule("a", []string{"b"}, getTrigger, getDep)
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected error when dependency missing")
	}

	getDep = func(cmdflags.FlagValidator, string) interface{} { return "x" }
	rule = NewDependencyRule("a", []string{"b"}, getTrigger, getDep)
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
