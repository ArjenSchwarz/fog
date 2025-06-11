package middleware

import (
	"context"

	"github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/registry"
	"github.com/ArjenSchwarz/fog/cmd/ui"
)

// ErrorHandlingMiddleware provides centralized error handling.
type ErrorHandlingMiddleware struct {
	formatter errors.ErrorFormatter
	ui        ui.OutputHandler
}

// NewErrorHandlingMiddleware creates a new error handling middleware.
func NewErrorHandlingMiddleware(formatter errors.ErrorFormatter, ui ui.OutputHandler) *ErrorHandlingMiddleware {
	return &ErrorHandlingMiddleware{
		formatter: formatter,
		ui:        ui,
	}
}

// Execute handles errors from command execution.
func (m *ErrorHandlingMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
	err := next(ctx)
	if err == nil {
		return nil
	}

	// Convert to FogError if needed.
	var fogErr errors.FogError
	if fe, ok := err.(errors.FogError); ok {
		fogErr = fe
	} else {
		errorCtx := errors.GetErrorContext(ctx)
		fogErr = errors.WrapError(errorCtx, err, errors.ErrUnknown, "Unknown error occurred")
	}

	// Format and display the error.
	m.displayError(fogErr)

	// Return the original error so exit codes propagate.
	return err
}

// GetName returns the name of the middleware.
func (m *ErrorHandlingMiddleware) GetName() string {
	return "error_handler"
}

// displayError formats and displays an error.
func (m *ErrorHandlingMiddleware) displayError(err errors.FogError) {
	if multiErr, ok := err.(*errors.MultiError); ok {
		m.ui.Error(m.formatter.FormatMultiError(multiErr))
		return
	}

	formatted := m.formatter.FormatError(err)

	switch err.Severity() {
	case errors.SeverityCritical, errors.SeverityHigh:
		m.ui.Error(formatted)
	case errors.SeverityMedium:
		m.ui.Warning(formatted)
	case errors.SeverityLow:
		m.ui.Info(formatted)
	default:
		m.ui.Error(formatted)
	}

	if m.ui.GetVerbose() && err.StackTrace() != nil {
		m.ui.Debug("Stack trace:")
		for _, frame := range err.StackTrace() {
			m.ui.Debug("  " + frame)
		}
	}
}

var _ registry.Middleware = (*ErrorHandlingMiddleware)(nil)
