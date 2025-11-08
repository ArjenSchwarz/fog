### Deployment Summary

| Status | Stack ARN | Changeset | Start Time | End Time | Duration |
| --- | --- | --- | --- | --- | --- |
| UPDATE\_COMPLETE | arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123 | arn:aws:cloudformation:us-east-1:123456789012:changeSet/test-changeset/def456 | 2025-11-07T10:00:00Z | 2025-11-07T10:05:30Z | 5m30s |


### Planned Changes

| Action | LogicalID | Type | ResourceID | Replacement |
| --- | --- | --- | --- | --- |
| Add | MyBucket | AWS::S3::Bucket | my-test-bucket-123 | False |
| Modify | MyFunction | AWS::Lambda::Function | test-stack-MyFunction-ABC123 | False |


### Stack Outputs

| OutputKey | OutputValue | Description |
| --- | --- | --- |
| BucketName | my-test-bucket-123 | The S3 bucket name |
| FunctionArn | arn:aws:lambda:us-east-1:123456789012:function:test-stack-MyFunction-ABC123 | The Lambda function ARN |

