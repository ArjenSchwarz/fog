package middleware

import (
	"context"

	"github.com/ArjenSchwarz/fog/cmd/flags"
	"github.com/ArjenSchwarz/fog/cmd/registry"
)

// FlagValidationMiddleware runs flag validation before invoking the next handler.
type FlagValidationMiddleware struct {
	validator flags.FlagValidator
}

// NewFlagValidationMiddleware returns a new FlagValidationMiddleware.
func NewFlagValidationMiddleware(validator flags.FlagValidator) *FlagValidationMiddleware {
	return &FlagValidationMiddleware{validator: validator}
}

// Execute validates flags and then calls the next handler.
func (m *FlagValidationMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
	if err := m.validator.Validate(ctx, &flags.ValidationContext{}); err != nil {
		return err
	}
	return next(ctx)
}

var _ registry.Middleware = (*FlagValidationMiddleware)(nil)
