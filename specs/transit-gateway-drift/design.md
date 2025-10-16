# Transit Gateway Drift Detection - Design Document

## Overview

This document details the technical design for extending fog's drift detection capabilities to support AWS Transit Gateway route tables. The implementation will follow the established pattern used for VPC route table drift detection, integrating seamlessly into the existing `fog stack drift` command.

### Goals

1. Detect manually added Transit Gateway routes that are not defined in CloudFormation templates
2. Detect removed or modified Transit Gateway routes
3. Provide consistent output formatting with existing VPC route drift detection
4. Maintain backward compatibility with all existing drift detection functionality

### Non-Goals

- Detecting drift in propagated routes (dynamic by design)
- Detecting drift in Transit Gateway attachments (CloudFormation native responsibility)
- Cross-stack Transit Gateway drift detection
- Automated drift remediation

---

## Architecture

### High-Level Flow

```
User runs: fog stack drift --stack-name my-stack

1. Execute CloudFormation drift detection (existing)
2. Retrieve drift results (existing)
3. Separate special cases (MODIFIED: add TGW route tables)
4. Check NACL entries (existing)
5. Check VPC route table routes (existing)
6. Check Transit Gateway route table routes (NEW)
7. Format and display results (existing)
```

### Integration Points

The Transit Gateway drift detection integrates with existing code at these points:

1. **[cmd/drift.go:86](cmd/drift.go#L86)** - `separateSpecialCases()` function
   - Add case for `AWS::EC2::TransitGatewayRouteTable` resource type
   - Return TGW route tables alongside NACL and VPC route table resources

2. **[cmd/drift.go:140-145](cmd/drift.go#L140-L145)** - Main drift execution flow
   - Add call to `checkTransitGatewayRouteTableRoutes()` after VPC route checking
   - Pass same parameters: TGW route tables map, template, parameters, logical-to-physical map, output array, AWS config

3. **[lib/tgw_routetables.go](lib/tgw_routetables.go)** - NEW FILE
   - AWS API operations for Transit Gateway route tables
   - Transit Gateway route comparison logic
   - Template parsing for Transit Gateway routes

4. **[lib/interfaces.go](lib/interfaces.go)** - Interface additions
   - Add `EC2SearchTransitGatewayRoutesAPI` interface for AWS SDK operations

---

## Components and Interfaces

### New Components

#### 1. Transit Gateway Route Checking Function

**Location:** `cmd/drift.go`

**Function Signature:**
```go
func checkTransitGatewayRouteTableRoutes(
    tgwRouteTableResources map[string]string,
    template lib.CfnTemplateBody,
    parameters []types.Parameter,
    logicalToPhysical map[string]string,
    output *format.OutputArray,
    awsConfig config.AWSConfig
)
```

**Purpose:** Main orchestration function for Transit Gateway route drift detection

**Responsibilities:**
1. Iterate through each Transit Gateway route table
2. Call AWS API to retrieve actual routes via `lib.GetTransitGatewayRouteTableRoutes()`
3. Parse template to extract expected routes via `lib.FilterTGWRoutesByLogicalId()`
4. Compare actual vs expected routes
5. Format drift results and add to output array

**Logic Flow:**
```
For each Transit Gateway route table:
  1. Fetch routes from AWS (via lib.GetTransitGatewayRouteTableRoutes)
  2. Filter out:
     - Propagated routes (Type == "propagated")
     - Routes in transient states (pending, deleting, deleted, failed)
  3. Parse template for expected routes (via lib.FilterTGWRoutesByLogicalId)
  4. Compare:
     - Routes in AWS but not in template → Unmanaged (green)
     - Routes in template but not in AWS → Removed (yellow/red)
     - Routes in both but different → Modified (standard)
  5. Format results matching VPC route pattern
  6. Add to output array
```

#### 2. Transit Gateway Route Retrieval

**Location:** `lib/tgw_routetables.go`

**Function Signature:**
```go
func GetTransitGatewayRouteTableRoutes(
    ctx context.Context,
    routeTableId string,
    svc EC2SearchTransitGatewayRoutesAPI
) ([]types.TransitGatewayRoute, error)
```

**Purpose:** Query AWS EC2 API to retrieve all routes for a Transit Gateway route table

**Implementation Details:**
- Use `SearchTransitGatewayRoutes` API operation (returns route details including Type field)
- Apply filters for specific route table ID and route states
- **Note:** This API does NOT support pagination - it returns all matching routes in a single call
- Filter routes in code after retrieval for better control
- Return slice of `types.TransitGatewayRoute` structures
- Handle AWS API errors using type assertions

**AWS API Call:**
```go
// Add timeout to context for API call
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()

input := ec2.SearchTransitGatewayRoutesInput{
    TransitGatewayRouteTableId: aws.String(routeTableId),
    Filters: []types.Filter{
        {
            Name:   aws.String("state"),
            Values: []string{"active", "blackhole"},
        },
    },
}
result, err := svc.SearchTransitGatewayRoutes(ctx, &input)
if err != nil {
    // Use type assertion for error handling
    var apiErr smithy.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.ErrorCode() {
        case "InvalidRouteTableID.NotFound":
            return nil, fmt.Errorf("transit gateway route table %s not found", routeTableId)
        case "UnauthorizedOperation":
            return nil, fmt.Errorf("insufficient IAM permissions: %w", err)
        }
    }
    return nil, err
}
```

#### 3. Template Route Extraction

**Location:** `lib/tgw_routetables.go`

**Function Signature:**
```go
func FilterTGWRoutesByLogicalId(
    logicalId string,
    template CfnTemplateBody,
    params []cfntypes.Parameter,
    logicalToPhysical map[string]string
) map[string]types.TransitGatewayRoute
```

**Purpose:** Extract Transit Gateway routes from CloudFormation template for a specific route table

**Implementation Details:**
1. Iterate through `template.Resources`
2. Filter for resources with `Type == "AWS::EC2::TransitGatewayRoute"`
3. Check `template.ShouldHaveResource(resource)` for condition evaluation
4. Extract `TransitGatewayRouteTableId` property and match against `logicalId`
5. For matching routes, call `TGWRouteResourceToTGWRoute()` to convert to AWS SDK type
6. Return map keyed by destination (CIDR or prefix list ID)

**Key Considerations:**
- Handle CloudFormation intrinsic functions (Ref, GetAtt)
- Resolve parameter values
- Handle conditions by returning empty destination for unresolvable routes

#### 4. Route Resource Conversion

**Location:** `lib/tgw_routetables.go`

**Function Signature:**
```go
func TGWRouteResourceToTGWRoute(
    resource CfnTemplateResource,
    params []cfntypes.Parameter,
    logicalToPhysical map[string]string
) types.TransitGatewayRoute
```

**Purpose:** Convert CloudFormation template route resource to AWS SDK TransitGatewayRoute type

**Implementation Details:**
1. Extract `DestinationCidrBlock` or `DestinationPrefixListId`
2. Extract `TransitGatewayAttachmentId`
3. Extract `Blackhole` property
4. Handle parameter resolution (Ref to parameters)
5. Handle resource references (Ref to other resources)
6. Set appropriate State based on Blackhole property
7. Set Type to "static" (all template routes are static)

**Property Mapping:**
```
CloudFormation Property           → AWS SDK Field
DestinationCidrBlock              → DestinationCidrBlock
DestinationPrefixListId           → PrefixListId
TransitGatewayAttachmentId        → TransitGatewayAttachments[0].TransitGatewayAttachmentId
Blackhole: true                   → State: "blackhole", TransitGatewayAttachments: nil
Blackhole: false                  → State: "active"
(implicit)                        → Type: "static"
```

#### 5. Route Comparison

**Location:** `lib/tgw_routetables.go`

**Function Signature:**
```go
func CompareTGWRoutes(
    route1 types.TransitGatewayRoute,
    route2 types.TransitGatewayRoute,
    blackholeIgnore []string
) bool
```

**Purpose:** Compare two Transit Gateway routes for equality

**Implementation Details:**
- Compare `DestinationCidrBlock` (if present)
- Compare `PrefixListId` (if present)
- Compare attachment IDs from `TransitGatewayAttachments[0].TransitGatewayAttachmentId`
- Compare `State` field
- Handle blackhole ignore list similar to VPC routes
- Return true if routes match, false otherwise

#### 6. Route Destination Extraction

**Location:** `lib/tgw_routetables.go`

**Function Signature:**
```go
func GetTGWRouteDestination(route types.TransitGatewayRoute) string
```

**Purpose:** Extract the destination identifier from a Transit Gateway route

**Implementation Details:**
```go
switch {
case route.DestinationCidrBlock != nil:
    return *route.DestinationCidrBlock
case route.PrefixListId != nil:
    return *route.PrefixListId
default:
    return ""
}
```

#### 7. Route Target Extraction

**Location:** `lib/tgw_routetables.go`

**Function Signature:**
```go
func GetTGWRouteTarget(route types.TransitGatewayRoute) string
```

**Purpose:** Extract the target identifier from a Transit Gateway route

**Implementation Details:**
```go
if route.State == types.TransitGatewayRouteStateBlackhole {
    return "blackhole"
}
// Validate attachment array exists and is not empty
if len(route.TransitGatewayAttachments) == 0 {
    return "" // No attachments available
}
// Use first attachment (ECMP limitation - see Known Limitations)
if route.TransitGatewayAttachments[0].TransitGatewayAttachmentId != nil {
    return *route.TransitGatewayAttachments[0].TransitGatewayAttachmentId
}
return ""
```

**Note:** For routes with multiple attachments (ECMP), only the first attachment is compared. This is a known limitation documented below.

#### 8. Route String Formatter

**Location:** `cmd/drift.go`

**Function Signature:**
```go
func tgwRouteToString(route types.TransitGatewayRoute) string
```

**Purpose:** Format a Transit Gateway route as a human-readable string

**Implementation Details:**
```go
destination := lib.GetTGWRouteDestination(route)
target := lib.GetTGWRouteTarget(route)
statusPart := ""
if route.State == types.TransitGatewayRouteStateBlackhole {
    statusPart = " (blackhole)"
}
return fmt.Sprintf("%s: %s%s", destination, target, statusPart)
```

**Example Output:**
- `10.0.0.0/16: tgw-attach-12345678`
- `10.1.0.0/16: blackhole (blackhole)`
- `pl-123456: tgw-attach-87654321`

#### 9. AWS SDK Interface

**Location:** `lib/interfaces.go`

**Interface Definition:**
```go
// EC2SearchTransitGatewayRoutesAPI defines the EC2 SearchTransitGatewayRoutes operation.
type EC2SearchTransitGatewayRoutesAPI interface {
    SearchTransitGatewayRoutes(
        ctx context.Context,
        params *ec2.SearchTransitGatewayRoutesInput,
        optFns ...func(*ec2.Options)
    ) (*ec2.SearchTransitGatewayRoutesOutput, error)
}
```

**Purpose:** Define interface for AWS SDK EC2 operations to enable mocking in tests

---

## Data Models

### AWS SDK Types (from github.com/aws/aws-sdk-go-v2/service/ec2/types)

#### TransitGatewayRoute
```go
type TransitGatewayRoute struct {
    DestinationCidrBlock *string
    PrefixListId *string
    State TransitGatewayRouteState  // active | blackhole
    Type TransitGatewayRouteType     // static | propagated
    TransitGatewayAttachments []TransitGatewayRouteAttachment
    TransitGatewayRouteTableAnnouncementId *string
}
```

#### TransitGatewayRouteAttachment
```go
type TransitGatewayRouteAttachment struct {
    ResourceId *string
    ResourceType TransitGatewayAttachmentResourceType  // vpc | vpn | direct-connect-gateway | connect | peering
    TransitGatewayAttachmentId *string
}
```

#### TransitGatewayRouteState
```go
type TransitGatewayRouteState string

const (
    TransitGatewayRouteStateActive    TransitGatewayRouteState = "active"
    TransitGatewayRouteStateBlackhole TransitGatewayRouteState = "blackhole"
    TransitGatewayRouteStateDeleting  TransitGatewayRouteState = "deleting"
    TransitGatewayRouteStateDeleted   TransitGatewayRouteState = "deleted"
    TransitGatewayRouteStatePending   TransitGatewayRouteState = "pending"
)
```

#### TransitGatewayRouteType
```go
type TransitGatewayRouteType string

const (
    TransitGatewayRouteTypeStatic     TransitGatewayRouteType = "static"
    TransitGatewayRouteTypePropagated TransitGatewayRouteType = "propagated"
)
```

### Internal Data Structures

#### Route Table Mapping
```go
// Map of logical resource ID to physical resource ID
tgwRouteTableResources := map[string]string{
    "MyTGWRouteTable": "tgw-rtb-0123456789abcdef0",
}
```

#### Template Routes Map
```go
// Map of route destination to TransitGatewayRoute
templateRoutes := map[string]types.TransitGatewayRoute{
    "10.0.0.0/16": {...},
    "10.1.0.0/16": {...},
    "pl-123456": {...},
}
```

#### Drift Output Structure
```go
content := map[string]any{
    "LogicalId":  "Route for TransitGatewayRouteTable MyTGWRouteTable",
    "Type":       "AWS::EC2::TransitGatewayRoute",
    "ChangeType": "MODIFIED",
    "Details":    "Unmanaged route: 10.2.0.0/16: tgw-attach-98765432",
}
```

---

## Error Handling

### AWS API Errors

**Scenario:** Transit Gateway route table not found
```go
var apiErr smithy.APIError
if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidRouteTableID.NotFound" {
    // Log warning and continue with next route table
    log.Printf("Transit Gateway route table %s not found, skipping", routeTableId)
    continue
}
```

**Scenario:** Permission denied
```go
var apiErr smithy.APIError
if errors.As(err, &apiErr) && apiErr.ErrorCode() == "UnauthorizedOperation" {
    return fmt.Errorf("insufficient IAM permissions: %w", err)
}
```

**Scenario:** API throttling
```go
// AWS SDK v2 handles retries automatically with exponential backoff
// No special handling needed beyond standard error propagation
// Context timeout prevents indefinite retries
```

**Scenario:** Context timeout
```go
if errors.Is(err, context.DeadlineExceeded) {
    return fmt.Errorf("API call timed out after 30 seconds: %w", err)
}
```

### Template Parsing Errors

**Scenario:** Invalid Ref in template
```go
// When a Ref cannot be resolved, return empty string as destination
// This causes the route to be skipped in comparison (matches VPC pattern)
if refValue == "" {
    return "" // Empty destination will be skipped
}
```

**Scenario:** Missing property in template
```go
// Handle nil properties gracefully with default values
if resource.Properties["DestinationCidrBlock"] == nil &&
   resource.Properties["DestinationPrefixListId"] == nil {
    return "" // Skip route with no destination
}
```

### Edge Cases

**Scenario:** Route table with zero routes
```go
// No special handling needed - loop will simply not execute
// No error should be raised for empty route tables
```

**Scenario:** Route table with only propagated routes
```go
// All routes filtered out, no drift reported
// This is expected behavior
```

**Scenario:** Route in transient state
```go
// Filter routes by state and type before comparison
if route.Type == types.TransitGatewayRouteTypePropagated {
    continue // Skip propagated routes
}
if route.State != types.TransitGatewayRouteStateActive &&
   route.State != types.TransitGatewayRouteStateBlackhole {
    continue // Skip transient states (pending, deleting, deleted)
}
```

**Scenario:** Route with empty attachments array
```go
// GetTGWRouteTarget handles this gracefully by returning empty string
// These routes will be skipped in comparison as they have no valid target
```

---

## Testing Strategy

### Unit Tests

#### Test File: `lib/tgw_routetables_test.go`

**Test Cases:**

1. **TestGetTransitGatewayRouteTableRoutes**
   - Mock AWS API response with multiple routes
   - Verify correct routes returned
   - Test error handling with type assertions
   - Test context timeout handling
   - Test empty route table response

2. **TestFilterTGWRoutesByLogicalId**
   - Parse template with multiple Transit Gateway routes
   - Filter by specific logical ID
   - Verify correct routes returned
   - Test parameter resolution
   - Test Ref resolution

3. **TestTGWRouteResourceToTGWRoute**
   - Convert template resource to AWS SDK type
   - Test CIDR block routes
   - Test prefix list routes
   - Test blackhole routes
   - Test attachment ID routes
   - Test parameter substitution

4. **TestCompareTGWRoutes**
   - Compare identical routes → true
   - Compare routes with different CIDR → false
   - Compare routes with different attachment ID → false
   - Compare routes with different state → false
   - Test blackhole ignore list

5. **TestGetTGWRouteDestination**
   - Extract CIDR block destination
   - Extract prefix list destination
   - Handle nil destinations

6. **TestGetTGWRouteTarget**
   - Extract attachment ID target
   - Extract blackhole target
   - Handle empty TransitGatewayAttachments array
   - Handle nil attachment ID pointer
   - Handle routes with multiple attachments (ECMP)

**Mock Implementation:**
```go
type mockEC2SearchTransitGatewayRoutesAPI struct {
    routes []types.TransitGatewayRoute
    err    error
}

func (m *mockEC2SearchTransitGatewayRoutesAPI) SearchTransitGatewayRoutes(
    ctx context.Context,
    params *ec2.SearchTransitGatewayRoutesInput,
    optFns ...func(*ec2.Options),
) (*ec2.SearchTransitGatewayRoutesOutput, error) {
    if m.err != nil {
        return nil, m.err
    }
    // Verify context is passed and not TODO
    if ctx == nil {
        return nil, fmt.Errorf("context must not be nil")
    }
    return &ec2.SearchTransitGatewayRoutesOutput{
        Routes: m.routes,
    }, nil
}
```

### Integration Tests

#### Test File: `cmd/drift_integration_test.go`

**Test Cases:**

1. **TestTransitGatewayDriftDetection_EndToEnd**
   - Use `//go:build integration` tag
   - Create complete stack drift scenario
   - Mock CloudFormation API responses
   - Mock EC2 API responses for Transit Gateway routes
   - Verify drift output format
   - Verify correct routes flagged as drifted

2. **TestTransitGatewayDrift_UnmanagedRoutes**
   - Template defines 2 routes
   - AWS returns 3 routes (1 unmanaged)
   - Verify unmanaged route detected with green styling

3. **TestTransitGatewayDrift_RemovedRoutes**
   - Template defines 3 routes
   - AWS returns 2 routes (1 removed)
   - Verify removed route detected with warning styling

4. **TestTransitGatewayDrift_PropagatedRoutesIgnored**
   - AWS returns mix of static and propagated routes
   - Verify only static routes compared
   - Verify propagated routes don't appear in drift output

5. **TestTransitGatewayDrift_PrefixListHandling**
   - Template defines prefix list route
   - Verify prefix list route is compared like CIDR routes
   - Test drift detection for prefix list routes

6. **TestTransitGatewayDrift_ECMPRoutes**
   - Route with multiple attachments
   - Verify only first attachment compared
   - Document expected behavior for ECMP drift detection

**Test Execution:**
```bash
# Run only integration tests
INTEGRATION=1 go test ./cmd -run TestTransitGateway -v

# Run all tests including integration
INTEGRATION=1 go test ./... -v
```

### Manual Testing Checklist

- [ ] Run drift detection on stack with Transit Gateway routes
- [ ] Manually add route via AWS Console, verify detection
- [ ] Manually remove route via AWS Console, verify detection
- [ ] Manually modify route attachment, verify detection
- [ ] Test with stack containing no Transit Gateway resources
- [ ] Test with stack containing Transit Gateway but no route tables
- [ ] Test with route table having only propagated routes
- [ ] Test with prefix list routes
- [ ] Test all output formats (table, CSV, JSON)
- [ ] Test --separate-properties flag
- [ ] Verify output matches VPC route drift format

---

## Performance Considerations

### API Call Optimization

**Sequential Execution:**
- Transit Gateway route tables queried sequentially (not in parallel)
- Follows existing VPC route table pattern
- Prevents potential rate limiting issues

**No Special Caching:**
- Unlike VPC routes, Transit Gateway routes do not require managed prefix list filtering
- All prefix list destinations are compared directly like CIDR blocks

**No Pagination Required:**
- `SearchTransitGatewayRoutes` API returns all matching routes in a single call
- No `NextToken` or pagination mechanism exists for this API
- API uses `MaxResults` as a limit, not a page size
- Typical route tables have 10-100 routes, well within API limits

### Memory Usage

**Data Structure Size:**
- Each TransitGatewayRoute ~100-200 bytes
- Typical route table: 10-50 routes
- Maximum practical: 1000 routes per table
- Multiple route tables: Scale linearly

**Template Parsing:**
- Template loaded once, shared across all checks
- Route resources filtered by logical ID
- Memory usage proportional to template size

### Time Complexity

**Per Route Table:**
1. AWS API call: O(1) - single API operation returning all routes
2. Template filtering: O(n) where n = total template resources
3. Route comparison: O(m) where m = number of routes
4. Total: O(n + m)

**Total Execution:**
- Dominated by AWS API call latency (~100-500ms per table typical)
- Multiple route tables: Sequential execution
- 5 route tables: ~1-3 seconds additional time
- No pagination overhead since API returns all routes at once

---

## Security Considerations

### IAM Permissions

**Required Permissions:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:SearchTransitGatewayRoutes",
        "ec2:DescribeTransitGatewayRouteTables",
        "ec2:DescribeManagedPrefixLists"
      ],
      "Resource": "*"
    }
  ]
}
```

**Permission Validation:**
- No automatic permission validation
- Users receive clear error message if permissions missing
- Error message includes specific permission needed

### Data Exposure

**Sensitive Information in Output:**
- Transit Gateway attachment IDs (not sensitive)
- CIDR blocks (network information, may be sensitive)
- Route table IDs (not sensitive)

**No Credentials in Output:**
- No AWS credentials exposed
- No customer data beyond network configuration
- Safe for logging and display

### API Rate Limiting

**Mitigation:**
- Sequential API calls reduce rate limit risk
- AWS SDK v2 automatic retry with exponential backoff
- No aggressive polling or batch operations

---

## Migration and Compatibility

### Backward Compatibility

**Existing Functionality:**
- No changes to VPC route drift detection
- No changes to NACL drift detection
- No changes to CloudFormation native drift detection
- All existing flags and options continue to work

**Output Format:**
- Transit Gateway routes use same format as VPC routes
- Consistent styling (green for unmanaged, yellow/red for removed)
- Consistent ChangeType ("MODIFIED" for all drift)
- Type field set to "AWS::EC2::TransitGatewayRoute"

### Configuration Compatibility

**Existing Settings:**
- `drift.ignore-blackholes` applies to Transit Gateway routes
- All output format settings apply consistently

**No New Configuration Required:**
- Feature enabled automatically when Transit Gateway resources detected
- No opt-in or opt-out mechanism needed
- No breaking changes to existing configurations

---

## Known Limitations

### ECMP Route Detection

**Limitation:** For Transit Gateway routes with multiple attachments (Equal-Cost Multi-Path routing), only the first attachment in the `TransitGatewayAttachments` array is compared during drift detection.

**Impact:**
- If a route has multiple attachments and drift occurs in the second or subsequent attachments, it will NOT be detected
- AWS does not guarantee the order of attachments in the array, so the "first" attachment may vary across API calls
- This affects ECMP configurations where traffic is distributed across multiple paths

**Workaround:** Users relying on ECMP should:
1. Verify ECMP configurations manually when drift is suspected
2. Use AWS Console or CLI to inspect all attachments for critical routes
3. Consider defining each ECMP path as a separate route in CloudFormation (if AWS allows)

**Future Enhancement:** A future version could implement full ECMP drift detection by comparing all attachments, but this requires:
- Determining the expected set of attachments from the template
- Handling attachment order variations
- Deciding how to report drift when attachment sets differ

**Documentation:** This limitation will be clearly documented in user-facing help text and README.

### API Rate Limits

**Limitation:** Sequential API execution means drift detection on many Transit Gateway route tables may be slow.

**Impact:** Stacks with 20+ Transit Gateway route tables may take 10+ seconds to complete drift detection.

**Mitigation:** This is an acceptable trade-off for reliability. Future versions could implement intelligent parallelization with rate limit backoff.

---

## Future Enhancements

### Potential Improvements

1. **Parallel API Calls:**
   - Could improve performance for many route tables
   - Requires testing rate limit behavior
   - Need careful error handling and coordination

2. **Cross-Stack Detection:**
   - Detect drift when Transit Gateway resources span multiple stacks
   - Requires stack discovery and correlation
   - Complex to implement reliably

3. **Attachment Name Resolution:**
   - Display VPC names instead of attachment IDs
   - Requires additional API calls (DescribeTransitGatewayVpcAttachments)
   - Trade-off between readability and performance

4. **Propagated Route Tracking:**
   - Optional tracking of expected propagated routes
   - Requires modeling expected propagations
   - May introduce false positives

5. **Drift Remediation:**
   - Automated import of drifted routes
   - Generate CloudFormation template updates
   - Out of scope for initial implementation

### Extension Points

**Plugin Architecture:**
- Additional route table types could follow same pattern
- AWS::EC2::VPCPeeringConnection routes
- AWS::EC2::LocalGatewayRouteTable routes

**Custom Filters:**
- Allow users to define route ignore patterns
- Filter routes by destination CIDR range
- Configuration-driven filtering

---

## Appendix

### AWS API Reference

- [SearchTransitGatewayRoutes](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SearchTransitGatewayRoutes.html)
- [DescribeTransitGatewayRouteTables](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeTransitGatewayRouteTables.html)
- [AWS::EC2::TransitGatewayRoute](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroute.html)

### Code References

- Existing VPC route checking: [cmd/drift.go:280-348](cmd/drift.go#L280-L348)
- Existing route comparison: [lib/ec2.go:97-152](lib/ec2.go#L97-L152)
- Existing template filtering: [lib/template.go](lib/template.go)
- Existing interfaces: [lib/interfaces.go](lib/interfaces.go)

### Related Documentation

- [Research Report](RESEARCH_REPORT.md)
- [Requirements](requirements.md)
- [Decision Log](decision_log.md)
