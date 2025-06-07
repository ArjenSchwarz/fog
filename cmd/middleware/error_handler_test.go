package middleware

import (
	"context"
	"errors"
	"io"
	"testing"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/ui"
)

// stubFormatter is a simple formatter used for tests.
type stubFormatter struct {
	formatted      string
	multiFormatted string
	lastErr        ferr.FogError
}

func (s *stubFormatter) FormatError(err ferr.FogError) string {
	s.lastErr = err
	s.formatted = "formatted:" + err.Message()
	return s.formatted
}
func (s *stubFormatter) FormatMultiError(err *ferr.MultiError) string {
	s.multiFormatted = "multi"
	return s.multiFormatted
}
func (s *stubFormatter) FormatValidationErrors([]ferr.FogError) string { return "" }

// stubUI implements ui.OutputHandler for testing.
type stubUI struct {
	errors   []string
	warnings []string
	infos    []string
	debugs   []string
	verbose  bool
}

func (s *stubUI) Success(string)     {}
func (s *stubUI) Info(msg string)    { s.infos = append(s.infos, msg) }
func (s *stubUI) Warning(msg string) { s.warnings = append(s.warnings, msg) }
func (s *stubUI) Error(msg string)   { s.errors = append(s.errors, msg) }
func (s *stubUI) Debug(msg string) {
	if s.verbose {
		s.debugs = append(s.debugs, msg)
	}
}
func (s *stubUI) Table(interface{}, ui.TableOptions) error  { return nil }
func (s *stubUI) JSON(interface{}) error                    { return nil }
func (s *stubUI) StartProgress(string) ui.ProgressIndicator { return nil }
func (s *stubUI) SetStatus(string)                          {}
func (s *stubUI) Confirm(string) bool                       { return false }
func (s *stubUI) ConfirmWithDefault(string, bool) bool      { return false }
func (s *stubUI) SetVerbose(v bool)                         { s.verbose = v }
func (s *stubUI) SetQuiet(bool)                             {}
func (s *stubUI) SetOutputFormat(ui.OutputFormat)           {}
func (s *stubUI) GetWriter() io.Writer                      { return nil }
func (s *stubUI) GetErrorWriter() io.Writer                 { return nil }
func (s *stubUI) GetVerbose() bool                          { return s.verbose }

// Ensure we import io

func TestErrorHandlingMiddleware_ConvertsGenericError(t *testing.T) {
	formatter := &stubFormatter{}
	ui := &stubUI{}
	mw := NewErrorHandlingMiddleware(formatter, ui)

	next := func(context.Context) error {
		return errors.New("boom")
	}

	err := mw.Execute(context.Background(), next)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected original error returned")
	}
	if formatter.lastErr == nil {
		t.Fatalf("formatter not called")
	}
	if len(ui.infos) == 0 || ui.infos[0] != formatter.formatted {
		t.Fatalf("formatted error not output")
	}
}

func TestRecoveryMiddleware_RecoversPanic(t *testing.T) {
	ui := &stubUI{verbose: true}
	mw := NewRecoveryMiddleware(ui)

	err := mw.Execute(context.Background(), func(context.Context) error {
		panic("kaboom")
	})

	if err == nil {
		t.Fatalf("expected error from panic")
	}
	if fe, ok := err.(ferr.FogError); !ok || fe.Code() != ferr.ErrInternal {
		t.Fatalf("unexpected error type: %#v", err)
	}
	if len(ui.errors) == 0 {
		t.Fatalf("error message not displayed")
	}
	if len(ui.debugs) == 0 {
		t.Fatalf("debug output expected in verbose mode")
	}
}

func TestErrorHandlingMiddleware_FormatterIntegration(t *testing.T) {
	formatter := &stubFormatter{}
	ui := &stubUI{}
	mw := NewErrorHandlingMiddleware(formatter, ui)

	// multi error should use FormatMultiError
	e1 := ferr.NewError(ferr.ErrUnknown, "one")
	e2 := ferr.NewError(ferr.ErrUnknown, "two")
	multi := ferr.NewMultiError("ctx", []ferr.FogError{e1, e2})

	next := func(context.Context) error { return multi }
	_ = mw.Execute(context.Background(), next)

	if formatter.multiFormatted == "" {
		t.Fatalf("multi error not formatted")
	}
	if len(ui.errors) == 0 || ui.errors[0] != formatter.multiFormatted {
		t.Fatalf("multi error not output")
	}
}
