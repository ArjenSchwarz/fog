package middleware

import (
	"context"
	"fmt"

	"github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/registry"
	"github.com/ArjenSchwarz/fog/cmd/ui"
)

// RecoveryMiddleware handles panics and converts them to errors.
type RecoveryMiddleware struct {
	ui ui.OutputHandler
}

// NewRecoveryMiddleware creates a new recovery middleware.
func NewRecoveryMiddleware(ui ui.OutputHandler) *RecoveryMiddleware {
	return &RecoveryMiddleware{ui: ui}
}

// Execute handles panic recovery.
func (m *RecoveryMiddleware) Execute(ctx context.Context, next func(context.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			errorCtx := errors.GetErrorContext(ctx)
			panicErr := func() errors.FogError {
				be := errors.ContextualError(
					errorCtx,
					errors.ErrInternal,
					fmt.Sprintf("Internal error (panic): %v", r),
				).(*errors.BaseError)
				return be.WithStackTrace()
			}()

			m.ui.Error(fmt.Sprintf("Internal error occurred: %v", r))
			if m.ui.GetVerbose() {
				m.ui.Debug("This is likely a bug. Please report it with the following information:")
				for _, frame := range panicErr.StackTrace() {
					m.ui.Debug("  " + frame)
				}
			}

			err = panicErr
		}
	}()

	return next(ctx)
}

var _ registry.Middleware = (*RecoveryMiddleware)(nil)
