package main

import (
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
