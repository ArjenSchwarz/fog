package errors

import "testing"

// TestBaseErrorBasic verifies that NewError sets basic fields correctly.
func TestBaseErrorBasic(t *testing.T) {
	err := NewError(ErrUnknown, "msg")
	if err.Code() != ErrUnknown {
		t.Fatalf("code mismatch: %v", err.Code())
	}
	if err.Message() != "msg" {
		t.Fatalf("message mismatch: %s", err.Message())
	}
	if err.Timestamp().IsZero() {
		t.Fatalf("timestamp not set")
	}
	if err.Category() != CategoryUnknown {
		t.Fatalf("unexpected category: %v", err.Category())
	}
	if err.Severity() != SeverityLow {
		t.Fatalf("unexpected severity: %v", err.Severity())
	}
	if err.Retryable() {
		t.Fatalf("expected not retryable")
	}
}

// TestBaseErrorFields ensures fields are copied when adding.
func TestBaseErrorFields(t *testing.T) {
	err := NewError(ErrUnknown, "msg")
	err.fields["a"] = 1
	e2 := err.WithField("b", 2).(*BaseError)

	if len(err.fields) != 1 {
		t.Fatalf("original fields changed")
	}
	if len(e2.fields) != 2 || e2.fields["a"] != 1 || e2.fields["b"] != 2 {
		t.Fatalf("fields not set correctly")
	}
}

// TestStackTrace ensures stack trace is captured.
func TestStackTrace(t *testing.T) {
	err := NewError(ErrUnknown, "msg").WithStackTrace()
	if len(err.StackTrace()) == 0 {
		t.Fatalf("stack trace not captured")
	}
}

// TestErrorAggregator verifies aggregator behavior.
func TestErrorAggregator(t *testing.T) {
	agg := NewErrorAggregator("ctx")
	if agg.HasErrors() {
		t.Fatalf("expected no errors")
	}

	e1 := NewError(ErrUnknown, "one")
	e2 := NewError(ErrUnknown, "two")
	agg.Add(e1)
	agg.Add(e2)

	if agg.Count() != 2 {
		t.Fatalf("count mismatch")
	}
	if agg.FirstError() != e1 {
		t.Fatalf("first error mismatch")
	}

	if err := agg.ToError(); err == nil {
		t.Fatalf("expected error")
	} else if me, ok := err.(*MultiError); !ok || len(me.Errors()) != 2 {
		t.Fatalf("unexpected ToError result: %#v", err)
	}
}

// TestMultiErrorError ensures error message formatting.
func TestMultiErrorError(t *testing.T) {
	e1 := NewError(ErrUnknown, "one")
	e2 := NewError(ErrUnknown, "two")
	me := NewMultiError("test", []FogError{e1, e2})
	if me.Error() == "" {
		t.Fatalf("empty error string")
	}
	meSingle := NewMultiError("test", []FogError{e1})
	if meSingle.Error() != e1.Error() {
		t.Fatalf("single error formatting incorrect")
	}
}
