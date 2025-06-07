package errors

import (
	"strings"
	"testing"
	"time"
)

type stubError struct {
	code        ErrorCode
	message     string
	details     string
	operation   string
	component   string
	timestamp   time.Time
	category    ErrorCategory
	severity    ErrorSeverity
	retryable   bool
	stack       []string
	cause       error
	userMessage string
	suggestions []string
	fields      map[string]interface{}
}

func (e *stubError) Error() string                  { return e.message }
func (e *stubError) Code() ErrorCode                { return e.code }
func (e *stubError) Message() string                { return e.message }
func (e *stubError) Details() string                { return e.details }
func (e *stubError) Operation() string              { return e.operation }
func (e *stubError) Component() string              { return e.component }
func (e *stubError) Timestamp() time.Time           { return e.timestamp }
func (e *stubError) Category() ErrorCategory        { return e.category }
func (e *stubError) Severity() ErrorSeverity        { return e.severity }
func (e *stubError) Retryable() bool                { return e.retryable }
func (e *stubError) StackTrace() []string           { return e.stack }
func (e *stubError) Cause() error                   { return e.cause }
func (e *stubError) UserMessage() string            { return e.userMessage }
func (e *stubError) Suggestions() []string          { return e.suggestions }
func (e *stubError) Fields() map[string]interface{} { return e.fields }
func (e *stubError) WithField(key string, value interface{}) FogError {
	if e.fields == nil {
		e.fields = make(map[string]interface{})
	}
	e.fields[key] = value
	return e
}
func (e *stubError) WithFields(fields map[string]interface{}) FogError {
	if e.fields == nil {
		e.fields = make(map[string]interface{})
	}
	for k, v := range fields {
		e.fields[k] = v
	}
	return e
}

func newStubError() *stubError {
	return &stubError{
		code:        "ERR",
		message:     "something went wrong",
		timestamp:   time.Unix(0, 0),
		severity:    SeverityHigh,
		operation:   "op",
		component:   "comp",
		stack:       []string{"frame1", "frame2"},
		userMessage: "invalid input",
	}
}

func TestConsoleErrorFormatter_VerboseIncludesStack(t *testing.T) {
	err := newStubError()
	formatter := NewConsoleErrorFormatter(false, true)
	out := formatter.FormatError(err)

	if !strings.Contains(out, "Stack trace:") {
		t.Errorf("expected stack trace in verbose output")
	}
	if !strings.Contains(out, "frame1") || !strings.Contains(out, "frame2") {
		t.Errorf("stack frames missing from output")
	}
	if !strings.Contains(out, "Context:") {
		t.Errorf("expected context information")
	}
}

func TestConsoleErrorFormatter_NonVerboseOmitsStack(t *testing.T) {
	err := newStubError()
	formatter := NewConsoleErrorFormatter(false, false)
	out := formatter.FormatError(err)

	if strings.Contains(out, "Stack trace:") {
		t.Errorf("stack trace should be omitted when not verbose")
	}
	if strings.Contains(out, "Context:") {
		t.Errorf("context should be omitted when not verbose")
	}
}

func TestJSONErrorFormatter_FormatError(t *testing.T) {
	err := newStubError()
	formatter := NewJSONErrorFormatter()
	out := formatter.FormatError(err)

	if !strings.Contains(out, string(err.Code())) {
		t.Errorf("code not found in JSON output")
	}
	if !strings.Contains(out, err.Message()) {
		t.Errorf("message not found in JSON output")
	}
	if !strings.Contains(out, formatter.severityName(err.Severity())) {
		t.Errorf("severity not found in JSON output")
	}
	if !strings.Contains(out, formatter.categoryName(err.Category())) {
		t.Errorf("category not found in JSON output")
	}
}
