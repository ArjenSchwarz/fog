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
- [ ] Add tests for `GetCfnStacks` - coverage: 0%
  - Mock paginated CloudFormation DescribeStacks responses
  - Test wildcard stack name filtering
  - Test single stack retrieval
  - Test output imports handling
- [ ] Add tests for `GetParametersMap` - coverage: 0%
  - Test converting CloudFormation parameters to map
- [ ] Add tests for sorting helpers (ReverseEvents, SortStacks) - coverage: 0%
  - Test Len, Less, Swap methods for both types
- [ ] Add tests for remaining uncovered functions:
  - `LoadDeploymentFile` (0%)
  - `CreateChangeSet` (0%)
  - `WaitUntilChangesetDone` (0%)
  - `AddChangeset` (0%)
  - `GetChangeset` (0%)
  - `GetEvents` (0%)
  - `GetSuccessStates` (0%)
  - `GetEventSummaries` (0%)
  - `DeleteStack` (0%)

## Changesets (`lib/changesets.go`)
- [x] Add tests for `AddChange` - ✅ Complete (100% coverage)
- [x] Add tests for `GenerateChangesetUrl` - ✅ Complete (100% coverage)
- [x] Add tests for `GetDangerDetails` - ✅ Complete (100% coverage)
- [ ] Add tests for `DeleteChangeset` - coverage: 0%
  - Mock CloudFormation DeleteChangeSet API
  - Test success and failure scenarios
- [ ] Add tests for `DeployChangeset` - coverage: 0%
  - Mock CloudFormation ExecuteChangeSet API
  - Test success and error handling
- [ ] Add tests for `GetStack` - coverage: 0%
  - Test calling GetStack with changeset's StackID
- [ ] Add tests for `GetStackAndChangesetFromURL` - coverage: 0%
  - Test URL parsing for stack and changeset IDs
  - Test URL decoding and query parameter extraction

## Drift Detection (`lib/drift.go`)
- [ ] Add tests for `StartDriftDetection` - coverage: 0%
  - Mock CloudFormation DetectStackDrift API
  - Test successful detection start
  - Test error handling
- [ ] Add tests for `WaitForDriftDetectionToFinish` - coverage: 0%
  - Mock CloudFormation DescribeStackDriftDetectionStatus API
  - Test waiting logic with in-progress status
  - Test immediate completion
  - Test error scenarios
- [ ] Add tests for `GetDefaultStackDrift` - coverage: 0%
  - Mock paginated DescribeStackResourceDrifts responses
  - Test collecting all drift results
  - Test error handling
- [ ] Add tests for `GetUncheckedStackResources` - coverage: 0%
  - Test filtering resources by checked list
  - Use existing GetResources mock infrastructure
- [ ] Add tests for `GetResource` - coverage: 0%
  - Mock CloudControl GetResource API
  - Test success and error scenarios
- [ ] Add tests for `ListAllResources` - coverage: 0%
  - Test special cases for SSO PermissionSet and Assignment
  - Test generic resource listing

## EC2 Functions (`lib/ec2.go`)
- [x] Add tests for `GetNacl` - ✅ Complete (100% coverage)
- [x] Add tests for `CompareNaclEntries` - ✅ Complete (100% coverage)
- [x] Add tests for `CompareRoutes` - ✅ Complete (97.1% coverage)
- [x] Add tests for helper functions - ✅ Complete (100% coverage)
- [ ] Add tests for `GetRouteTable` - coverage: 0%
  - Mock EC2 DescribeRouteTables API
  - Test successful route table retrieval
  - Test error handling
- [ ] Improve coverage for `GetManagedPrefixLists` from 80% to 100%
  - Add edge case tests

## Files (`lib/files.go`)
- [x] Add tests for `ReadFile` - ✅ Complete (93.8% coverage)
- [x] Add tests for `ReadTemplate` - ✅ Complete (100% coverage)
- [x] Add tests for `ReadTagsfile` - ✅ Complete (100% coverage)
- [x] Add tests for `ReadParametersfile` - ✅ Complete (100% coverage)
- [x] Add tests for `ReadDeploymentFile` - ✅ Complete (100% coverage)
- [x] Add tests for `YamlToJson` - ✅ Complete (87.5% coverage)
- [x] Add tests for `convertMapInterfaceToMapString` - ✅ Complete (100% coverage)
- [ ] Add tests for `UploadTemplate` - coverage: 0%
  - Mock S3 PutObject API
  - Test successful upload
  - Test error handling
- [ ] Improve coverage for `RunPrechecks` from 35% to >80%
  - Test successful precheck execution
  - Test command output capture
  - Test failed precheck scenarios

## Template (`lib/template.go`)
- [x] Add tests for `ParseTemplateString` - ✅ Complete (87.5% coverage)
- [x] Add tests for `NaclResourceToNaclEntry` - ✅ Complete (85.2% coverage)
- [x] Add tests for `RouteResourceToRoute` - ✅ Complete (100% coverage)
- [x] Add tests for `ShouldHaveResource` - ✅ Complete (100% coverage)
- [x] Add tests for `FilterNaclEntriesByLogicalId` - ✅ Complete (100% coverage)
- [x] Add tests for `FilterRoutesByLogicalId` - ✅ Complete (100% coverage)
- [x] Add tests for `CfnTemplateTransform.Value()` - ✅ Complete (100% coverage)
- [x] Add tests for `CfnTemplateTransform.UnmarshalJSON()` - ✅ Complete (91.7% coverage)
- [ ] Add tests for `GetTemplateBody` - coverage: 0%
  - Mock CloudFormation GetTemplate API
  - Test successful template retrieval and parsing
  - Test error handling
- [ ] Improve coverage for `customRefHandler` from 56.2% to >80%
  - Test more pseudo-parameter cases
  - Test parameter default resolution
  - Test nested template lookups

## Interfaces (`lib/interfaces.go`)
- Interface definitions only - no tests needed (interfaces don't have executable code)

