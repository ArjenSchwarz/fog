package validators

import (
	"context"
	"testing"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
)

func TestConflictRule(t *testing.T) {
	get := func(cmdflags.FlagValidator, string) interface{} { return "" }
	rule := NewConflictRule([]string{"a", "b"}, get)
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error when none set: %v", err)
	}

	get = func(_ cmdflags.FlagValidator, field string) interface{} {
		if field == "a" {
			return "val"
		}
		return ""
	}
	rule = NewConflictRule([]string{"a", "b"}, get)
	if err := rule.Validate(context.Background(), nil, nil); err != nil {
		t.Fatalf("unexpected error when one set: %v", err)
	}

	get = func(_ cmdflags.FlagValidator, field string) interface{} { return "val" }
	rule = NewConflictRule([]string{"a", "b"}, get)
	if err := rule.Validate(context.Background(), nil, nil); err == nil {
		t.Fatalf("expected conflict error")
	}
}
