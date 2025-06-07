package common

import (
	"errors"
	"testing"
)

func TestServiceError(t *testing.T) {
	base := errors.New("base")
	err := Wrap(ErrCodeAWS, "failed", base)

	if err.Code != ErrCodeAWS {
		t.Fatalf("unexpected code %s", err.Code)
	}

	if err.Unwrap() != base {
		t.Fatalf("unwrap did not return base error")
	}

	expected := "failed: base"
	if err.Error() != expected {
		t.Fatalf("expected '%s', got '%s'", expected, err.Error())
	}
}
