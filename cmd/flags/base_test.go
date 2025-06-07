package flags

import (
	"context"
	"errors"
	"testing"
)

type mockRule struct {
	severity ValidationSeverity
	err      error
}

func (m mockRule) Validate(ctx context.Context, flags FlagValidator, vCtx *ValidationContext) error {
	return m.err
}

func (m mockRule) GetDescription() string { return "mock" }

func (m mockRule) GetSeverity() ValidationSeverity { return m.severity }

func TestBaseFlagValidator_ValidateAggregatesErrors(t *testing.T) {
	b := NewBaseFlagValidator()
	err1 := errors.New("first")
	err2 := errors.New("second")

	b.AddRule(mockRule{severity: SeverityError, err: err1})
	b.AddRule(mockRule{severity: SeverityError, err: err2})

	err := b.Validate(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("unexpected error type %T", err)
	}
	if !errors.Is(vErr.Err, err1) || !errors.Is(vErr.Err, err2) {
		t.Errorf("aggregated error missing sub-errors: %v", vErr.Err)
	}
}

func TestBaseFlagValidator_ValidateWarningsReturned(t *testing.T) {
	b := NewBaseFlagValidator()
	b.AddRule(mockRule{severity: SeverityWarning, err: errors.New("warn")})

	err := b.Validate(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected warning error, got nil")
	}
	vErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("unexpected error type %T", err)
	}
	if len(vErr.Warnings) != 1 || vErr.Warnings[0] != "warn" {
		t.Errorf("unexpected warnings: %v", vErr.Warnings)
	}
	if vErr.Err != nil {
		t.Errorf("expected no error severity, got %v", vErr.Err)
	}
}
