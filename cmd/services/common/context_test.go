package common

import (
	"context"
	"testing"
)

func TestOperationContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithOperation(ctx, "deploy")
	if op := GetOperation(ctx); op != "deploy" {
		t.Fatalf("expected operation 'deploy', got '%s'", op)
	}
}

func TestComponentContext(t *testing.T) {
	ctx := context.Background()
	if comp := GetComponent(ctx); comp != "" {
		t.Fatalf("expected empty component, got '%s'", comp)
	}

	ctx = WithComponent(ctx, "service")
	if comp := GetComponent(ctx); comp != "service" {
		t.Fatalf("expected component 'service', got '%s'", comp)
	}
}
