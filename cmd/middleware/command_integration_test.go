package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/ArjenSchwarz/fog/cmd/registry"
	"github.com/ArjenSchwarz/fog/cmd/validation"
	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/cobra"
)

// failingValidator returns a validation.MultiError when Validate is called.
type failingValidator struct{}

func (failingValidator) Validate() error {
	vb := validation.NewValidationErrorBuilder("test")
	vb.RequiredField("name")
	vb.InvalidValue("region", "", "invalid")
	return vb.Build()
}
func (failingValidator) RegisterFlags(cmd *cobra.Command) {}

// recordingHandler records whether Execute was called and if the config was present.
type recordingHandler struct {
	sawConfig bool
	cfg       *config.Config
}

func (h *recordingHandler) Execute(ctx context.Context) error {
	if ctx.Value(configKey) == h.cfg {
		h.sawConfig = true
	}
	return errors.New("service failed")
}
func (h *recordingHandler) ValidateFlags() error { return nil }

// passValidator does nothing and always succeeds.
type passValidator struct{}

func (passValidator) Validate() error                  { return nil }
func (passValidator) RegisterFlags(cmd *cobra.Command) {}

func buildTestRoot(builder registry.CommandBuilder) *cobra.Command {
	root := &cobra.Command{Use: "root"}
	root.AddCommand(builder.BuildCommand())
	return root
}

// TestCommandValidationError ensures validation errors are formatted and the handler is not called.
func TestCommandValidationError(t *testing.T) {
	formatter := &stubFormatter{}
	ui := &stubUI{}
	errMw := NewErrorHandlingMiddleware(formatter, ui)
	cfg := &config.Config{}
	ctxMw := NewContextMiddleware(func() (*config.Config, error) { return cfg, nil })

	handler := &recordingHandler{cfg: cfg}

	builder := registry.NewBaseCommandBuilder("test", "", "").
		WithHandler(handler).
		WithValidator(failingValidator{}).
		WithMiddleware(ctxMw).
		WithMiddleware(errMw)

	root := buildTestRoot(builder)
	root.SetArgs([]string{"test"})
	err := root.Execute()

	if err == nil {
		t.Fatalf("expected error from command")
	}
	if handler.sawConfig {
		t.Errorf("handler should not run on validation failure")
	}
	if formatter.multiFormatted == "" {
		t.Errorf("validation error not formatted")
	}
	if len(ui.errors) == 0 || ui.errors[0] != formatter.multiFormatted {
		t.Errorf("formatted validation error not output")
	}
}

// TestCommandServiceFailure verifies that service errors are formatted and that context middleware passes config.
func TestCommandServiceFailure(t *testing.T) {
	formatter := &stubFormatter{}
	ui := &stubUI{}
	errMw := NewErrorHandlingMiddleware(formatter, ui)
	cfg := &config.Config{}
	ctxMw := NewContextMiddleware(func() (*config.Config, error) { return cfg, nil })

	handler := &recordingHandler{cfg: cfg}

	builder := registry.NewBaseCommandBuilder("test", "", "").
		WithHandler(handler).
		WithValidator(passValidator{}).
		WithMiddleware(ctxMw).
		WithMiddleware(errMw)

	root := buildTestRoot(builder)
	root.SetArgs([]string{"test"})
	err := root.Execute()

	if err == nil || err.Error() != "service failed" {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handler.sawConfig {
		t.Errorf("config not propagated to handler")
	}
	if formatter.formatted == "" {
		t.Errorf("error was not formatted")
	}
	if len(ui.infos) == 0 || ui.infos[0] != formatter.formatted {
		t.Errorf("formatted error not output")
	}
}
