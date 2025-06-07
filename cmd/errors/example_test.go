package errors_test

import (
	"fmt"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/validation"
)

func TestErrorHandlingExample(t *testing.T) {
	// Example 1: Creating a basic error
	basicErr := errors.NewError(errors.ErrStackNotFound, "Stack 'my-stack' not found").
		WithOperation("describe").
		WithComponent("cloudformation").
		WithUserMessage("The CloudFormation stack 'my-stack' could not be found").
		WithSuggestions([]string{
			"Check that the stack name is correct",
			"Verify you're in the correct AWS region",
		})

	// Add additional fields
	enrichedErr := basicErr.WithFields(map[string]interface{}{
		"stack_name": "my-stack",
		"region":     "us-east-1",
	})

	if enrichedErr.Code() != errors.ErrStackNotFound {
		t.Errorf("Expected error code %s, got %s", errors.ErrStackNotFound, enrichedErr.Code())
	}

	if enrichedErr.Category() != errors.CategoryAWS {
		t.Errorf("Expected category %d, got %d", errors.CategoryAWS, enrichedErr.Category())
	}

	// Example 2: Using error context
	ctx := errors.NewErrorContext("deploy", "command").
		WithStackName("my-stack").
		WithTemplate("template.yaml").
		WithRegion("us-east-1")

	contextErr := errors.ContextualError(ctx, errors.ErrTemplateInvalid, "Template validation failed")

	if contextErr.Operation() != "deploy" {
		t.Errorf("Expected operation 'deploy', got '%s'", contextErr.Operation())
	}

	// Example 3: Using validation error builder
	validator := validation.NewValidationErrorBuilder("flag-validation")
	validator.RequiredField("stack-name").
		InvalidValue("region", "invalid-region", "not a valid AWS region").
		ConflictingFlags([]string{"template", "deployment-file"})

	if !validator.HasErrors() {
		t.Error("Expected validation errors")
	}

	err := validator.Build()
	if err == nil {
		t.Error("Expected validation error to be built")
	}

	// Example 4: Error formatting
	formatter := errors.NewConsoleErrorFormatter(false, false) // no color, not verbose
	formatted := formatter.FormatError(basicErr)

	if formatted == "" {
		t.Error("Expected formatted error output")
	}

	fmt.Printf("Formatted error:\n%s\n", formatted)
}

func TestErrorMetadata(t *testing.T) {
	// Test error metadata
	metadata := errors.GetErrorMetadata(errors.ErrStackNotFound)

	if metadata.Code != errors.ErrStackNotFound {
		t.Errorf("Expected code %s, got %s", errors.ErrStackNotFound, metadata.Code)
	}

	if metadata.Category != errors.CategoryAWS {
		t.Errorf("Expected category %d, got %d", errors.CategoryAWS, metadata.Category)
	}

	if len(metadata.Suggestions) == 0 {
		t.Error("Expected suggestions to be provided")
	}

	fmt.Printf("Error metadata: %+v\n", metadata)
}

func TestErrorAggregation(t *testing.T) {
	// Test error aggregation
	aggregator := errors.NewErrorAggregator("deployment-preparation")

	err1 := errors.NewError(errors.ErrFileNotFound, "Template file not found")
	err2 := errors.NewError(errors.ErrParameterMissing, "Required parameter missing")
	err3 := errors.NewError(errors.ErrInvalidCredentials, "AWS credentials invalid")

	aggregator.Add(err1)
	aggregator.Add(err2)
	aggregator.Add(err3)

	if aggregator.Count() != 3 {
		t.Errorf("Expected 3 errors, got %d", aggregator.Count())
	}

	multiErr := aggregator.ToError()
	if multiErr == nil {
		t.Error("Expected aggregated error")
	}

	if multiError, ok := multiErr.(*errors.MultiError); ok {
		if len(multiError.Errors()) != 3 {
			t.Errorf("Expected 3 individual errors, got %d", len(multiError.Errors()))
		}
	} else {
		t.Errorf("Expected MultiError, got %T", multiErr)
	}
}
