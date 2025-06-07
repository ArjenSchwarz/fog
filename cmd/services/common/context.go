package common

import "context"

// contextKey is used for values stored in a context.Context.
type contextKey string

const (
	operationKey contextKey = "operation"
	componentKey contextKey = "component"
)

// WithOperation returns a new context carrying the operation name.
func WithOperation(ctx context.Context, operation string) context.Context {
	return context.WithValue(ctx, operationKey, operation)
}

// GetOperation retrieves the operation name from the context.
func GetOperation(ctx context.Context) string {
	if v, ok := ctx.Value(operationKey).(string); ok {
		return v
	}
	return ""
}

// WithComponent returns a new context carrying the component name.
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, componentKey, component)
}

// GetComponent retrieves the component from the context.
func GetComponent(ctx context.Context) string {
	if v, ok := ctx.Value(componentKey).(string); ok {
		return v
	}
	return ""
}
