# Deploy Output Golden Files

This directory contains golden files for testing the deploy command's multi-format output feature.

## Golden Files

### Successful Deployment
- `success-output.json` - JSON format for successful deployment
- `success-output.yaml` - YAML format for successful deployment
- `success-output.csv` - CSV format for successful deployment
- `success-output.md` - Markdown format for successful deployment

### Failed Deployment
- `failure-output.json` - JSON format for failed deployment
- `failure-output.yaml` - YAML format for failed deployment

### No Changes
- `no-changes-output.json` - JSON format for no-changes scenario

### Dry Run
- `dry-run-output.json` - JSON format for dry-run output

## Updating Golden Files

To regenerate golden files when output format changes:

```bash
# Run tests with UPDATE_GOLDEN=1 environment variable
UPDATE_GOLDEN=1 go test ./cmd -run TestGoldenFile
```

## Data Structure

All golden files use realistic sample data that matches actual CloudFormation deployment structures:

- Stack ARN: `arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123`
- Changeset ID: `arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/def456`
- Resource changes: Create S3 bucket, Update Lambda function
- Stack outputs: BucketName, FunctionArn
