package deploy

import (
	"context"
	"testing"
)

// TestValidateFlags verifies that ValidateFlags returns any errors from the Flags
// validation logic.
func TestValidateFlags(t *testing.T) {
	h := NewHandler(&Flags{StackName: "test"})
	if err := h.ValidateFlags(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	h = NewHandler(&Flags{})
	if err := h.ValidateFlags(); err == nil {
		t.Fatalf("expected validation error when stack name missing")
	}
}

// TestExecute verifies that Execute currently returns the not implemented error.
func TestExecute(t *testing.T) {
	h := NewHandler(&Flags{StackName: "test"})
	err := h.Execute(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
}
