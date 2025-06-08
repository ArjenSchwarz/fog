package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/flags"
	"github.com/spf13/cobra"
)

type mockValidator struct {
	err    error
	called bool
}

func (m *mockValidator) Validate(ctx context.Context, vCtx *flags.ValidationContext) error {
	m.called = true
	return m.err
}
func (m *mockValidator) RegisterFlags(cmd *cobra.Command)           {}
func (m *mockValidator) GetValidationRules() []flags.ValidationRule { return nil }

func TestFlagValidationMiddlewareExecuteSuccess(t *testing.T) {
	mv := &mockValidator{}
	mw := NewFlagValidationMiddleware(mv)

	nextCalled := false
	next := func(ctx context.Context) error { nextCalled = true; return nil }

	if err := mw.Execute(context.Background(), next); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mv.called {
		t.Errorf("validator not called")
	}
	if !nextCalled {
		t.Errorf("next not called")
	}
}

func TestFlagValidationMiddlewareExecuteError(t *testing.T) {
	mv := &mockValidator{err: errors.New("fail")}
	mw := NewFlagValidationMiddleware(mv)

	called := false
	next := func(ctx context.Context) error { called = true; return nil }

	err := mw.Execute(context.Background(), next)
	if err == nil || err.Error() != "fail" {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !mv.called {
		t.Errorf("validator not called")
	}
	if called {
		t.Errorf("next should not run on validation failure")
	}
}
