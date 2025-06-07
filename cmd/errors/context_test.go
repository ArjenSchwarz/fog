package errors

import (
	"context"
	"fmt"
	"reflect"
	"testing"
)

func TestErrorContextToMap(t *testing.T) {
	ctx := NewErrorContext("deploy", "cmd").
		WithStackName("stack").
		WithTemplate("tmpl").
		WithRegion("us-west-2").
		WithAccount("123").
		WithRequestID("req").
		WithCorrelationID("corr").
		WithField("custom", 42)

	m := ctx.ToMap()
	expected := map[string]interface{}{
		"operation":      "deploy",
		"component":      "cmd",
		"stack_name":     "stack",
		"template_path":  "tmpl",
		"region":         "us-west-2",
		"account":        "123",
		"request_id":     "req",
		"correlation_id": "corr",
		"custom":         42,
	}
	if !reflect.DeepEqual(m, expected) {
		t.Fatalf("unexpected map: %#v", m)
	}
}

func TestWithGetErrorContext(t *testing.T) {
	ec := NewErrorContext("op", "comp")
	c := WithErrorContext(context.Background(), ec)
	got := GetErrorContext(c)
	if got != ec {
		t.Fatalf("expected same context back")
	}

	missing := GetErrorContext(context.Background())
	if missing.Operation != "unknown" || missing.Component != "unknown" {
		t.Fatalf("expected unknown context, got %#v", missing)
	}
}

func TestContextualError(t *testing.T) {
	ec := NewErrorContext("op", "comp").WithField("k", "v")
	err := ContextualError(ec, "CODE", "msg")
	be, ok := err.(*BaseError)
	if !ok {
		t.Fatalf("unexpected type %T", err)
	}
	if be.Code() != "CODE" || be.Message() != "msg" {
		t.Fatalf("incorrect code or message")
	}
	if be.Operation() != "op" || be.Component() != "comp" {
		t.Fatalf("context not applied")
	}
	if be.Fields()["k"] != "v" {
		t.Fatalf("field not applied")
	}
}

func TestWrapError(t *testing.T) {
	base := fmt.Errorf("fail")
	ec := NewErrorContext("op", "comp")
	err := WrapError(ec, base, "CODE", "msg")
	be, ok := err.(*BaseError)
	if !ok {
		t.Fatalf("unexpected type %T", err)
	}
	if be.Cause() != base {
		t.Fatalf("cause not set")
	}
}
