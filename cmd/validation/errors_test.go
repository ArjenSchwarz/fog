package validation

import (
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/errors"
)

func TestValidationErrorBuilder(t *testing.T) {
	builder := NewValidationErrorBuilder("test-operation")

	// Test building multiple validation errors
	builder.RequiredField("stack-name").
		InvalidValue("region", "invalid-region", "not a valid AWS region").
		ConflictingFlags([]string{"template", "deployment-file"}).
		FileNotFound("template", "/path/to/nonexistent.yaml")

	if !builder.HasErrors() {
		t.Error("Expected validation errors to be present")
	}

	if builder.ErrorCount() != 4 {
		t.Errorf("Expected 4 errors, got %d", builder.ErrorCount())
	}

	// Test building the error
	err := builder.Build()
	if err == nil {
		t.Error("Expected error to be built")
	}

	// Test that it's a multi-error
	if multiErr, ok := err.(*errors.MultiError); ok {
		if len(multiErr.Errors()) != 4 {
			t.Errorf("Expected 4 individual errors, got %d", len(multiErr.Errors()))
		}

		// Check that the first error has the right code
		firstErr := multiErr.Errors()[0]
		if firstErr.Code() != errors.ErrRequiredField {
			t.Errorf("Expected first error to be ErrRequiredField, got %s", firstErr.Code())
		}

		// Check that the error has the right operation context
		if firstErr.Operation() != "test-operation" {
			t.Errorf("Expected operation to be 'test-operation', got '%s'", firstErr.Operation())
		}

		// Check that the error has the right component context
		if firstErr.Component() != "validation" {
			t.Errorf("Expected component to be 'validation', got '%s'", firstErr.Component())
		}
	} else {
		t.Errorf("Expected MultiError, got %T", err)
	}
}

func TestValidationErrorBuilder_NoErrors(t *testing.T) {
	builder := NewValidationErrorBuilder("test-operation")

	if builder.HasErrors() {
		t.Error("Expected no validation errors")
	}

	if builder.ErrorCount() != 0 {
		t.Errorf("Expected 0 errors, got %d", builder.ErrorCount())
	}

	err := builder.Build()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidationErrorBuilder_SingleError(t *testing.T) {
	builder := NewValidationErrorBuilder("test-operation")
	builder.RequiredField("stack-name")

	if !builder.HasErrors() {
		t.Error("Expected validation errors to be present")
	}

	if builder.ErrorCount() != 1 {
		t.Errorf("Expected 1 error, got %d", builder.ErrorCount())
	}

	err := builder.Build()
	if err == nil {
		t.Error("Expected error to be built")
	}

	// With a single error, it should return the error directly, not wrapped in MultiError
	if fogErr, ok := err.(errors.FogError); ok {
		if fogErr.Code() != errors.ErrRequiredField {
			t.Errorf("Expected error code to be ErrRequiredField, got %s", fogErr.Code())
		}
	} else {
		t.Errorf("Expected FogError, got %T", err)
	}
}
