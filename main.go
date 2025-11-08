/*
Copyright © 2021 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package main provides the entry point for the fog CLI tool and AWS Lambda handler.
//
// Fog is a command-line tool for managing AWS CloudFormation stacks. It provides
// functionality for deploying, reporting, drift detection, and managing CloudFormation
// resources. The application can run both as a CLI tool and as an AWS Lambda function
// for automated reporting.
//
// When run as a CLI tool, fog provides various commands for stack management.
// When deployed as a Lambda function (detected via AWS_LAMBDA_FUNCTION_NAME environment
// variable), it processes EventBridge messages for CloudFormation events and generates
// reports that are stored in S3.
package main

import (
	"os"
	"time"

	"github.com/ArjenSchwarz/fog/cmd"
	"github.com/aws/aws-lambda-go/lambda"
)

// EventBridgeMessage represents an AWS EventBridge message for CloudFormation stack events.
//
// This structure is used to parse EventBridge notifications when fog runs as a Lambda function.
// It captures CloudFormation stack state changes and provides the necessary information to
// generate deployment reports. The message follows the standard EventBridge event format
// with CloudFormation-specific details in the Detail field.
//
// Example EventBridge message:
//
//	{
//	  "version": "0",
//	  "source": "aws.cloudformation",
//	  "account": "123456789012",
//	  "id": "abc-def-ghi",
//	  "region": "us-east-1",
//	  "detail-type": "CloudFormation Stack Status Change",
//	  "time": "2024-01-01T12:00:00Z",
//	  "resources": ["arn:aws:cloudformation:..."],
//	  "detail": {
//	    "stack-id": "arn:aws:cloudformation:us-east-1:123456789012:stack/my-stack/...",
//	    "status-details": {
//	      "status": "CREATE_COMPLETE",
//	      "status-reason": ""
//	    }
//	  }
//	}
type EventBridgeMessage struct {
	Version    string    `json:"version"`
	Source     string    `json:"source"`
	Account    string    `json:"account"`
	Id         string    `json:"id"`
	Region     string    `json:"region"`
	DetailType string    `json:"detail-type"`
	Time       time.Time `json:"time"`
	Resources  []string  `json:"resources"`
	Detail     struct {
		StackId       string `json:"stack-id"`
		StatusDetails struct {
			Status       string `json:"status"`
			StatusReason string `json:"status-reason"`
		} `json:"status-details"`
	} `json:"detail"`
}

func main() {
	// Check the env var to see if we're running as a Lambda function
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(HandleRequest)
	} else {
		cmd.Execute()
	}
}

// HandleRequest is the handler for the Lambda function
func HandleRequest(message EventBridgeMessage) {
	s3bucket := os.Getenv("ReportS3Bucket")
	filename := os.Getenv("ReportNamePattern")
	format := os.Getenv("ReportOutputFormat")
	timezone := os.Getenv("ReportTimezone")
	cmd.GenerateReportFromLambda(message.Detail.StackId, s3bucket, filename, format, timezone)
}
