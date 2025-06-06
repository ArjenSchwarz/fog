package registry

import (
	"context"

	"github.com/spf13/cobra"
)

// contextKey is the type used for values stored in context.Context.
// Using a custom type prevents collisions with keys from other packages.
type contextKey string

const (
	commandCtxKey contextKey = "command"
	argsCtxKey    contextKey = "args"
)

// Middleware defines a command middleware.
type Middleware interface {
	Execute(ctx context.Context, next func(context.Context) error) error
}

// BaseCommandBuilder provides common logic for command builders.
type BaseCommandBuilder struct {
	name        string
	short       string
	long        string
	handler     CommandHandler
	validator   FlagValidator
	middlewares []Middleware
}

// NewBaseCommandBuilder creates a new builder with the given name and descriptions.
func NewBaseCommandBuilder(name, short, long string) *BaseCommandBuilder {
	return &BaseCommandBuilder{
		name:        name,
		short:       short,
		long:        long,
		middlewares: make([]Middleware, 0),
	}
}

// WithHandler assigns the command handler.
func (b *BaseCommandBuilder) WithHandler(handler CommandHandler) *BaseCommandBuilder {
	b.handler = handler
	return b
}

// WithValidator assigns a flag validator.
func (b *BaseCommandBuilder) WithValidator(validator FlagValidator) *BaseCommandBuilder {
	b.validator = validator
	return b
}

// WithMiddleware appends a middleware to the chain.
func (b *BaseCommandBuilder) WithMiddleware(m Middleware) *BaseCommandBuilder {
	b.middlewares = append(b.middlewares, m)
	return b
}

// BuildCommand constructs the cobra command including flags and middleware chain.
func (b *BaseCommandBuilder) BuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   b.name,
		Short: b.short,
		Long:  b.long,
		RunE:  b.createRunFunc(),
	}

	if b.validator != nil {
		b.validator.RegisterFlags(cmd)
	}

	return cmd
}

// GetHandler returns the associated handler.
func (b *BaseCommandBuilder) GetHandler() CommandHandler { return b.handler }

func (b *BaseCommandBuilder) createRunFunc() func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := context.WithValue(context.Background(), commandCtxKey, cmd)
		ctx = context.WithValue(ctx, argsCtxKey, args)

		handler := func(ctx context.Context) error {
			if b.validator != nil {
				if err := b.validator.Validate(); err != nil {
					return err
				}
			}
			if b.handler == nil {
				return nil
			}
			return b.handler.Execute(ctx)
		}

		for i := len(b.middlewares) - 1; i >= 0; i-- {
			mw := b.middlewares[i]
			next := handler
			handler = func(ctx context.Context) error {
				return mw.Execute(ctx, next)
			}
		}

		return handler(ctx)
	}
}
