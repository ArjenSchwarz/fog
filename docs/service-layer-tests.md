# Service Layer Test Coverage

This document summarizes unit tests for the service layer introduced in `cmd/services`.

## AWS Client Mocks
- `cmd/services/aws/mocks_test.go` provides mock implementations for the CloudFormation and S3 clients. The mocks allow tests to control AWS responses without real network calls. Tests verify default behaviour and that custom function overrides are invoked.

## Deployment Service
- `cmd/services/deployment/service_test.go` tests `PrepareDeployment` and `ValidateDeployment` success and failure paths using stubbed template services. It ensures parameters and tags are validated and that invalid templates result in errors.
- `cmd/services/deployment/template_test.go` verifies template loading from the configured directory and basic validation rules.
- `cmd/services/deployment/others_test.go` covers parameter and tag service helpers as well as upload logic.

## Service Factory
- `cmd/services/factory/factory_test.go` ensures the factory injects configuration correctly and panics when the AWS configuration is missing.

Together these tests raise coverage above 85% for the service packages.
