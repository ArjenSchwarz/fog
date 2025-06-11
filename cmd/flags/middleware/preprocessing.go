package middleware

import (
	"context"

	"github.com/ArjenSchwarz/fog/cmd/registry"
)

// PreprocessingMiddleware is a placeholder for flag preprocessing logic.
type PreprocessingMiddleware struct{}

// NewPreprocessingMiddleware creates a new PreprocessingMiddleware.
func NewPreprocessingMiddleware() *PreprocessingMiddleware { return &PreprocessingMiddleware{} }

// Execute runs preprocessing before the next handler.
func (m *PreprocessingMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
	// Preprocessing logic would be added here.
	return next(ctx)
}

// GetName returns the name of the middleware.
func (m *PreprocessingMiddleware) GetName() string {
	return "preprocessing"
}

var _ registry.Middleware = (*PreprocessingMiddleware)(nil)
