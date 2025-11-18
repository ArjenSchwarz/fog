# AWS CloudFormation StackSets - Overview and Capabilities

**Research Date:** 2025-11-16
**Purpose:** Evaluate AWS StackSets for integration into the fog CLI tool

## Executive Summary

AWS CloudFormation StackSets extends CloudFormation's infrastructure-as-code capabilities to enable deployment and management of stacks across multiple AWS accounts and regions with a single operation. This capability addresses fog's current limitation of single-account, single-region operations per command execution.

## What are StackSets?

### Core Concept

A **StackSet** is a CloudFormation construct that allows you to create, update, or delete stacks across multiple AWS accounts and regions using a single template and set of parameters. A StackSet manages stack instances - individual stacks deployed to specific account/region combinations.

**Key Components:**

1. **StackSet** - The container that holds the template and configuration
2. **Stack Instance** - An individual stack deployed to a specific account and region
3. **Operation** - An asynchronous action (create, update, delete) performed on stack instances
4. **Administrator Account** - The account where the StackSet is created and managed
5. **Target Accounts** - Accounts where stack instances are deployed

### Permission Models

**Self-Managed Permissions:**
- Requires manual IAM role setup in each target account
- Administrator account assumes `AWSCloudFormationStackSetExecutionRole` in target accounts
- More control but higher setup overhead
- Roles: `AWSCloudFormationStackSetAdministrationRole` (admin account), `AWSCloudFormationStackSetExecutionRole` (target accounts)

**Service-Managed Permissions (with AWS Organizations):**
- Automatic permission management through AWS Organizations
- Enables automatic deployment to OUs (Organizational Units)
- Supports automatic deployment to new accounts joining OUs
- Simplified permission model
- Requires Organizations integration

## Key StackSet Operations

### StackSet Management

| Operation | Purpose | AWS SDK Method |
|-----------|---------|----------------|
| Create | Create new StackSet | `CreateStackSet` |
| List | Enumerate all StackSets | `ListStackSets` |
| Describe | Get StackSet details | `DescribeStackSet` |
| Update | Modify StackSet template/parameters | `UpdateStackSet` |
| Delete | Remove StackSet (after deleting instances) | `DeleteStackSet` |
| Import | Import existing stacks into StackSet | `ImportStacksToStackSet` |

### Stack Instance Management

| Operation | Purpose | AWS SDK Method |
|-----------|---------|----------------|
| Create Instances | Deploy stacks to accounts/regions | `CreateStackInstances` |
| List Instances | Enumerate stack instances | `ListStackInstances` |
| Describe Instance | Get instance details | `DescribeStackInstance` |
| Update Instances | Modify instance parameters | `UpdateStackInstances` |
| Delete Instances | Remove instances from accounts/regions | `DeleteStackInstances` |

### Operations Tracking

| Operation | Purpose | AWS SDK Method |
|-----------|---------|----------------|
| List Operations | View StackSet operation history | `ListStackSetOperations` |
| Describe Operation | Get operation details | `DescribeStackSetOperation` |
| List Operation Results | View per-instance operation results | `ListStackSetOperationResults` |
| Stop Operation | Cancel in-progress operation | `StopStackSetOperation` |

### Drift Detection

| Operation | Purpose | AWS SDK Method |
|-----------|---------|----------------|
| Detect Drift | Scan all instances for drift | `DetectStackSetDrift` |
| List Drift Results | View drift detection results | Uses standard drift operations |

## StackSet Properties

### Template and Parameters

- **Template**: CloudFormation template (JSON/YAML) up to 51,200 bytes (or S3 URL for larger)
- **Parameters**: Global parameters applied to all instances
- **Parameter Overrides**: Per-account or per-region parameter overrides
- **Tags**: Applied to StackSet and propagated to instances
- **Capabilities**: IAM capabilities (CAPABILITY_IAM, CAPABILITY_NAMED_IAM, CAPABILITY_AUTO_EXPAND)

### Deployment Configuration

**Operation Preferences:**
```yaml
OperationPreferences:
  RegionOrder:              # Order to deploy regions
    - us-east-1
    - us-west-2
  FailureToleranceCount: 1  # OR FailureTolerancePercentage: 20
  MaxConcurrentCount: 2     # OR MaxConcurrentPercentage: 50
  RegionConcurrencyType:    # SEQUENTIAL or PARALLEL
  ConcurrencyMode:          # STRICT_FAILURE_TOLERANCE or SOFT_FAILURE_TOLERANCE
```

**Deployment Options:**
- **AutoDeployment** (Service-managed only): Automatically deploy to new accounts in OUs
- **ManagedExecution**: CloudFormation optimizes operation handling (queues conflicts, runs non-conflicts concurrently)

### Status Values

**StackSet Status:**
- `ACTIVE` - StackSet is operational
- `DELETED` - StackSet has been deleted

**Stack Instance Status:**
- `CURRENT` - Instance matches StackSet template
- `OUTDATED` - StackSet updated but instance not yet updated
- `INOPERABLE` - Instance failed to create/update

**Operation Status:**
- `RUNNING` - Operation in progress
- `SUCCEEDED` - Completed successfully
- `FAILED` - Failed (some instances may have succeeded)
- `STOPPING` - Being stopped
- `STOPPED` - Stopped before completion
- `QUEUED` - Waiting for other operations (with ManagedExecution)

## Best Practices

### Deployment Strategy

1. **Start Small**: Test with 1 account, 1 region, verify success
2. **Progressive Rollout**: Gradually increase concurrent accounts and regions
3. **Conservative Tolerances**: Start with low failure tolerance (0-1), increase as confidence grows
4. **Region Ordering**: Deploy to lowest-impact regions first
5. **Concurrency Control**: Balance speed vs risk with MaxConcurrent and FailureTolerance settings

### Operation Management

1. **Monitor Operations**: Track operation status and results
2. **Handle Failures**: Review failed instances, fix issues, retry
3. **Use Managed Execution**: Enable for better operation queuing and conflict resolution
4. **Selective Updates**: Update specific instances rather than all when testing changes
5. **Drift Detection**: Regularly check for configuration drift across instances

### Testing and Validation

1. **Dry-Run Equivalent**: Test in single account/region before full deployment
2. **Parameter Validation**: Validate parameters work across all target environments
3. **Template Testing**: Ensure template works in all target regions (service availability varies)
4. **Permission Verification**: Verify IAM roles and permissions before deployment

### Organizations Integration

1. **Use Service-Managed Permissions**: Simplifies permission management
2. **Organizational Unit Strategy**: Structure OUs to match deployment patterns
3. **Auto-Deployment**: Enable for automatic deployment to new accounts
4. **Account Filtering**: Use filters to target specific accounts within OUs

## Use Cases

### Multi-Account Governance

- Deploy security baseline (GuardDuty, SecurityHub, Config)
- Enforce tagging policies
- Deploy logging and monitoring infrastructure
- Create cross-account IAM roles

### Multi-Region Resilience

- Deploy disaster recovery infrastructure
- Create multi-region applications
- Deploy global network infrastructure
- Configure cross-region backups

### Organizational Standards

- Deploy standard VPC configurations
- Create common security groups
- Deploy shared services (transit gateway attachments)
- Standardize resource configurations

## Limitations and Considerations

### Operational Limits

- StackSet operations are asynchronous and can be long-running
- Updating 40 instances (20 accounts × 2 regions) updates all by default
- No built-in rollback for failed multi-instance deployments
- Maximum 2000 stack instances per StackSet in service-managed mode
- Maximum 500 stack instances per StackSet in self-managed mode

### Regional Constraints

- Not all AWS services available in all regions
- Service limits vary by region
- Template must be region-agnostic or handle region-specific resources
- Some resources require different configurations per region

### Permission Complexity

- Self-managed requires IAM role setup in every target account
- Service-managed requires Organizations integration
- Execution role must have permissions for all resources in template
- Cross-account trust relationships require careful configuration

### Operational Overhead

- More complex error handling (partial successes/failures)
- Requires monitoring across multiple accounts/regions
- Debugging failures requires per-instance investigation
- Parameter management becomes more complex with overrides

## AWS SDK v2 Go Support

The `github.com/aws/aws-sdk-go-v2/service/cloudformation` package provides full StackSet support:

- All StackSet operations available
- Paginator support for list operations (`ListStackSetsPaginator`, `ListStackInstancesPaginator`)
- Standard AWS SDK v2 patterns (context, options, errors)
- Type-safe input/output structures
- Full integration with AWS SDK v2 credential chain

## Comparison: Traditional Stacks vs StackSets

| Aspect | Traditional Stacks | StackSets |
|--------|-------------------|-----------|
| Scope | Single account, single region | Multiple accounts, multiple regions |
| Deployment | Synchronous per stack | Asynchronous across instances |
| Management | Individual stack operations | Centralized StackSet management |
| Consistency | Manual cross-account/region | Automatic via single template |
| Complexity | Lower | Higher (operations, permissions) |
| Use Case | Single environment | Multi-account/multi-region |
| Permissions | Stack-level IAM | Cross-account IAM roles or Organizations |
| Drift Detection | Per stack | Across all instances |
| Updates | Immediate per stack | Batched with concurrency controls |

## Integration Opportunities for Fog

### Current Gaps

- Fog operates on single account/region per command
- No multi-account orchestration capabilities
- No centralized view of multi-account deployments
- Manual repetition required for multi-region deployments

### StackSets Benefits

- **Multi-Account Overview**: Single command to view all deployments across accounts
- **Simplified Deployment**: One command to deploy/update across environments
- **Consistency**: Ensure identical configurations across accounts/regions
- **Operational Efficiency**: Reduce repetitive commands and manual tracking
- **Enhanced Visibility**: Track deployment operations and status centrally

### Potential Features

1. **List and Status**
   - View all StackSets and their instances
   - Show deployment status across accounts/regions
   - Display operation history and current operations
   - Detect drift across all instances

2. **Deploy and Update**
   - Create/update StackSets with fog's familiar UX
   - Deploy instances to target accounts/regions
   - Support deployment preferences (concurrency, failure tolerance)
   - Dry-run capabilities before deployment
   - Interactive approval workflows

3. **Operations Management**
   - Monitor operation progress
   - View per-instance operation results
   - Handle failures and retries
   - Stop in-progress operations

4. **Integration Points**
   - Leverage existing fog patterns (flag groups, output formats)
   - Reuse AWS configuration and credential handling
   - Apply fog's output formatting (table, CSV, JSON, etc.)
   - Maintain fog's interactive/non-interactive modes

## Conclusion

AWS CloudFormation StackSets provides powerful multi-account and multi-region deployment capabilities that would significantly enhance fog's utility for organizations managing infrastructure across multiple AWS environments. The AWS SDK v2 Go support is comprehensive, and StackSets align well with fog's existing architectural patterns.

**Key Takeaway**: StackSets transforms CloudFormation from a single-environment tool to an enterprise-scale multi-account orchestration platform, making it an ideal addition to fog's capabilities.

## References

- [AWS CloudFormation StackSets User Guide](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/what-is-cfnstacksets.html)
- [StackSets Best Practices](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/stacksets-bestpractices.html)
- [AWS SDK Go v2 CloudFormation Package](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/cloudformation)
- [StackSets API Reference](https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/Welcome.html)
