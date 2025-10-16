# Decision Log - Transit Gateway Drift Detection

## Overview
This document records all significant decisions made during the development of the Transit Gateway drift detection feature.

---

## Requirements Phase Decisions

### Decision 1: Reuse Existing VPC Route Table Pattern
**Date:** 2025-10-14
**Context:** Need to determine the architectural approach for implementing Transit Gateway route drift detection.
**Decision:** Follow the same pattern used for VPC route table drift detection, integrating into the existing drift command workflow.
**Rationale:**
- Provides consistent user experience across VPC and Transit Gateway resources
- Leverages proven, tested patterns already in the codebase
- Reduces learning curve for both users and developers
- Minimizes code duplication through reuse of existing infrastructure

**Alternatives Considered:**
- Create a separate command for Transit Gateway drift detection
- Build a completely new detection system from scratch

**Status:** Accepted

---

### Decision 2: Exclude Propagated Routes from Drift Detection
**Date:** 2025-10-14
**Context:** Transit Gateway route tables can contain both static and propagated routes.
**Decision:** Only detect drift for static routes; exclude all propagated routes from drift comparison.
**Rationale:**
- Propagated routes are dynamic by design and expected to change based on attachment lifecycle
- They are not explicitly defined in CloudFormation templates as `AWS::EC2::TransitGatewayRoute` resources
- Reporting propagated routes as drift would create false positives and confusion
- This aligns with the principle that drift detection should only compare IaC-defined resources

**Implementation:** Routes with `Type: propagated` will be filtered out before comparison.

**Status:** Accepted

---

### Decision 3: Filter Routes by State (Active/Blackhole Only)
**Date:** 2025-10-14
**Context:** Transit Gateway routes can be in various states (pending, active, blackhole, deleting, deleted, failed).
**Decision:** Only compare routes in 'active' or 'blackhole' states; exclude routes in transient states.
**Rationale:**
- Transient states (pending, deleting, deleted, failed) represent temporary conditions during route lifecycle
- Comparing transient states would create false positives when routes are being created or destroyed
- 'active' and 'blackhole' represent steady-state conditions that should match the template

**Status:** Accepted

---

### Decision 4: Support All Attachment Types Without Special Handling
**Date:** 2025-10-14
**Context:** Transit Gateway routes can target different attachment types (VPC, VPN, Direct Connect Gateway, Peering).
**Decision:** Support all attachment types uniformly using the `TransitGatewayAttachmentId` property without type-specific logic.
**Rationale:**
- All attachment types are referenced by their attachment ID in route definitions
- Type-specific handling would add unnecessary complexity
- CloudFormation templates use the same `AWS::EC2::TransitGatewayRoute` resource for all types
- Drift detection only needs to compare attachment IDs, not understand attachment semantics

**Status:** Accepted

---

### Decision 5: Include Prefix List Destinations with Same Pattern as VPC Routes
**Date:** 2025-10-14
**Context:** Transit Gateway routes can use `DestinationPrefixListId` instead of `DestinationCidrBlock`.
**Decision:** Follow the same prefix list handling as VPC routes: exclude by default unless `--verbose` flag is set, always exclude AWS-managed prefix lists.
**Rationale:**
- Provides consistency between VPC and Transit Gateway drift detection
- Prefix lists are often managed outside CloudFormation templates
- AWS-managed prefix lists (like S3 endpoints) should never be reported as drift
- Users who need prefix list visibility can opt-in with `--verbose`

**Implementation:** Reuse the existing `GetManagedPrefixLists` function and filtering logic from VPC route checking.

**Status:** Accepted

---

### Decision 6: Sequential API Calls (No Parallel Execution)
**Date:** 2025-10-14
**Context:** Determine whether to query AWS APIs for multiple route tables in parallel or sequentially.
**Decision:** Execute AWS API calls sequentially, not in parallel.
**Rationale:**
- Matches the existing pattern used for VPC route table drift detection
- User explicitly stated: "we can't query in parallel (using go func) as that will prevent it from running well"
- Avoids potential issues with rate limiting or API throttling
- Maintains consistent behavior across all drift detection operations

**Note:** While AWS rate limits are typically per-second rather than based on parallelism, this decision respects the existing working pattern and user's explicit guidance.

**Status:** Accepted

---

### Decision 7: Handle CloudFormation Conditions with Empty Route Identifier Pattern
**Date:** 2025-10-14
**Context:** CloudFormation templates may use conditions that prevent routes from being created.
**Decision:** Follow the existing VPC route pattern: when a route's destination cannot be resolved (due to conditions), return an empty string as the route identifier, which will be excluded from drift comparison.
**Rationale:**
- This is how the existing code handles conditional resources (see [cmd/drift.go:321-323](cmd/drift.go#L321-L323))
- Provides consistent behavior across VPC and Transit Gateway routes
- Simplifies implementation by reusing proven logic
- Avoids false positives for routes that shouldn't exist due to conditions

**Implementation:** The template parsing functions will return empty identifiers for unresolvable routes.

**Status:** Accepted

---

### Decision 8: Create New lib/tgw_routetables.go File
**Date:** 2025-10-14
**Context:** Determine where to place Transit Gateway-specific helper functions.
**Decision:** Create a new `lib/tgw_routetables.go` file following patterns from `lib/ec2.go` and `lib/template.go`.
**Rationale:**
- Separates Transit Gateway logic from generic EC2 operations
- Follows the naming convention of existing files (e.g., `lib/nacls.go`, `lib/ec2.go`)
- Makes the codebase easier to navigate and maintain
- Allows for future Transit Gateway features to be added to the same file

**Alternatives Considered:**
- Adding functions to `lib/ec2.go` (rejected: would mix VPC and Transit Gateway concerns)
- Creating `lib/routes.go` (rejected: too generic, doesn't distinguish between route types)

**Status:** Accepted

---

### Decision 9: Handle Multiple Attachments by Using First in List
**Date:** 2025-10-14
**Context:** Transit Gateway routes can have multiple attachments in the TransitGatewayAttachments list.
**Decision:** When a route has multiple attachments, use the first attachment in the list as the route target.
**Rationale:**
- Simplifies comparison logic by treating routes as having a single target
- In practice, Transit Gateway routes typically have one active attachment
- Multiple attachments are rare and represent edge cases
- If multiple attachments exist, the first one represents the primary target

**Future Consideration:** If multiple attachment scenarios become common, this could be revisited to compare all attachments.

**Status:** Accepted

---

### Decision 10: Require Full IAM Permission Set
**Date:** 2025-10-14
**Context:** Determine which IAM permissions are needed for Transit Gateway drift detection.
**Decision:** Document that users need:
- `ec2:DescribeTransitGatewayRouteTables` (primary API call)
- `ec2:DescribeTransitGateways` (for route table validation)
- `ec2:DescribeManagedPrefixLists` (for prefix list filtering)

**Rationale:**
- Matches the permission pattern used for VPC route table drift detection
- Provides complete functionality without requiring users to troubleshoot missing permissions
- Documents expectations clearly upfront

**Status:** Accepted

---

## Questions Answered During Requirements Phase

### Q1: Should we track propagated routes?
**Answer:** No. Propagated routes are dynamic by design and not defined in templates. They should be excluded from drift detection.

### Q2: How should we handle routes in transient states?
**Answer:** Only compare routes in 'active' or 'blackhole' states. Exclude pending, deleting, deleted, and failed states.

### Q3: Should we support all Transit Gateway attachment types?
**Answer:** Yes, but without special handling. All types use the same `TransitGatewayAttachmentId` property.

### Q4: How should prefix list destinations be handled?
**Answer:** Follow the same pattern as VPC routes: exclude by default unless `--verbose`, always exclude AWS-managed.

### Q5: Should we query AWS APIs in parallel?
**Answer:** No. Use sequential execution to match existing VPC route table behavior and avoid potential issues.

### Q6: Where should Transit Gateway helper functions go?
**Answer:** Create a new `lib/tgw_routetables.go` file following patterns from `lib/ec2.go` and `lib/template.go`.

### Q7: How should CloudFormation conditions be handled?
**Answer:** Follow the existing pattern: return empty route identifiers for unresolvable routes, which will be excluded from comparison.

---

## Open Questions

None at this time. All questions raised during requirements review have been answered.

---

---

## Design Phase Decisions

### Decision 11: Use SearchTransitGatewayRoutes API (Not DescribeTransitGatewayRouteTables)
**Date:** 2025-10-14
**Context:** Initial requirements document incorrectly specified `DescribeTransitGatewayRouteTables` API.
**Decision:** Use `SearchTransitGatewayRoutes` API for retrieving Transit Gateway routes.
**Rationale:**
- `SearchTransitGatewayRoutes` returns route details including the `Type` field (static vs propagated)
- `DescribeTransitGatewayRouteTables` only returns route table metadata, not route details
- The `Type` field is essential for filtering out propagated routes
- This API allows filtering by state (active, blackhole) directly

**Implementation:** Updated requirements document to reflect correct API choice.

**Status:** Accepted

---

### Decision 12: No Pagination Implementation Required
**Date:** 2025-10-14
**Context:** Initial design assumed AWS API supported pagination with NextToken.
**Decision:** Do not implement pagination logic for `SearchTransitGatewayRoutes`.
**Rationale:**
- The `SearchTransitGatewayRoutes` API does NOT support pagination
- It returns all matching routes in a single API call
- `MaxResults` parameter is a limit, not a page size
- No `NextToken` field exists in the response
- Typical Transit Gateway route tables have 10-100 routes, well within API limits

**Implementation:** Removed all pagination logic from design. API will be called once per route table.

**Status:** Accepted

---

### Decision 13: Use Type Assertions for Error Handling
**Date:** 2025-10-14
**Context:** Need robust error handling for AWS API calls.
**Decision:** Use type assertions (`errors.As`) instead of string matching for AWS API errors.
**Rationale:**
- String matching (`strings.Contains(err.Error(), "...")`) is fragile and breaks if AWS changes error messages
- Type assertions against `smithy.APIError` provide reliable error code checking
- Allows specific handling for different error scenarios (NotFound, Unauthorized, etc.)
- Follows AWS SDK v2 best practices

**Example:**
```go
var apiErr smithy.APIError
if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidRouteTableID.NotFound" {
    // Handle specifically
}
```

**Status:** Accepted

---

### Decision 14: Use Proper Context with Timeouts
**Date:** 2025-10-14
**Context:** AWS API calls need timeout protection.
**Decision:** Pass `context.Context` with 30-second timeout to all AWS API calls; never use `context.TODO()`.
**Rationale:**
- Prevents API calls from hanging indefinitely
- Allows request cancellation if drift detection is interrupted
- Enables proper resource cleanup
- Follows Go best practices for API calls

**Implementation:**
```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
result, err := svc.SearchTransitGatewayRoutes(ctx, &input)
```

**Status:** Accepted

---

### Decision 15: ECMP Routes - Only Compare First Attachment
**Date:** 2025-10-14
**Context:** Transit Gateway routes can have multiple attachments for Equal-Cost Multi-Path routing.
**Decision:** For routes with multiple attachments, only compare the first attachment in the array. Document this as a known limitation.
**Rationale:**
- Comparing all attachments requires complex logic to determine expected attachment sets
- AWS does not guarantee attachment order in the array
- ECMP is an advanced use case not commonly used
- Simple implementation reduces risk of false positives
- Can be enhanced in future version if users require it

**Known Limitation:** Drift in secondary ECMP paths will not be detected.

**Documentation:** This limitation will be clearly documented in user-facing help text and in the Known Limitations section of the design document.

**Status:** Accepted

---

### Decision 16: Validate Attachment Array Before Access
**Date:** 2025-10-14
**Context:** Routes may have empty `TransitGatewayAttachments` arrays.
**Decision:** Always check array length before accessing elements; handle empty arrays gracefully.
**Rationale:**
- Prevents index out of range panics
- Routes with zero attachments are invalid but may exist in API responses
- Defensive programming prevents runtime crashes

**Implementation:**
```go
if len(route.TransitGatewayAttachments) == 0 {
    return "" // No attachments available
}
```

**Status:** Accepted

---

## Revision History

| Date | Author | Change |
|------|--------|--------|
| 2025-10-14 | Requirements Phase | Initial decision log created with all requirements phase decisions |
| 2025-10-14 | Design Phase | Added decisions 11-16 covering API choice, pagination, error handling, context usage, ECMP limitations, and array validation |
