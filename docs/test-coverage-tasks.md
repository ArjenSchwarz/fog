# Test Coverage Tasks

The following tasks outline additional tests required to improve coverage for the library. Each task should include success and failure scenarios and use mocks where needed.

## Identity Center (`lib/identitycenter.go`)
- [ ] Add unit tests for `GetPermissionSetArns`
- [ ] Add unit tests for `GetSSOInstanceArn`
- [ ] Add unit tests for `GetAssignmentArns`
- [ ] Add unit tests for `GetAccountAssignmentArnsForPermissionSet`
- [ ] Add unit tests for `GetAccountIDs`
  - Use mocked `ssoadmin` and `organizations` clients
  - Cover both successful responses and error conditions

## Outputs (`lib/outputs.go`)
- [ ] Add tests for `GetExports`
- [ ] Add tests for `getOutputsForStack`
- [ ] Add tests for `FillImports`
  - Mock CloudFormation interactions
  - Validate behavior with and without import errors

## Resources (`lib/resources.go`)
- [ ] Add tests for `GetResources`
  - Include throttling and error scenarios

## Template Parsing (`lib/template.go`)
- [ ] Add tests for `ParseTemplateString`
- [ ] Add tests for `NaclResourceToNaclEntry`
- [ ] Add tests for `RouteResourceToRoute`
- [ ] Add tests for `ShouldHaveResource`
  - Verify parsing and conversion logic for various input types

## Stacks (`lib/stacks.go`)
- [ ] Implement `TestDeployInfo_GetExecutionTimes`
- [ ] Add tests for helper functions (`stringInSlice`, duration calculations, etc.)
- [ ] Add tests for stack-related methods (`IsReadyForUpdate`, `IsOngoing`, etc.)

