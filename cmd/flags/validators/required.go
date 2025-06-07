package validators

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ArjenSchwarz/fog/cmd/flags"
)

// RequiredFieldRule validates that required fields are not empty
type RequiredFieldRule struct {
	FieldName   string
	GetValue    func(flags.FlagValidator) interface{}
	Description string
}

// NewRequiredFieldRule creates a new required field rule
func NewRequiredFieldRule(fieldName string, getValue func(flags.FlagValidator) interface{}) *RequiredFieldRule {
	return &RequiredFieldRule{
		FieldName:   fieldName,
		GetValue:    getValue,
		Description: fmt.Sprintf("%s is required", fieldName),
	}
}

// Validate checks if the field has a value
func (r *RequiredFieldRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	value := r.GetValue(flags)
	if isEmpty(value) {
		return fmt.Errorf("required field '%s' is missing or empty", r.FieldName)
	}
	return nil
}

// GetDescription returns the rule description
func (r *RequiredFieldRule) GetDescription() string { return r.Description }

// GetSeverity returns the rule severity
func (r *RequiredFieldRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityError }

// isEmpty checks if a value is considered empty
func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Slice, reflect.Array:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
