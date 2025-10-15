---
references:
    - specs/transit-gateway-drift/requirements.md
    - specs/transit-gateway-drift/design.md
    - specs/transit-gateway-drift/decision_log.md
---
# Transit Gateway Drift Detection - Implementation Tasks

## Foundation - Interfaces and Types

- [x] 1. Add AWS SDK Interface for SearchTransitGatewayRoutes
  - Add EC2SearchTransitGatewayRoutesAPI interface to lib/interfaces.go
  - Follow existing pattern from EC2DescribeRouteTablesAPI
  - Include SearchTransitGatewayRoutes method signature with context, params, and options
  - Requirements: [3.2](requirements.md#3.2)
  - References: lib/interfaces.go

- [x] 2. Create lib/tgw_routetables.go with basic structure
  - Create new file lib/tgw_routetables.go
  - Add package declaration and imports
  - Add initial function stubs: GetTransitGatewayRouteTableRoutes, GetTGWRouteDestination, GetTGWRouteTarget
  - Follow coding patterns from lib/ec2.go
  - Requirements: [11.2](requirements.md#11.2)
  - References: lib/ec2.go, lib/template.go

## Core Functions - Route Helpers

- [x] 3. Write unit tests for GetTGWRouteDestination
  - Create lib/tgw_routetables_test.go
  - Test extraction of DestinationCidrBlock
  - Test extraction of PrefixListId when CIDR is nil
  - Test handling of nil destinations (return empty string)
  - Use table-driven test pattern
  - Requirements: [3.4](requirements.md#3.4), [5.1](requirements.md#5.1)
  - References: lib/ec2_test.go

- [x] 4. Implement GetTGWRouteDestination function
  - Implement function in lib/tgw_routetables.go
  - Return DestinationCidrBlock if present
  - Return PrefixListId if CIDR is nil
  - Return empty string if both are nil
  - Run tests to verify implementation
  - Requirements: [3.4](requirements.md#3.4), [5.1](requirements.md#5.1)

- [x] 5. Write unit tests for GetTGWRouteTarget
  - Test extraction of attachment ID from first attachment
  - Test blackhole state returns 'blackhole'
  - Test empty TransitGatewayAttachments array returns empty string
  - Test nil attachment ID pointer returns empty string
  - Test routes with multiple attachments (ECMP) uses first attachment
  - Requirements: [3.5](requirements.md#3.5), [3.6](requirements.md#3.6), [5.2](requirements.md#5.2), [10.8](requirements.md#10.8)
  - References: lib/tgw_routetables_test.go

- [x] 6. Implement GetTGWRouteTarget function
  - Implement function in lib/tgw_routetables.go
  - Check if State is blackhole, return 'blackhole'
  - Validate TransitGatewayAttachments array length
  - Return first attachment ID if available
  - Handle nil pointer gracefully
  - Run tests to verify implementation including ECMP edge case
  - Requirements: [3.5](requirements.md#3.5), [3.6](requirements.md#3.6), [5.2](requirements.md#5.2), [10.8](requirements.md#10.8)

## Core Functions - AWS API Integration

- [x] 7. Write unit tests for GetTransitGatewayRouteTableRoutes
  - Create mock EC2SearchTransitGatewayRoutesAPI implementation
  - Test successful route retrieval with multiple routes
  - Test error handling with type assertions (InvalidRouteTableID.NotFound, UnauthorizedOperation)
  - Test context timeout handling
  - Test empty route table response
  - Verify context is passed (not nil or TODO)
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [10.3](requirements.md#10.3), [10.9](requirements.md#10.9)
  - References: lib/ec2_test.go

- [x] 8. Implement GetTransitGatewayRouteTableRoutes function
  - Add context.WithTimeout wrapping the passed context
  - Build SearchTransitGatewayRoutesInput with route table ID and state filters
  - Call API with context
  - Use errors.As for smithy.APIError type assertions
  - Handle specific error codes: InvalidRouteTableID.NotFound, UnauthorizedOperation
  - Handle context.DeadlineExceeded
  - Return routes slice on success
  - Run tests to verify implementation
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [10.3](requirements.md#10.3), [10.9](requirements.md#10.9)

## Template Parsing

- [ ] 9. Write unit tests for TGWRouteResourceToTGWRoute
  - Test conversion of CloudFormation resource to TransitGatewayRoute type
  - Test extraction of DestinationCidrBlock
  - Test extraction of DestinationPrefixListId
  - Test extraction of TransitGatewayAttachmentId
  - Test extraction of Blackhole property
  - Test parameter resolution (Ref to parameters)
  - Test resource references (Ref to other resources)
  - Test handling of nil/missing properties
  - Requirements: [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8)
  - References: lib/template_test.go

- [ ] 10. Implement TGWRouteResourceToTGWRoute function
  - Extract DestinationCidrBlock or DestinationPrefixListId from Properties
  - Handle string values and map[string]any (Ref) for destinations
  - Extract TransitGatewayAttachmentId, handle Ref resolution
  - Extract Blackhole property
  - Set State based on Blackhole (blackhole vs active)
  - Set Type to 'static' (all template routes are static)
  - Build and return TransitGatewayRoute struct
  - Return empty destination if unresolvable (for condition handling)
  - Run tests to verify implementation
  - Requirements: [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8), [2.9](requirements.md#2.9)

- [ ] 11. Write unit tests for FilterTGWRoutesByLogicalId
  - Test parsing template with multiple Transit Gateway routes
  - Test filtering by specific logical ID
  - Test parameter resolution
  - Test Ref resolution for TransitGatewayRouteTableId
  - Test that only routes matching the logical ID are returned
  - Test map keyed by destination (CIDR or prefix list)
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8)
  - References: lib/template_test.go

- [ ] 12. Implement FilterTGWRoutesByLogicalId function
  - Create result map[string]types.TransitGatewayRoute
  - Iterate through template.Resources
  - Filter for Type == 'AWS::EC2::TransitGatewayRoute'
  - Check template.ShouldHaveResource(resource) for conditions
  - Extract TransitGatewayRouteTableId property, handle REF: prefix
  - Compare with logicalId parameter
  - Call TGWRouteResourceToTGWRoute for matching routes
  - Use GetTGWRouteDestination as map key
  - Run tests to verify implementation
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.7](requirements.md#2.7), [2.8](requirements.md#2.8), [2.9](requirements.md#2.9)

## Route Comparison Logic

- [ ] 13. Write unit tests for CompareTGWRoutes
  - Test comparing identical routes (should return true)
  - Test routes with different DestinationCidrBlock (should return false)
  - Test routes with different PrefixListId (should return false)
  - Test routes with different attachment IDs (should return false)
  - Test routes with different State (should return false)
  - Test blackhole ignore list handling
  - Test nil pointer handling for all fields
  - Requirements: [5.1](requirements.md#5.1), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7), [5.8](requirements.md#5.8)
  - References: lib/ec2_test.go

- [ ] 14. Implement CompareTGWRoutes function
  - Compare DestinationCidrBlock using stringPointerValueMatch helper
  - Compare PrefixListId using stringPointerValueMatch helper
  - Extract and compare attachment IDs from TransitGatewayAttachments[0]
  - Compare State fields
  - Handle blackhole ignore list (check if route should be ignored)
  - Return true if all fields match, false otherwise
  - Run tests to verify implementation
  - Requirements: [5.1](requirements.md#5.1), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7), [5.8](requirements.md#5.8)

## Command Integration

- [ ] 15. Update separateSpecialCases function in cmd/drift.go
  - Add case for 'AWS::EC2::TransitGatewayRouteTable' in switch statement
  - Add tgwRouteTableResources map[string]string variable
  - Store logical-to-physical mapping for TGW route tables
  - Return tgwRouteTableResources as fourth return value (breaking change)
  - Update function signature to return four maps
  - Update all callers of separateSpecialCases to handle fourth return value
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3)
  - References: cmd/drift.go

- [ ] 16. Implement tgwRouteToString formatter function in cmd/drift.go
  - Create function following pattern of routeToString
  - Call lib.GetTGWRouteDestination for destination
  - Call lib.GetTGWRouteTarget for target
  - Add '(blackhole)' status if State is blackhole
  - Format as '{destination}: {target} {status}'
  - Return formatted string
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.6](requirements.md#7.6), [7.7](requirements.md#7.7), [7.8](requirements.md#7.8)
  - References: cmd/drift.go

- [ ] 17. Implement checkTransitGatewayRouteTableRoutes function in cmd/drift.go
  - Create function with signature matching design (tgwRouteTableResources, template, parameters, logicalToPhysical, output, awsConfig)
  - Iterate through each Transit Gateway route table
  - Call lib.GetTransitGatewayRouteTableRoutes with context
  - Filter out propagated routes (Type == propagated)
  - Filter out routes in transient states (not active or blackhole)
  - Call lib.FilterTGWRoutesByLogicalId to get template routes
  - Compare AWS routes vs template routes
  - Detect unmanaged routes (in AWS, not in template) - format with outputsettings.StringPositiveInline
  - Detect removed routes (in template, not in AWS) - format with outputsettings.StringWarningInline
  - Detect modified routes (different attachment/state) - standard format
  - Skip routes with empty destination (condition handling)
  - Build drift output entries matching VPC route pattern
  - Support --separate-properties flag
  - Add to output array
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7), [5.9](requirements.md#5.9), [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [6.4](requirements.md#6.4), [6.5](requirements.md#6.5), [6.6](requirements.md#6.6), [6.7](requirements.md#6.7), [6.8](requirements.md#6.8), [6.9](requirements.md#6.9), [6.10](requirements.md#6.10), [8.3](requirements.md#8.3)
  - References: cmd/drift.go

- [ ] 18. Integrate checkTransitGatewayRouteTableRoutes into drift detection flow
  - Update detectDrift function in cmd/drift.go
  - Add call to checkTransitGatewayRouteTableRoutes after checkRouteTableRoutes
  - Pass tgwRouteTableResources from separateSpecialCases
  - Pass all required parameters (template, parameters, logicalToPhysical, output, awsConfig)
  - Ensure no errors in compilation
  - Follow existing pattern from NACL and VPC route checking
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4)
  - References: cmd/drift.go

## Command Testing

- [ ] 19. Write unit tests for checkTransitGatewayRouteTableRoutes
  - Create cmd/drift_test.go tests for the function
  - Mock AWS API responses
  - Test unmanaged route detection (AWS has route not in template)
  - Test removed route detection (template has route not in AWS)
  - Test modified route detection (different attachment ID or state)
  - Test propagated routes are filtered out
  - Test transient state routes are filtered out
  - Test empty route table handling
  - Test --separate-properties flag behavior
  - Verify output format matches requirements
  - Requirements: [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6), [6.5](requirements.md#6.5), [6.6](requirements.md#6.6), [6.8](requirements.md#6.8), [10.1](requirements.md#10.1), [10.2](requirements.md#10.2)
  - References: cmd/drift_test.go, lib/drift_test.go

## Integration Testing

- [ ] 20. Write integration test for end-to-end Transit Gateway drift detection
  - Create test with //go:build integration tag
  - Create complete drift scenario with CloudFormation stack mock
  - Mock EC2 API responses for SearchTransitGatewayRoutes
  - Include multiple route tables with various drift scenarios
  - Test detection of unmanaged, removed, and modified routes
  - Verify output format and styling
  - Run with INTEGRATION=1 environment variable
  - Requirements: [11.6](requirements.md#11.6)
  - References: cmd/drift_integration_test.go

- [ ] 21. Write integration test for propagated routes handling
  - Create test with //go:build integration tag
  - Mock API response with mix of static and propagated routes
  - Verify only static routes are compared
  - Verify propagated routes don't appear in drift output
  - Verify route tables with only propagated routes show no drift
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3), [10.2](requirements.md#10.2)
  - References: cmd/drift_integration_test.go

- [ ] 22. Write integration test for prefix list handling
  - Create test with //go:build integration tag
  - Mock template with prefix list route
  - Mock API response with prefix list route
  - Verify prefix list route is compared like CIDR routes
  - Verify drift detection works for prefix list destinations
  - Requirements: [2.4](requirements.md#2.4), [4.4](requirements.md#4.4)
  - References: cmd/drift_integration_test.go

- [ ] 23. Write integration test for ECMP routes
  - Create test with //go:build integration tag
  - Mock route with multiple attachments
  - Verify only first attachment is compared
  - Document expected ECMP limitation behavior in test comments
  - Requirements: [3.6](requirements.md#3.6)
  - References: cmd/drift_integration_test.go

## Code Quality

- [ ] 24. Run go fmt on all modified files
  - Run go fmt on lib/tgw_routetables.go
  - Run go fmt on lib/tgw_routetables_test.go
  - Run go fmt on lib/interfaces.go
  - Run go fmt on cmd/drift.go
  - Run go fmt on cmd/drift_test.go (if created)
  - Verify no formatting issues
  - Requirements: [11.8](requirements.md#11.8)

- [ ] 25. Run all unit tests
  - Run go test ./lib -v
  - Run go test ./cmd -v
  - Verify all tests pass
  - Check test coverage for new code
  - Ensure no regressions in existing tests
  - Requirements: [11.5](requirements.md#11.5), [11.8](requirements.md#11.8)

- [ ] 26. Run integration tests
  - Run INTEGRATION=1 go test ./cmd -v
  - Verify all integration tests pass
  - Test Transit Gateway-specific scenarios
  - Ensure existing drift detection still works
  - Requirements: [11.6](requirements.md#11.6), [11.8](requirements.md#11.8)

- [ ] 27. Run linter validation
  - Run golangci-lint run on modified files
  - Fix any linting issues
  - Ensure code follows Go best practices
  - Verify no new warnings introduced
  - Requirements: [11.7](requirements.md#11.7)

## Final Validation

- [ ] 28. Verify backward compatibility
  - Test drift detection on stack with no Transit Gateway resources
  - Test drift detection on stack with only VPC route tables
  - Test drift detection on stack with both VPC and Transit Gateway route tables
  - Verify all existing flags work correctly
  - Verify all output formats (table, CSV, JSON) work correctly
  - Ensure no breaking changes to existing functionality
  - Requirements: [8.1](requirements.md#8.1), [8.5](requirements.md#8.5), [8.7](requirements.md#8.7), [8.8](requirements.md#8.8), [10.5](requirements.md#10.5)
