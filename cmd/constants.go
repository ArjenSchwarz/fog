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
package cmd

// Shared string literals used as output column keys and map keys across the
// cmd package. Defining them as named constants keeps the values consistent
// and satisfies the goconst linter.
const (
	colStack       = "Stack"
	colDescription = "Description"
	colCfnName     = "CfnName"
	colStatus      = "Status"
	colReason      = "Reason"
	colDuration    = "Duration"
	colAction      = "Action"
	colReplacement = "Replacement"
	colDetails     = "Details"
	colAccount     = "Account"
	colValue       = "Value"
	colExport      = "Export"
	colRegion      = "Region"
	colStartTime   = "Start time"
	colImported    = "Imported"
	colSuccess     = "Success"
)

// File extension literals.
const (
	extJSON = ".json"
	extYAML = ".yaml"
)

// Shared descriptive literals.
const (
	s3BucketArnDescription = "ARN of the S3 bucket"
)
