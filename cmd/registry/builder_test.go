package registry

import (
	"context"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

type testHandler struct{ calls *[]string }

func (h *testHandler) Execute(_ context.Context) error {
	*h.calls = append(*h.calls, "handler")
	return nil
}
func (h *testHandler) ValidateFlags() error { return nil }

type testValidator struct {
	registered bool
	calls      *[]string
}

func (v *testValidator) Validate() error {
	*v.calls = append(*v.calls, "validate")
	return nil
}
func (v *testValidator) RegisterFlags(cmd *cobra.Command) {
	v.registered = true
	cmd.Flags().Bool("x", false, "")
}

type testMiddleware struct {
	id    string
	calls *[]string
}

func (m testMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
	*m.calls = append(*m.calls, m.id+" before")
	err := next(ctx)
	*m.calls = append(*m.calls, m.id+" after")
	return err
}

func TestBaseCommandBuilder_Run(t *testing.T) {
	executionOrder := []string{}
	handler := &testHandler{calls: &executionOrder}
	validator := &testValidator{calls: &executionOrder}
	middleware1 := testMiddleware{id: "m1", calls: &executionOrder}
	middleware2 := testMiddleware{id: "m2", calls: &executionOrder}

	builder := NewBaseCommandBuilder("test", "", "").
		WithHandler(handler).
		WithValidator(validator).
		WithMiddleware(middleware1).
		WithMiddleware(middleware2)

	cmd := builder.BuildCommand()
	if !validator.registered {
		t.Errorf("validator did not register flags")
	}
	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("command run failed: %v", err)
	}

	expected := []string{"m1 before", "m2 before", "validate", "handler", "m2 after", "m1 after"}
	if !reflect.DeepEqual(executionOrder, expected) {
		t.Errorf("execution order mismatch\nwant %v\n got %v", expected, executionOrder)
	}
}
