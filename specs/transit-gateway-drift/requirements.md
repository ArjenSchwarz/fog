# Transit Gateway Drift Detection - Requirements

## Introduction

This feature extends fog's existing drift detection capabilities to support AWS Transit Gateway route tables. Currently, fog provides custom drift detection for VPC route tables that goes beyond CloudFormation's native capabilities by detecting manually added routes that are not defined in the CloudFormation template. This enhancement will bring the same level of detection to Transit Gateway route tables.

CloudFormation's native drift detection only tracks resources explicitly defined in templates. As documented in the research report, manually added Transit Gateway routes are not detected as drift by CloudFormation. This feature will implement custom detection logic (similar to the existing VPC route table implementation in [cmd/drift.go:280-348](cmd/drift.go#L280-L348)) to identify these unmanaged routes.

The implementation will integrate seamlessly with fog's existing drift detection command and output formatting, providing a consistent user experience across both VPC and Transit Gateway route table drift detection.

---

## Requirements

### 1. Transit Gateway Route Table Detection

**User Story:** As a DevOps engineer using fog for drift detection, I want the tool to automatically detect Transit Gateway route tables in my CloudFormation stack, so that I can identify drift in Transit Gateway routing configurations alongside VPC route table drift.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL identify `AWS::EC2::TransitGatewayRouteTable` resources during drift detection execution
2. <a name="1.2"></a>The system SHALL extract the logical resource ID and physical resource ID for each Transit Gateway route table from the CloudFormation stack drift results
3. <a name="1.3"></a>The system SHALL store Transit Gateway route table mappings in a separate data structure from VPC route tables
4. <a name="1.4"></a>The system SHALL process Transit Gateway route tables using the same workflow as VPC route tables (after standard CloudFormation drift detection completes)
5. <a name="1.5"></a>The system SHALL handle stacks containing both VPC route tables and Transit Gateway route tables simultaneously

### 2. Template Route Extraction

**User Story:** As a DevOps engineer, I want fog to parse Transit Gateway routes from my CloudFormation template, so that it can compare template-defined routes against actual AWS state.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL parse the CloudFormation template to extract all `AWS::EC2::TransitGatewayRoute` resources
2. <a name="2.2"></a>The system SHALL associate each Transit Gateway route with its parent route table using the `TransitGatewayRouteTableId` property
3. <a name="2.3"></a>The system SHALL extract the `DestinationCidrBlock` property as the route identifier when present
4. <a name="2.4"></a>The system SHALL extract the `DestinationPrefixListId` property as the route identifier when `DestinationCidrBlock` is not present
5. <a name="2.5"></a>The system SHALL extract the `TransitGatewayAttachmentId` property as the route target for all attachment types (VPC, VPN, Direct Connect Gateway, Peering)
6. <a name="2.6"></a>The system SHALL extract the `Blackhole` property to determine if the route drops traffic
7. <a name="2.7"></a>The system SHALL resolve CloudFormation intrinsic functions (Ref, GetAtt) when processing template routes
8. <a name="2.8"></a>The system SHALL handle CloudFormation parameters and substitute their values when parsing routes
9. <a name="2.9"></a>The system SHALL handle CloudFormation conditions by returning an empty route identifier for unresolvable routes, which will be excluded from drift comparison

### 3. AWS API Route Retrieval

**User Story:** As a DevOps engineer, I want fog to query AWS for the actual routes in Transit Gateway route tables, so that it can detect manually added or modified routes.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL use the AWS EC2 API to retrieve all routes for each Transit Gateway route table
2. <a name="3.2"></a>The system SHALL retrieve routes using the `SearchTransitGatewayRoutes` API call with the physical route table ID and state filters
3. <a name="3.3"></a>The system SHALL NOT implement pagination as the `SearchTransitGatewayRoutes` API returns all matching routes in a single call
4. <a name="3.4"></a>The system SHALL extract the destination CIDR block or prefix list ID from each route as the route identifier
5. <a name="3.5"></a>The system SHALL extract the attachment ID from the route's TransitGatewayAttachments list as the route target
6. <a name="3.6"></a>The system SHALL handle routes with multiple attachments by using the first attachment in the list
7. <a name="3.7"></a>The system SHALL NOT query AWS APIs in parallel (sequential execution) to match existing VPC route table behavior
8. <a name="3.8"></a>The system SHALL handle AWS API errors gracefully and report them to the user

### 4. Propagated Route Filtering

**User Story:** As a DevOps engineer, I want fog to exclude propagated routes from drift detection, so that dynamic routes expected by design are not reported as drift.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL identify propagated routes by checking the route's `Type` field for value `propagated`
2. <a name="4.2"></a>The system SHALL exclude all propagated routes from drift comparison
3. <a name="4.3"></a>The system SHALL only compare routes with Type `static` against template-defined routes
4. <a name="4.4"></a>The system SHALL include prefix list destinations in drift detection and compare them like CIDR block destinations

### 5. Route Comparison Logic

**User Story:** As a DevOps engineer, I want fog to compare template-defined routes against actual AWS routes, so that I can identify added, removed, or modified routes.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL compare routes based on their destination (CIDR block or prefix list ID) as the unique identifier
2. <a name="5.2"></a>The system SHALL only compare routes in 'active' or 'blackhole' states from AWS
3. <a name="5.3"></a>The system SHALL exclude routes in 'pending', 'deleting', 'deleted', or 'failed' states from drift comparison
4. <a name="5.4"></a>The system SHALL detect routes present in AWS but not in the template as "unmanaged routes"
5. <a name="5.5"></a>The system SHALL detect routes present in the template but not in AWS as "removed routes"
6. <a name="5.6"></a>The system SHALL detect routes present in both but with different attachment IDs as "modified routes"
7. <a name="5.7"></a>The system SHALL detect routes with changed blackhole status as "modified routes"
8. <a name="5.8"></a>The system SHALL honor the `drift.ignore-blackholes` configuration setting when comparing blackhole route status
9. <a name="5.9"></a>The system SHALL exclude routes with empty identifiers (unresolvable due to CloudFormation conditions) from the comparison

### 6. Drift Output Formatting

**User Story:** As a DevOps engineer, I want Transit Gateway route drift to be displayed in the same format as VPC route drift, so that I have a consistent experience across all drift detection results.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL add Transit Gateway route drift entries to the same output array as other drift results
2. <a name="6.2"></a>The system SHALL use "Route for TransitGatewayRouteTable {LogicalId}" as the LogicalId field for route drift entries
3. <a name="6.3"></a>The system SHALL use "AWS::EC2::TransitGatewayRoute" as the Type field for route drift entries
4. <a name="6.4"></a>The system SHALL use "MODIFIED" as the ChangeType for unmanaged, removed, or modified routes
5. <a name="6.5"></a>The system SHALL format unmanaged routes with positive inline styling (green)
6. <a name="6.6"></a>The system SHALL format removed routes with warning inline styling (yellow/red)
7. <a name="6.7"></a>The system SHALL format modified routes with standard text
8. <a name="6.8"></a>The system SHALL support the `--separate-properties` flag to output each route drift as a separate row
9. <a name="6.9"></a>The system SHALL group route drifts by route table when `--separate-properties` is not used
10. <a name="6.10"></a>The system SHALL format route details as "Destination: Target (status)" matching VPC route formatting

### 7. Route Detail String Formatting

**User Story:** As a DevOps engineer, I want route details to be presented in a clear, readable format, so that I can quickly understand what has drifted.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL format unmanaged routes as "Unmanaged route: {destination}: {target} {status}"
2. <a name="7.2"></a>The system SHALL format removed routes as "Removed route: {destination}: {target} {status}"
3. <a name="7.3"></a>The system SHALL format modified routes as "Expected: {template_route}<separator>Actual: {aws_route}"
4. <a name="7.4"></a>The system SHALL display the destination CIDR block (e.g., "10.0.0.0/16")
5. <a name="7.5"></a>The system SHALL display the target as the attachment ID (e.g., "tgw-attach-12345678")
6. <a name="7.6"></a>The system SHALL display blackhole routes with target as "blackhole"
7. <a name="7.7"></a>The system SHALL include status "(blackhole)" for blackhole routes if the route state is blackhole
8. <a name="7.8"></a>The system SHALL use the configured output separator between expected and actual values

### 8. Integration with Existing Infrastructure

**User Story:** As a DevOps engineer, I want Transit Gateway drift detection to work seamlessly with fog's existing drift command, so that I don't need to learn new commands or workflows.

**Acceptance Criteria:**

1. <a name="8.1"></a>The system SHALL reuse the existing `fog stack drift` command without requiring new flags or subcommands
2. <a name="8.2"></a>The system SHALL reuse the existing drift detection workflow from [cmd/drift.go:69-182](cmd/drift.go#L69-L182)
3. <a name="8.3"></a>The system SHALL call Transit Gateway route checking after NACL checking and VPC route table checking
4. <a name="8.4"></a>The system SHALL reuse the existing flag system (--stack-name, --results-only, --separate-properties, etc.)
5. <a name="8.5"></a>The system SHALL support all existing output formats (table, CSV, JSON)
6. <a name="8.6"></a>The system SHALL integrate with the existing output settings and separator configuration
7. <a name="8.7"></a>The system SHALL handle errors using the existing error handling patterns
8. <a name="8.8"></a>The system SHALL work when no Transit Gateway route tables exist in the stack without errors

### 9. Configuration Compatibility

**User Story:** As a DevOps engineer, I want to use existing configuration options with Transit Gateway drift detection, so that I have consistent control over drift behavior.

**Acceptance Criteria:**

1. <a name="9.1"></a>The system SHALL respect the `drift.ignore-blackholes` configuration setting for Transit Gateway routes
2. <a name="9.2"></a>The system SHALL support per-stack blackhole ignore patterns if that configuration exists
3. <a name="9.3"></a>The system SHALL use the same configuration file format and loading mechanism as existing drift detection
4. <a name="9.4"></a>The system SHALL NOT require new configuration options for basic Transit Gateway drift detection

### 10. Error Handling and Edge Cases

**User Story:** As a DevOps engineer, I want fog to handle errors and edge cases gracefully, so that drift detection doesn't fail unexpectedly.

**Acceptance Criteria:**

1. <a name="10.1"></a>The system SHALL handle Transit Gateway route tables with zero routes without errors
2. <a name="10.2"></a>The system SHALL handle Transit Gateway route tables with only propagated routes without reporting drift
3. <a name="10.3"></a>The system SHALL handle AWS API errors using type assertions and display specific error messages based on error codes
4. <a name="10.4"></a>The system SHALL handle invalid template references (e.g., !Ref to non-existent resource) gracefully
5. <a name="10.5"></a>The system SHALL handle stacks with no Transit Gateway resources without attempting Transit Gateway API calls
6. <a name="10.6"></a>The system SHALL handle Transit Gateway route tables deleted outside CloudFormation without crashing
7. <a name="10.7"></a>The system SHALL continue processing other route tables if one route table fails to query
8. <a name="10.8"></a>The system SHALL handle routes with empty TransitGatewayAttachments arrays gracefully
9. <a name="10.9"></a>The system SHALL use proper context with timeouts for AWS API calls

### 11. Code Structure and Maintainability

**User Story:** As a developer maintaining fog, I want the Transit Gateway drift detection code to follow existing patterns, so that it's easy to understand and maintain.

**Acceptance Criteria:**

1. <a name="11.1"></a>The system SHALL implement Transit Gateway route checking similar to the VPC route checking function at [cmd/drift.go:280-348](cmd/drift.go#L280-L348)
2. <a name="11.2"></a>The system SHALL add helper functions to a new `lib/tgw_routetables.go` file for Transit Gateway operations following patterns from [lib/ec2.go](lib/ec2.go) and [lib/template.go](lib/template.go)
3. <a name="11.3"></a>The system SHALL separate Transit Gateway route tables in the `separateSpecialCases` function similar to VPC route tables
4. <a name="11.4"></a>The system SHALL create a route-to-string formatter function for Transit Gateway routes following the pattern of `routeToString` at [cmd/drift.go:525-533](cmd/drift.go#L525-L533)
5. <a name="11.5"></a>The system SHALL include unit tests for Transit Gateway route comparison logic with coverage comparable to existing EC2 tests in [lib/ec2_test.go](lib/ec2_test.go)
6. <a name="11.6"></a>The system SHALL include integration tests that validate end-to-end Transit Gateway drift detection using the INTEGRATION=1 environment variable pattern
7. <a name="11.7"></a>The system SHALL follow Go language rules from [language-rules/go.md](language-rules/go.md)
8. <a name="11.8"></a>The system SHALL run `go fmt` and `go test ./...` successfully after implementation

---

## Out of Scope

The following items are explicitly **not** included in this feature:

1. **Propagated route tracking**: Propagated routes are dynamic by design and will not be tracked for drift
2. **Static route priority detection**: Detection of scenarios where static route deletion causes a propagated route to become active
3. **Configuration options for acceptable manual routes**: All manual routes will be reported; filtering will be added in a future enhancement if needed
4. **Cross-stack Transit Gateway drift**: Detection of drift when Transit Gateway resources span multiple CloudFormation stacks
5. **Transit Gateway attachment drift detection**: The research report identified a CloudFormation bug with attachment deletion detection, but fixing CloudFormation's native behavior is out of scope
6. **Automated drift remediation**: This feature only reports drift; it does not modify infrastructure
7. **Parallel AWS API calls**: To avoid rate limiting, API calls will remain sequential
8. **New CLI commands**: All functionality will be accessed through the existing `fog stack drift` command

---

## Success Criteria

This feature will be considered successful when:

1. The `fog stack drift` command detects manually added Transit Gateway routes that are not in the CloudFormation template
2. Drift output for Transit Gateway routes matches the format and style of VPC route drift output
3. All existing drift detection functionality continues to work unchanged
4. The implementation follows fog's architecture patterns and passes all tests
5. Documentation is updated to reflect Transit Gateway support

---

## Dependencies

- AWS SDK v2 EC2 client for `DescribeTransitGatewayRouteTables` API calls
- Existing CloudFormation template parsing infrastructure in `lib` package
- Existing drift detection workflow in [cmd/drift.go](cmd/drift.go)
- Existing output formatting system from `github.com/ArjenSchwarz/go-output`

---

## Assumptions

1. The CloudFormation bug for Transit Gateway attachment deletion (issue #1271) has been resolved by AWS
2. Users have appropriate IAM permissions including:
   - `ec2:SearchTransitGatewayRoutes` (primary API call)
   - `ec2:DescribeManagedPrefixLists` (for prefix list filtering)
3. Transit Gateway route tables are managed by CloudFormation (stack exists and is queryable)
4. CloudFormation templates use `AWS::EC2::TransitGatewayRoute` resources (not inline route definitions or custom resources)
5. For routes with multiple attachments (ECMP), only the first attachment will be compared for drift detection
