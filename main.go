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
package main

import (
	"os"
	"time"

	"github.com/ArjenSchwarz/fog/cmd"
	"github.com/aws/aws-lambda-go/lambda"
)

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
