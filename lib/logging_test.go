package lib

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/viper"
)

func TestNewDeploymentLog(t *testing.T) {
	// Setup test data
	awsConfig := config.AWSConfig{
		AccountID: "123456789012",
		Region:    "us-west-2",
		UserID:    "test-user",
	}

	deployInfoNew := DeployInfo{
		StackName: "test-stack-new",
		IsNew:     true,
	}

	deployInfoUpdate := DeployInfo{
		StackName: "test-stack-update",
		IsNew:     false,
	}

	// Test cases
	t.Run("Create new stack deployment log", func(t *testing.T) {
		log := NewDeploymentLog(awsConfig, deployInfoNew)

		if log.Account != awsConfig.AccountID {
			t.Errorf("Account = %v, want %v", log.Account, awsConfig.AccountID)
		}
		if log.Region != awsConfig.Region {
			t.Errorf("Region = %v, want %v", log.Region, awsConfig.Region)
		}
		if log.Deployer != awsConfig.UserID {
			t.Errorf("Deployer = %v, want %v", log.Deployer, awsConfig.UserID)
		}
		if log.StackName != deployInfoNew.StackName {
			t.Errorf("StackName = %v, want %v", log.StackName, deployInfoNew.StackName)
		}
		if log.DeploymentType != DeploymentTypeCreateStack {
			t.Errorf("DeploymentType = %v, want %v", log.DeploymentType, DeploymentTypeCreateStack)
		}
		if log.PreChecks != DeploymentLogPreChecksNone {
			t.Errorf("PreChecks = %v, want %v", log.PreChecks, DeploymentLogPreChecksNone)
		}
		// Check that StartedAt is set to a reasonable time (within the last minute)
		now := time.Now().UTC()
		if log.StartedAt.After(now) || log.StartedAt.Before(now.Add(-time.Minute)) {
			t.Errorf("StartedAt = %v, should be close to current time %v", log.StartedAt, now)
		}
	})

	t.Run("Update existing stack deployment log", func(t *testing.T) {
		log := NewDeploymentLog(awsConfig, deployInfoUpdate)

		if log.DeploymentType != DeploymentTypeUpdateStack {
			t.Errorf("DeploymentType = %v, want %v", log.DeploymentType, DeploymentTypeUpdateStack)
		}
	})
}

func TestDeploymentLog_Write(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Setup viper config
	originalLogEnabled := viper.GetBool("logging.enabled")
	originalLogFile := viper.GetString("logging.filename")
	defer func() {
		viper.Set("logging.enabled", originalLogEnabled)
		viper.Set("logging.filename", originalLogFile)
	}()

	viper.Set("logging.enabled", true)
	viper.Set("logging.filename", logFile)

	// Create test deployment log
	deployLog := DeploymentLog{
		Account:        "123456789012",
		Region:         "us-west-2",
		Deployer:       "test-user",
		StackName:      "test-stack",
		DeploymentName: "CFN-123456789012-us-west-2-test-stack",
		DeploymentType: DeploymentTypeCreateStack,
		PreChecks:      DeploymentLogPreChecksNone,
		StartedAt:      time.Now().UTC(),
	}

	// Test writing to log file
	if err := deployLog.Write(); err != nil {
		t.Fatalf("Write() returned unexpected error: %v", err)
	}

	// Verify log file was created and contains valid JSON
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created")
	}

	// Read the log file and verify content
	file, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Failed to close log file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("Failed to read log file")
	}

	var readLog DeploymentLog
	if err := json.Unmarshal(scanner.Bytes(), &readLog); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	// Verify key fields match
	if readLog.Account != deployLog.Account {
		t.Errorf("Account = %v, want %v", readLog.Account, deployLog.Account)
	}
	if readLog.StackName != deployLog.StackName {
		t.Errorf("StackName = %v, want %v", readLog.StackName, deployLog.StackName)
	}
	if readLog.DeploymentType != deployLog.DeploymentType {
		t.Errorf("DeploymentType = %v, want %v", readLog.DeploymentType, deployLog.DeploymentType)
	}
}

func TestDeploymentLog_AddChangeSet(t *testing.T) {
	// Create test data
	deployLog := DeploymentLog{}
	changeset := &ChangesetInfo{
		Changes: []ChangesetChanges{
			{
				Action:    "Add",
				LogicalID: "Resource1",
				Type:      "AWS::S3::Bucket",
			},
			{
				Action:    "Modify",
				LogicalID: "Resource2",
				Type:      "AWS::IAM::Role",
			},
		},
	}

	// Test adding changeset
	deployLog.AddChangeSet(changeset)

	// Verify changes were added
	if len(deployLog.Changes) != len(changeset.Changes) {
		t.Errorf("Changes length = %v, want %v", len(deployLog.Changes), len(changeset.Changes))
	}

	for i, change := range changeset.Changes {
		if deployLog.Changes[i].Action != change.Action {
			t.Errorf("Change[%d].Action = %v, want %v", i, deployLog.Changes[i].Action, change.Action)
		}
		if deployLog.Changes[i].LogicalID != change.LogicalID {
			t.Errorf("Change[%d].LogicalID = %v, want %v", i, deployLog.Changes[i].LogicalID, change.LogicalID)
		}
		if deployLog.Changes[i].Type != change.Type {
			t.Errorf("Change[%d].Type = %v, want %v", i, deployLog.Changes[i].Type, change.Type)
		}
	}
}

func TestDeploymentLog_Success(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "success.log")

	viper.Set("logging.enabled", true)
	viper.Set("logging.filename", logFile)

	deployLog := DeploymentLog{
		Account:   "123456789012",
		StackName: "test-stack",
		StartedAt: time.Now().UTC(),
	}

	// Test marking as success
	if err := deployLog.Success(); err != nil {
		t.Fatalf("Success() returned unexpected error: %v", err)
	}

	// Verify log file was created
	file, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Failed to close log file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("Failed to read log file")
	}

	var readLog DeploymentLog
	if err := json.Unmarshal(scanner.Bytes(), &readLog); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	// Verify status was set to success
	if readLog.Status != DeploymentLogStatusSuccess {
		t.Errorf("Status = %v, want %v", readLog.Status, DeploymentLogStatusSuccess)
	}
}

func TestDeploymentLog_Failed(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "failed.log")

	viper.Set("logging.enabled", true)
	viper.Set("logging.filename", logFile)

	deployLog := DeploymentLog{
		Account:   "123456789012",
		StackName: "test-stack",
		StartedAt: time.Now().UTC(),
	}

	// Test failures
	failures := []map[string]any{
		{
			"resource": "Resource1",
			"reason":   "Permission denied",
		},
	}

	if err := deployLog.Failed(failures); err != nil {
		t.Fatalf("Failed() returned unexpected error: %v", err)
	}

	// Verify log file was created
	file, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("Failed to open log file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Failed to close log file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("Failed to read log file")
	}

	var readLog DeploymentLog
	if err := json.Unmarshal(scanner.Bytes(), &readLog); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	// Verify status was set to failed
	if readLog.Status != DeploymentLogStatusFailed {
		t.Errorf("Status = %v, want %v", readLog.Status, DeploymentLogStatusFailed)
	}

	// Verify failures were recorded
	if len(readLog.Failures) != len(failures) {
		t.Errorf("Failures length = %v, want %v", len(readLog.Failures), len(failures))
	}
}

func TestReadAllLogs(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "all_logs.log")

	viper.Set("logging.filename", logFile)

	// Create test log entries
	logs := []DeploymentLog{
		{
			Account:   "123456789012",
			StackName: "stack1",
			StartedAt: time.Now().UTC().Add(-2 * time.Hour),
			Status:    DeploymentLogStatusSuccess,
			UpdatedAt: time.Now().UTC().Add(-1 * time.Hour),
		},
		{
			Account:   "123456789012",
			StackName: "stack2",
			StartedAt: time.Now().UTC().Add(-1 * time.Hour),
			Status:    DeploymentLogStatusFailed,
			UpdatedAt: time.Now().UTC().Add(-30 * time.Minute),
		},
		{
			Account:   "123456789012",
			StackName: "stack3",
			StartedAt: time.Now().UTC(),
			Status:    DeploymentLogStatusSuccess,
			UpdatedAt: time.Now().UTC(),
		},
	}

	// Write logs to file
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	for _, log := range logs {
		data, err := json.Marshal(log)
		if err != nil {
			t.Fatalf("Failed to marshal log: %v", err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatalf("Failed to write to log file: %v", err)
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			t.Fatalf("Failed to write newline to log file: %v", err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close log file: %v", err)
	}

	// Test reading all logs
	readLogs := ReadAllLogs()

	// Verify logs were read and sorted correctly (newest first)
	if len(readLogs) != len(logs) {
		t.Errorf("Read %d logs, want %d", len(readLogs), len(logs))
	}

	// Check that logs are sorted by StartedAt in reverse order
	for i := 0; i < len(readLogs)-1; i++ {
		if readLogs[i].StartedAt.Before(readLogs[i+1].StartedAt) {
			t.Errorf("Logs not sorted correctly by StartedAt in reverse order")
		}
	}
}

func TestReadAllLogsSkipsMalformedLogEntries(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "malformed_logs.log")

	originalLogFile := viper.GetString("logging.filename")
	t.Cleanup(func() {
		viper.Set("logging.filename", originalLogFile)
	})

	viper.Set("logging.filename", logFile)

	validOlder := DeploymentLog{
		Account:   "123456789012",
		StackName: "stack-old",
		StartedAt: time.Now().UTC().Add(-2 * time.Hour),
		Status:    DeploymentLogStatusSuccess,
		UpdatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	validNewer := DeploymentLog{
		Account:   "123456789012",
		StackName: "stack-new",
		StartedAt: time.Now().UTC().Add(-1 * time.Hour),
		Status:    DeploymentLogStatusFailed,
		UpdatedAt: time.Now().UTC().Add(-1 * time.Hour),
	}

	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	data, err := json.Marshal(validOlder)
	if err != nil {
		t.Fatalf("Failed to marshal older log: %v", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		t.Fatalf("Failed to write older log: %v", err)
	}
	if _, err := file.Write([]byte("{malformed-json}\n")); err != nil {
		t.Fatalf("Failed to write malformed log line: %v", err)
	}
	data, err = json.Marshal(validNewer)
	if err != nil {
		t.Fatalf("Failed to marshal newer log: %v", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		t.Fatalf("Failed to write newer log: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close log file: %v", err)
	}

	var warningBuffer bytes.Buffer
	warningLogger := func(format string, args ...any) {
		warningBuffer.WriteString(fmt.Sprintf(format, args...))
		warningBuffer.WriteByte('\n')
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("ReadAllLogs panicked on malformed log line: %v", recovered)
		}
	}()

	readLogs := readAllLogs(warningLogger)
	if len(readLogs) != 2 {
		t.Fatalf("Read %d logs, want 2 valid logs", len(readLogs))
	}

	// Expected behavior: malformed lines are skipped and valid logs are still sorted newest first.
	if readLogs[0].StackName != validNewer.StackName {
		t.Errorf("First log StackName = %q, want %q", readLogs[0].StackName, validNewer.StackName)
	}
	if readLogs[1].StackName != validOlder.StackName {
		t.Errorf("Second log StackName = %q, want %q", readLogs[1].StackName, validOlder.StackName)
	}

	warningOutput := warningBuffer.String()
	if !strings.Contains(warningOutput, "Warning: skipping malformed deployment log entry on line 2") {
		t.Errorf("Expected warning for malformed log line, got: %q", warningOutput)
	}
	if !strings.Contains(warningOutput, logFile) {
		t.Errorf("Expected warning to include log file path %q, got: %q", logFile, warningOutput)
	}
}

func TestGetLatestSuccessFulLogByDeploymentName(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "success_logs.log")

	viper.Set("logging.filename", logFile)

	deploymentName := "CFN-123456789012-us-west-2-test-stack"

	// Create test log entries
	logs := []DeploymentLog{
		{
			DeploymentName: "CFN-123456789012-us-west-2-other-stack",
			StartedAt:      time.Now().UTC().Add(-3 * time.Hour),
			Status:         DeploymentLogStatusSuccess,
		},
		{
			DeploymentName: deploymentName,
			StartedAt:      time.Now().UTC().Add(-2 * time.Hour),
			Status:         DeploymentLogStatusFailed,
		},
		{
			DeploymentName: deploymentName,
			StartedAt:      time.Now().UTC().Add(-1 * time.Hour),
			Status:         DeploymentLogStatusSuccess,
			StackName:      "test-stack",
		},
	}

	// Write logs to file
	file, err := os.Create(logFile)
	if err != nil {
		t.Fatalf("Failed to create log file: %v", err)
	}

	for _, log := range logs {
		data, err := json.Marshal(log)
		if err != nil {
			t.Fatalf("Failed to marshal log: %v", err)
		}
		if _, err := file.Write(data); err != nil {
			t.Fatalf("Failed to write to log file: %v", err)
		}
		if _, err := file.Write([]byte("\n")); err != nil {
			t.Fatalf("Failed to write newline to log file: %v", err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Failed to close log file: %v", err)
	}

	// Test getting latest successful log
	log := GetLatestSuccessFulLogByDeploymentName(deploymentName)

	// Verify correct log was returned
	if log.DeploymentName != deploymentName {
		t.Errorf("DeploymentName = %v, want %v", log.DeploymentName, deploymentName)
	}
	if log.Status != DeploymentLogStatusSuccess {
		t.Errorf("Status = %v, want %v", log.Status, DeploymentLogStatusSuccess)
	}
	if log.StackName != "test-stack" {
		t.Errorf("StackName = %v, want %v", log.StackName, "test-stack")
	}
}

func TestDeploymentLog_WriteReturnsErrorOnFileWriteFailure(t *testing.T) {
	// Regression test for T-466: Write() used to call log.Fatal on file write errors,
	// which terminated the process. It should return an error instead.
	originalLogEnabled := viper.GetBool("logging.enabled")
	originalLogFile := viper.GetString("logging.filename")
	defer func() {
		viper.Set("logging.enabled", originalLogEnabled)
		viper.Set("logging.filename", originalLogFile)
	}()

	viper.Set("logging.enabled", true)
	// Point to a path that cannot be opened (directory that doesn't exist)
	viper.Set("logging.filename", "/nonexistent-dir/impossible-path/test.log")

	deployLog := DeploymentLog{
		Account:   "123456789012",
		StackName: "test-stack",
		StartedAt: time.Now().UTC(),
	}

	// Write should return an error, not panic or call os.Exit
	err := deployLog.Write()
	if err == nil {
		t.Error("Write() returned nil error for an unwritable log path; expected an error")
	}
}

func TestDeploymentLog_WriteReturnsErrorOnLoggingDisabled(t *testing.T) {
	// When logging is disabled, Write should succeed (no-op) and return nil.
	originalLogEnabled := viper.GetBool("logging.enabled")
	defer func() {
		viper.Set("logging.enabled", originalLogEnabled)
	}()

	viper.Set("logging.enabled", false)

	deployLog := DeploymentLog{
		Account:   "123456789012",
		StackName: "test-stack",
		StartedAt: time.Now().UTC(),
	}

	err := deployLog.Write()
	if err != nil {
		t.Errorf("Write() returned error when logging is disabled: %v", err)
	}
}

func TestDeploymentLog_SuccessReturnsErrorOnWriteFailure(t *testing.T) {
	// Regression test for T-466: Success() wraps Write(), which used to
	// panic/exit on errors. Success() should propagate the error instead.
	originalLogEnabled := viper.GetBool("logging.enabled")
	originalLogFile := viper.GetString("logging.filename")
	defer func() {
		viper.Set("logging.enabled", originalLogEnabled)
		viper.Set("logging.filename", originalLogFile)
	}()

	viper.Set("logging.enabled", true)
	viper.Set("logging.filename", "/nonexistent-dir/impossible-path/test.log")

	deployLog := DeploymentLog{
		Account:   "123456789012",
		StackName: "test-stack",
		StartedAt: time.Now().UTC(),
	}

	err := deployLog.Success()
	if err == nil {
		t.Error("Success() returned nil error for an unwritable log path; expected an error")
	}
	// Status should still be set even if the write fails
	if deployLog.Status != DeploymentLogStatusSuccess {
		t.Errorf("Status = %v, want %v", deployLog.Status, DeploymentLogStatusSuccess)
	}
}

func TestDeploymentLog_FailedReturnsErrorOnWriteFailure(t *testing.T) {
	// Regression test for T-466: Failed() wraps Write(), which used to
	// panic/exit on errors. Failed() should propagate the error instead.
	originalLogEnabled := viper.GetBool("logging.enabled")
	originalLogFile := viper.GetString("logging.filename")
	defer func() {
		viper.Set("logging.enabled", originalLogEnabled)
		viper.Set("logging.filename", originalLogFile)
	}()

	viper.Set("logging.enabled", true)
	viper.Set("logging.filename", "/nonexistent-dir/impossible-path/test.log")

	deployLog := DeploymentLog{
		Account:   "123456789012",
		StackName: "test-stack",
		StartedAt: time.Now().UTC(),
	}

	failures := []map[string]any{
		{"resource": "Resource1", "reason": "Permission denied"},
	}

	err := deployLog.Failed(failures)
	if err == nil {
		t.Error("Failed() returned nil error for an unwritable log path; expected an error")
	}
	// Status and failures should still be set even if the write fails
	if deployLog.Status != DeploymentLogStatusFailed {
		t.Errorf("Status = %v, want %v", deployLog.Status, DeploymentLogStatusFailed)
	}
	if len(deployLog.Failures) != 1 {
		t.Errorf("Failures length = %d, want 1", len(deployLog.Failures))
	}
}

func TestGenerateDeploymentName(t *testing.T) {
	// Test data
	awsConfig := config.AWSConfig{
		AccountID: "123456789012",
		Region:    "us-west-2",
	}
	stackName := "test-stack"

	// Expected format: CFN-{AccountID}-{Region}-{StackName}
	expected := "CFN-123456789012-us-west-2-test-stack"

	// Test generating deployment name
	result := GenerateDeploymentName(awsConfig, stackName)

	if result != expected {
		t.Errorf("GenerateDeploymentName() = %v, want %v", result, expected)
	}
}

func TestReverseLogs(t *testing.T) {
	// Create test logs with different timestamps
	now := time.Now().UTC()
	logs := ReverseLogs{
		DeploymentLog{StartedAt: now.Add(-2 * time.Hour)}, // oldest
		DeploymentLog{StartedAt: now},                     // newest
		DeploymentLog{StartedAt: now.Add(-1 * time.Hour)}, // middle
	}

	// Test sorting
	expected := ReverseLogs{
		DeploymentLog{StartedAt: now}, // newest first
		DeploymentLog{StartedAt: now.Add(-1 * time.Hour)},
		DeploymentLog{StartedAt: now.Add(-2 * time.Hour)}, // oldest last
	}

	// Sort the logs
	sort.Sort(logs)

	// Verify sorting order
	for i := range logs {
		if !logs[i].StartedAt.Equal(expected[i].StartedAt) {
			t.Errorf("Logs[%d].StartedAt = %v, want %v", i, logs[i].StartedAt, expected[i].StartedAt)
		}
	}
}
