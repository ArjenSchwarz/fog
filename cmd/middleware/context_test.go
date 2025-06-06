package middleware

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ArjenSchwarz/fog/config"
)

// TestContextMiddlewareExecute verifies that the middleware loads the config and
// stores it on the context before calling the next handler.
func TestContextMiddlewareExecute(t *testing.T) {
	cfg := &config.Config{}
	loaderCalled := false
	mw := NewContextMiddleware(func() (*config.Config, error) {
		loaderCalled = true
		return cfg, nil
	})

	nextCalled := false
	next := func(ctx context.Context) error {
		nextCalled = true
		got := ctx.Value(configKey)
		if got != cfg {
			t.Errorf("config not passed to context")
		}
		return nil
	}

	if err := mw.Execute(context.Background(), next); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !loaderCalled {
		t.Errorf("config loader was not called")
	}
	if !nextCalled {
		t.Errorf("next handler was not called")
	}
}

// TestContextMiddlewareExecuteError verifies that errors from the loader are
// returned and that the next handler is not called.
func TestContextMiddlewareExecuteError(t *testing.T) {
	mw := NewContextMiddleware(func() (*config.Config, error) {
		return nil, fmt.Errorf("load failure")
	})

	called := false
	next := func(ctx context.Context) error {
		called = true
		return nil
	}

	err := mw.Execute(context.Background(), next)
	if err == nil || !strings.Contains(err.Error(), "load failure") {
		t.Fatalf("expected loader error, got %v", err)
	}
	if called {
		t.Errorf("next should not be called on loader failure")
	}
}
