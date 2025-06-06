package middleware

import (
	"context"
	"testing"
)

// TestValidationMiddlewareExecute ensures the middleware forwards execution
// to the next handler without modifying the context or returning an error.
func TestValidationMiddlewareExecute(t *testing.T) {
	mw := NewValidationMiddleware()

	called := false
	next := func(ctx context.Context) error {
		called = true
		if ctx == nil {
			t.Errorf("context should not be nil")
		}
		return nil
	}

	if err := mw.Execute(context.Background(), next); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !called {
		t.Errorf("next function was not called")
	}
}
