package middleware

import (
	"context"

	"github.com/ArjenSchwarz/fog/config"
)

// ctxKey is used for storing values in context.
type ctxKey string

// configKey is the key for config objects in context.
const configKey ctxKey = "config"

// ContextMiddleware loads configuration and attaches it to the context.
type ContextMiddleware struct {
	configLoader func() (*config.Config, error)
}

// NewContextMiddleware returns a new ContextMiddleware that uses the provided loader.
func NewContextMiddleware(loader func() (*config.Config, error)) *ContextMiddleware {
	return &ContextMiddleware{configLoader: loader}
}

// Execute loads configuration and passes the updated context to the next handler.
func (m *ContextMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
	cfg, err := m.configLoader()
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, configKey, cfg)
	return next(ctx)
}
