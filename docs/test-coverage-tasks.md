# Test Coverage Tasks

The following tasks outline additional tests required to improve coverage for the library. Each task should include success and failure scenarios and use mocks where needed.

## Coverage Summary
- **Current coverage**: 61.9% (improved from 55.1%)
- **Recent improvements**: +6.8 percentage points

## Identity Center (`lib/identitycenter.go`)
- [x] Add unit tests for `GetPermissionSetArns` - ✅ Complete
- [x] Add unit tests for `GetSSOInstanceArn` - ✅ Complete
- [x] Add unit tests for `GetAssignmentArns` - ✅ Complete
- [x] Add unit tests for `GetAccountAssignmentArnsForPermissionSet` - ✅ Complete
- [x] Add unit tests for `GetAccountIDs` - ✅ Complete
  - Use mocked `ssoadmin` and `organizations` clients
  - Cover both successful responses and error conditions

## Outputs (`lib/outputs.go`)
- [x] Add tests for `GetExports` - ✅ Complete
- [x] Add tests for `getOutputsForStack` - ✅ Complete
- [x] Add tests for `FillImports` - ✅ Complete
  - Mock CloudFormation interactions
  - Validate behavior with and without import errors

## Resources (`lib/resources.go`)
- [x] Add tests for `GetResources` - ✅ Complete
  - Include throttling and error scenarios

## Template Parsing (`lib/template.go`)
- [x] Add tests for `ParseTemplateString` - ✅ Complete
- [x] Add tests for `NaclResourceToNaclEntry` - ✅ Complete
- [x] Add tests for `RouteResourceToRoute` - ✅ Complete
- [x] Add tests for `ShouldHaveResource` - ✅ Complete
- [x] Add tests for `FilterNaclEntriesByLogicalId` - ✅ New (100% coverage)
- [x] Add tests for `FilterRoutesByLogicalId` - ✅ New (100% coverage)
- [x] Add tests for `CfnTemplateTransform.Value()` - ✅ New (100% coverage)
- [x] Add tests for `CfnTemplateTransform.UnmarshalJSON()` - ✅ New (91.7% coverage)

## Stacks (`lib/stacks.go`)
- [x] Implement `TestDeployInfo_GetExecutionTimes` - ✅ Complete
- [x] Add tests for helper functions (`stringInSlice`, duration calculations, etc.) - ✅ Complete
- [x] Add tests for stack-related methods (`IsReadyForUpdate`, `IsOngoing`, etc.) - ✅ Complete
- [x] Add tests for `ChangesetType` - ✅ New (100% coverage)
- [x] Add tests for `GetStack` - ✅ New (100% coverage)
- [x] Add tests for `ParseParameterString` - ✅ New (100% coverage)
- [x] Add tests for `ParseTagString` - ✅ New (100% coverage)
- [x] Add tests for `ParseDeploymentFile` - ✅ New (90.9% coverage)

