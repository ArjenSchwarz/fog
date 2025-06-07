package validators

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ArjenSchwarz/fog/cmd/flags"
)

// AWSRegionRule validates that a provided region is a valid AWS region
// It is a basic regex validation and does not check actual availability.
type AWSRegionRule struct {
	FieldName   string
	GetValue    func(flags.FlagValidator) string
	Description string
}

// NewAWSRegionRule creates a new AWS region validation rule
func NewAWSRegionRule(fieldName string, getValue func(flags.FlagValidator) string) *AWSRegionRule {
	return &AWSRegionRule{
		FieldName:   fieldName,
		GetValue:    getValue,
		Description: fmt.Sprintf("%s must be a valid AWS region", fieldName),
	}
}

var regionPattern = regexp.MustCompile(`^[a-z]{2}(?:-gov)?-[a-z0-9-]+-\d$`)

// Validate checks the AWS region format
func (a *AWSRegionRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	value := a.GetValue(flags)
	if value == "" {
		return nil
	}
	if !regionPattern.MatchString(value) {
		return fmt.Errorf("invalid AWS region '%s' for '%s'", value, a.FieldName)
	}
	return nil
}

// GetDescription returns the rule description
func (a *AWSRegionRule) GetDescription() string { return a.Description }

// GetSeverity returns the rule severity
func (a *AWSRegionRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityError }
