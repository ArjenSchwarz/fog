package lib

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/viper"
)

type DeploymentType string

const (
	DeploymentTypeCreateStack DeploymentType = "CREATE"
	DeploymentTypeUpdateStack DeploymentType = "UPDATE"
)

type DeploymentLogStatus string

const (
	DeploymentLogStatusSuccess DeploymentLogStatus = "SUCCESS"
	DeploymentLogStatusFailed  DeploymentLogStatus = "FAILED"
)

type DeploymentLogPreChecks string

const (
	DeploymentLogPreChecksNone   DeploymentLogPreChecks = "NONE"
	DeploymentLogPreChecksPassed DeploymentLogPreChecks = "PASSED"
	DeploymentLogPreChecksFailed DeploymentLogPreChecks = "FAILED"
)

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
	Failures []map[string]interface{}
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

func (deploymentlog *DeploymentLog) Write() {
	if viper.GetBool("logging.enabled") {
		deploymentlog.UpdatedAt = time.Now().UTC()
		jsonString, err := json.Marshal(deploymentlog)
		if err != nil {
			panic("error with encoding logs")
		}
		outputFile := viper.GetString("logging.filename")
		err = writeLogToFile(jsonString, outputFile)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}

// writeLogToFile prints the provided contents to stdout or the provided filepath
func writeLogToFile(contents []byte, outputFile string) error {
	file, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	w := bufio.NewWriter(file)
	contents = append(contents, '\n')
	if _, werr := w.Write(contents); werr != nil {
		return werr
	}
	if ferr := w.Flush(); ferr != nil {
		return ferr
	}
	return nil
}

func (deploymentlog *DeploymentLog) AddChangeSet(changeset *ChangesetInfo) {
	deploymentlog.Changes = changeset.Changes
}

func (deploymentlog *DeploymentLog) Success() {
	deploymentlog.Status = DeploymentLogStatusSuccess
	deploymentlog.Write()
}

func (deploymentlog *DeploymentLog) Failed(failures []map[string]interface{}) {
	deploymentlog.Status = DeploymentLogStatusFailed
	deploymentlog.Failures = failures
	deploymentlog.Write()
}

func ReadAllLogs() []DeploymentLog {
	result := make([]DeploymentLog, 0)
	filename := viper.GetString("logging.filename")
	file, err := os.Open(filename)
	if err != nil {
		// If the file doesn't exist, just return an empty result
		return result
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file: %v", err)
		}
	}()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		deploymentlog := DeploymentLog{}
		err := json.Unmarshal(scanner.Bytes(), &deploymentlog)
		if err != nil {
			panic(err)
		}
		result = append(result, deploymentlog)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	sort.Sort(ReverseLogs(result))
	return result
}

func GetLatestSuccessFulLogByDeploymentName(deploymentName string) DeploymentLog {
	logs := ReadAllLogs()
	for _, log := range logs {
		if log.DeploymentName == deploymentName && log.Status == DeploymentLogStatusSuccess {
			return log
		}
	}
	return DeploymentLog{}
}

func GenerateDeploymentName(awsConfig config.AWSConfig, stackName string) string {
	return fmt.Sprintf("CFN-%v-%v-%v", awsConfig.AccountID, awsConfig.Region, stackName)
}

type ReverseLogs []DeploymentLog

func (a ReverseLogs) Len() int           { return len(a) }
func (a ReverseLogs) Less(i, j int) bool { return a[i].StartedAt.After(a[j].StartedAt) }
func (a ReverseLogs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
