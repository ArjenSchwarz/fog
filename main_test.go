package main

import (
	"errors"
	"strings"
	"testing"
)

// TestHandleRequestReturnsErrorOnMissingBucket verifies that a missing
// ReportS3Bucket env var causes an error even when other vars are set.
// Bug: T-483 — HandleRequest did not return errors, so Lambda always reported
// success even when report generation could not proceed.
func TestHandleRequestReturnsErrorOnMissingBucket(t *testing.T) {
	t.Setenv("ReportS3Bucket", "")
	t.Setenv("ReportOutputFormat", "markdown")
	t.Setenv("ReportNamePattern", "report-$STACKNAME.md")
	t.Setenv("ReportTimezone", "UTC")

	msg := EventBridgeMessage{}
	msg.Detail.StackId = "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc"

	err := HandleRequest(msg)
	if err == nil {
		t.Fatal("HandleRequest should return an error when ReportS3Bucket is empty")
	}
	if !strings.Contains(err.Error(), "ReportS3Bucket") {
		t.Errorf("error should mention ReportS3Bucket, got: %v", err)
	}
}

// TestHandleRequestReturnsErrorOnMissingFormat verifies that a missing
// ReportOutputFormat env var causes an error.
func TestHandleRequestReturnsErrorOnMissingFormat(t *testing.T) {
	t.Setenv("ReportS3Bucket", "my-bucket")
	t.Setenv("ReportOutputFormat", "")
	t.Setenv("ReportNamePattern", "report-$STACKNAME.md")
	t.Setenv("ReportTimezone", "UTC")

	msg := EventBridgeMessage{}
	msg.Detail.StackId = "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc"

	err := HandleRequest(msg)
	if err == nil {
		t.Fatal("HandleRequest should return an error when ReportOutputFormat is empty")
	}
	if !strings.Contains(err.Error(), "ReportOutputFormat") {
		t.Errorf("error should mention ReportOutputFormat, got: %v", err)
	}
}

// TestHandleRequestReturnsErrorOnMissingStackID verifies that invalid EventBridge
// payloads are rejected before report generation starts.
func TestHandleRequestReturnsErrorOnMissingStackID(t *testing.T) {
	t.Setenv("ReportS3Bucket", "my-bucket")
	t.Setenv("ReportOutputFormat", "markdown")
	t.Setenv("ReportNamePattern", "report-$STACKNAME.md")
	t.Setenv("ReportTimezone", "UTC")

	originalGenerateReportFromLambda := generateReportFromLambda
	t.Cleanup(func() {
		generateReportFromLambda = originalGenerateReportFromLambda
	})

	var called bool
	generateReportFromLambda = func(stackname string, bucketname string, outputfilename string, outputformat string, timezone string) error {
		called = true
		return errors.New("report generation should not be called")
	}

	err := HandleRequest(EventBridgeMessage{})
	if err == nil {
		t.Fatal("HandleRequest should return an error when detail.stack-id is empty")
	}
	if !strings.Contains(err.Error(), "stack-id") {
		t.Errorf("error should mention stack-id, got: %v", err)
	}
	if called {
		t.Fatal("HandleRequest should fail before invoking report generation")
	}
}

func TestHandleRequestPassesTrimmedStackIDToReportGeneration(t *testing.T) {
	t.Setenv("ReportS3Bucket", "my-bucket")
	t.Setenv("ReportOutputFormat", "markdown")
	t.Setenv("ReportNamePattern", "report-$STACKNAME.md")
	t.Setenv("ReportTimezone", "UTC")

	originalGenerateReportFromLambda := generateReportFromLambda
	t.Cleanup(func() {
		generateReportFromLambda = originalGenerateReportFromLambda
	})

	var receivedStackID string
	generateReportFromLambda = func(stackname string, bucketname string, outputfilename string, outputformat string, timezone string) error {
		receivedStackID = stackname
		return nil
	}

	msg := EventBridgeMessage{}
	msg.Detail.StackId = "  arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc  "

	if err := HandleRequest(msg); err != nil {
		t.Fatalf("HandleRequest returned unexpected error: %v", err)
	}
	if receivedStackID != "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc" {
		t.Fatalf("expected trimmed stack-id to be forwarded, got %q", receivedStackID)
	}
}
