package main

import (
	"testing"
)

// TestHandleRequestReturnsError verifies that HandleRequest returns an error
// when required environment variables are missing.
// Bug: T-483 — HandleRequest did not return errors, so Lambda always reported
// success even when report generation could not proceed.
func TestHandleRequestReturnsErrorOnMissingEnvVars(t *testing.T) {
	t.Parallel()

	// Ensure required env vars are not set
	t.Setenv("ReportS3Bucket", "")
	t.Setenv("ReportOutputFormat", "")

	msg := EventBridgeMessage{}
	msg.Detail.StackId = "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc"

	err := HandleRequest(msg)
	if err == nil {
		t.Fatal("HandleRequest should return an error when ReportS3Bucket is empty")
	}
}

// TestHandleRequestReturnsErrorOnMissingBucket verifies that a missing
// ReportS3Bucket env var causes an error even when other vars are set.
func TestHandleRequestReturnsErrorOnMissingBucket(t *testing.T) {
	t.Parallel()

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
}

// TestHandleRequestReturnsErrorOnMissingFormat verifies that a missing
// ReportOutputFormat env var causes an error.
func TestHandleRequestReturnsErrorOnMissingFormat(t *testing.T) {
	t.Parallel()

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
}
