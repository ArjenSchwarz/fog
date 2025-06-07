package errors

import "testing"

func TestGetErrorCategory(t *testing.T) {
	cases := []struct {
		code     ErrorCode
		expected ErrorCategory
	}{
		{ErrValidationFailed, CategoryValidation},
		{ErrConfigInvalid, CategoryConfiguration},
		{ErrFileNotFound, CategoryFileSystem},
		{ErrTemplateInvalid, CategoryTemplate},
		{ErrAWSServiceError, CategoryAWS},
		{ErrNetworkTimeout, CategoryNetwork},
		{ErrResourceLocked, CategoryResource},
		{ErrInternal, CategoryInternal},
		{ErrorCode("UNKNOWN_CODE"), CategoryUnknown},
	}

	for _, c := range cases {
		if got := GetErrorCategory(c.code); got != c.expected {
			t.Errorf("GetErrorCategory(%s)=%v, want %v", c.code, got, c.expected)
		}
	}
}

func TestGetErrorSeverity(t *testing.T) {
	cases := []struct {
		code     ErrorCode
		expected ErrorSeverity
	}{
		{ErrDeploymentFailed, SeverityCritical},
		{ErrAWSPermission, SeverityHigh},
		{ErrValidationFailed, SeverityMedium},
		{ErrorCode("OTHER"), SeverityLow},
	}

	for _, c := range cases {
		if got := GetErrorSeverity(c.code); got != c.expected {
			t.Errorf("GetErrorSeverity(%s)=%v, want %v", c.code, got, c.expected)
		}
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		code     ErrorCode
		expected bool
	}{
		{ErrNetworkTimeout, true},
		{ErrAWSServiceError, true},
		{ErrValidationFailed, false},
		{ErrorCode("OTHER"), false},
	}
	for _, c := range cases {
		if got := IsRetryable(c.code); got != c.expected {
			t.Errorf("IsRetryable(%s)=%v, want %v", c.code, got, c.expected)
		}
	}
}

func TestGetErrorMetadata(t *testing.T) {
	md := GetErrorMetadata(ErrTemplateNotFound)
	if md.Code != ErrTemplateNotFound {
		t.Errorf("wrong code: %v", md.Code)
	}
	if md.Category != CategoryTemplate {
		t.Errorf("wrong category: %v", md.Category)
	}
	if md.Severity != SeverityLow {
		t.Errorf("wrong severity: %v", md.Severity)
	}
	if md.Retryable {
		t.Errorf("expected not retryable")
	}
	if md.Description == "" || len(md.Suggestions) == 0 {
		t.Errorf("expected description and suggestions for metadata")
	}
}
