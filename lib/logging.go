package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/viper"
)

// DeploymentType represents the type of CloudFormation deployment operation.
type DeploymentType string

const (
	// DeploymentTypeCreateStack indicates a new stack creation.
	DeploymentTypeCreateStack DeploymentType = "CREATE"
	// DeploymentTypeUpdateStack indicates an existing stack update.
	DeploymentTypeUpdateStack DeploymentType = "UPDATE"
)

// DeploymentLogStatus represents the final status of a deployment operation.
type DeploymentLogStatus string

const (
	// DeploymentLogStatusSuccess indicates the deployment completed successfully.
	DeploymentLogStatusSuccess DeploymentLogStatus = "SUCCESS"
	// DeploymentLogStatusFailed indicates the deployment failed.
	DeploymentLogStatusFailed DeploymentLogStatus = "FAILED"
)

// DeploymentLogPreChecks represents the result of pre-deployment validation checks.
type DeploymentLogPreChecks string

const (
	// DeploymentLogPreChecksNone indicates no pre-checks were run.
	DeploymentLogPreChecksNone DeploymentLogPreChecks = "NONE"
	// DeploymentLogPreChecksPassed indicates pre-checks completed successfully.
	DeploymentLogPreChecksPassed DeploymentLogPreChecks = "PASSED"
	// DeploymentLogPreChecksFailed indicates pre-checks failed.
	DeploymentLogPreChecksFailed DeploymentLogPreChecks = "FAILED"
)

// DeploymentLog represents a log entry for a CloudFormation deployment
type DeploymentLog struct {
	// The AWS Account
	Account string
	// The list of changes that comprise the change set
	Changes []ChangesetChanges
	// Deployer is the name of the user/role who deploys the stack
	Deployer string
	// DeploymentName is a unique name that combines the account, region, and stackname to ensure uniqueness
	DeploymentName string
	// The type of deployment
	DeploymentType DeploymentType
	// The rows that failed
	Failures []map[string]any
	// Did the prechecks pass?
	PreChecks DeploymentLogPreChecks
	// The AWS Region
	Region string
	// The name of the stack to be deployed
	StackName string
	// The status of the deployment
	Status DeploymentLogStatus
	// A longer description of the status
	StatusDescription string
	// The time (in UTC) the deployment started
	StartedAt time.Time
	// The time (in UTC) the status of the deployment was last updated
	UpdatedAt time.Time
}

// NewDeploymentLog creates a new deployment log entry from AWS config and deployment info
func NewDeploymentLog(awsConfig config.AWSConfig, deployment DeployInfo) DeploymentLog {
	deploylog := DeploymentLog{
		Account:        awsConfig.AccountID,
		Region:         awsConfig.Region,
		Deployer:       awsConfig.UserID,
		StackName:      deployment.StackName,
		DeploymentName: GenerateDeploymentName(awsConfig, deployment.StackName),
		PreChecks:      DeploymentLogPreChecksNone,
		StartedAt:      time.Now().UTC(),
	}
	if deployment.IsNew {
		deploylog.DeploymentType = DeploymentTypeCreateStack
	} else {
		deploylog.DeploymentType = DeploymentTypeUpdateStack
	}
	return deploylog
}

// Write writes the deployment log to the configured log file if logging is enabled.
// Returns an error if the log entry cannot be marshalled or written to the file.
func (deploymentlog *DeploymentLog) Write() error {
	if !viper.GetBool("logging.enabled") {
		return nil
	}
	deploymentlog.UpdatedAt = time.Now().UTC()
	jsonString, err := json.Marshal(deploymentlog)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment log: %w", err)
	}
	outputFile := viper.GetString("logging.filename")
	if err := writeLogToFile(jsonString, outputFile); err != nil {
		return fmt.Errorf("failed to write deployment log to %s: %w", outputFile, err)
	}
	return nil
}

// writeLogToFile opens the given file path and delegates to writeToFile.
func writeLogToFile(contents []byte, outputFile string) error {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	return writeToFile(contents, file)
}

// writeToFile writes contents followed by a newline to the given writer and
// closes it. A named return is used so the deferred Close error is propagated
// when no prior write or flush error occurred.
func writeToFile(contents []byte, file io.WriteCloser) (err error) {
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	w := bufio.NewWriter(file)
	contents = append(contents, '\n')
	if _, err = w.Write(contents); err != nil {
		return err
	}
	if err = w.Flush(); err != nil {
		return err
	}
	return nil
}

// AddChangeSet adds the changeset information to the deployment log
func (deploymentlog *DeploymentLog) AddChangeSet(changeset *ChangesetInfo) {
	deploymentlog.Changes = changeset.Changes
}

// Success marks the deployment as successful and writes the log.
// Returns an error if the log entry cannot be written.
func (deploymentlog *DeploymentLog) Success() error {
	deploymentlog.Status = DeploymentLogStatusSuccess
	return deploymentlog.Write()
}

// Failed marks the deployment as failed with the provided failure details and writes the log.
// Returns an error if the log entry cannot be written.
func (deploymentlog *DeploymentLog) Failed(failures []map[string]any) error {
	deploymentlog.Status = DeploymentLogStatusFailed
	deploymentlog.Failures = failures
	return deploymentlog.Write()
}

// ReadAllLogs reads all deployment logs from the configured log file
func ReadAllLogs() []DeploymentLog {
	return readAllLogs(log.Printf)
}

func readAllLogs(logf func(string, ...any)) []DeploymentLog {
	result := make([]DeploymentLog, 0)
	filename := viper.GetString("logging.filename")
	file, err := os.Open(filename)
	if err != nil {
		// If the file doesn't exist, just return an empty result
		return result
	}
	defer func() {
		if err := file.Close(); err != nil {
			logf("Error closing file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	// Allow larger log lines than the default 64KB scanner token limit.
	// Cap at 10MB to handle large deployment records without unbounded memory growth.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		deploymentlog := DeploymentLog{}
		err := json.Unmarshal(scanner.Bytes(), &deploymentlog)
		if err != nil {
			logf("Warning: skipping malformed deployment log entry on line %d in %s: %v", lineNumber, filename, err)
			continue
		}
		result = append(result, deploymentlog)
	}

	if err := scanner.Err(); err != nil {
		logf("Error scanning file: %v", err)
		return result
	}

	sort.Sort(ReverseLogs(result))
	return result
}

// GetLatestSuccessFulLogByDeploymentName retrieves the most recent successful deployment log for the given deployment name
func GetLatestSuccessFulLogByDeploymentName(deploymentName string) DeploymentLog {
	logs := ReadAllLogs()
	for _, log := range logs {
		if log.DeploymentName == deploymentName && log.Status == DeploymentLogStatusSuccess {
			return log
		}
	}
	return DeploymentLog{}
}

// GenerateDeploymentName generates a unique deployment name from account, region, and stack name
func GenerateDeploymentName(awsConfig config.AWSConfig, stackName string) string {
	return fmt.Sprintf("CFN-%v-%v-%v", awsConfig.AccountID, awsConfig.Region, stackName)
}

// ReverseLogs implements sort.Interface for sorting deployment logs in reverse chronological order
type ReverseLogs []DeploymentLog

// Len returns the length of the slice
func (a ReverseLogs) Len() int { return len(a) }

// Less compares two logs by start time in reverse order
func (a ReverseLogs) Less(i, j int) bool { return a[i].StartedAt.After(a[j].StartedAt) }

// Swap swaps two elements in the slice
func (a ReverseLogs) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
