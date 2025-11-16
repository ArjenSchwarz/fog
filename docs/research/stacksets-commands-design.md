# StackSets Commands - Detailed Design

**Research Date:** 2025-11-16
**Purpose:** Detailed command specifications for StackSet functionality in fog

## Command Overview

All StackSet commands live under the `stackset` command group, following fog's existing pattern for `stack` and `resource` groups.

```bash
fog stackset <subcommand> [flags]
```

## Command Reference

### 1. `fog stackset list`

**Purpose:** List all StackSets in the current account/region

**Usage:**
```bash
fog stackset list [flags]
```

**Flags:**
```
--status string         Filter by status (ACTIVE, DELETED)
--max-items int        Maximum items to return (default: 100)
--output string        Output format (table, csv, json, yaml) (default: table)
--profile string       AWS profile
--region string        AWS region
```

**Output Fields:**
- StackSet Name
- Status
- Total Instances
- Drift Status
- Permission Model
- Last Updated
- Description (truncated)

**Examples:**
```bash
# List all active StackSets
fog stackset list

# List in JSON format
fog stackset list --output json

# Filter by status
fog stackset list --status ACTIVE

# Use specific profile
fog stackset list --profile production --region us-east-1
```

**Table Output:**
```
STACKSET NAME           STATUS  INSTANCES  DRIFT STATUS  PERMISSION MODEL    LAST UPDATED
baseline-security       ACTIVE  40         DRIFTED       SERVICE_MANAGED     2025-11-15 14:32:11
vpc-networking          ACTIVE  20         IN_SYNC       SELF_MANAGED        2025-11-14 09:15:43
logging-infrastructure  ACTIVE  60         NOT_CHECKED   SERVICE_MANAGED     2025-11-13 16:20:05
```

**JSON Output:**
```json
{
  "stackSets": [
    {
      "stackSetName": "baseline-security",
      "stackSetId": "baseline-security:a1b2c3d4-e5f6-7890-abcd-ef1234567890",
      "status": "ACTIVE",
      "instanceCount": 40,
      "driftStatus": "DRIFTED",
      "permissionModel": "SERVICE_MANAGED",
      "lastUpdated": "2025-11-15T14:32:11Z",
      "description": "Security baseline for all accounts"
    }
  ]
}
```

---

### 2. `fog stackset describe`

**Purpose:** Show detailed information about a specific StackSet

**Usage:**
```bash
fog stackset describe <stackset-name> [flags]
```

**Flags:**
```
--output string        Output format (table, json, yaml) (default: table)
--show-template        Include template body in output
--profile string       AWS profile
--region string        AWS region
```

**Output Sections:**
1. **StackSet Overview**: Name, ID, Status, Description
2. **Configuration**: Permission model, admin role, execution role
3. **Template**: Template summary (or full body with --show-template)
4. **Parameters**: Parameter definitions and default values
5. **Tags**: Applied tags
6. **Capabilities**: Required capabilities (IAM, NAMED_IAM, AUTO_EXPAND)
7. **Auto Deployment**: Settings (if service-managed)
8. **Managed Execution**: Status
9. **Drift**: Drift status and last check time
10. **Instances**: Total count, status breakdown
11. **Recent Operations**: Last 5 operations

**Examples:**
```bash
# Describe a StackSet
fog stackset describe baseline-security

# Show full template
fog stackset describe baseline-security --show-template

# Output as JSON
fog stackset describe baseline-security --output json
```

**Output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
StackSet: baseline-security
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Overview:
  Name:               baseline-security
  ID:                 baseline-security:a1b2c3d4-e5f6-7890-abcd-ef1234567890
  Status:             ACTIVE
  Description:        Security baseline for all accounts
  Created:            2025-11-01 10:15:30

Configuration:
  Permission Model:   SERVICE_MANAGED
  Admin Role:         <managed by AWS Organizations>
  Execution Role:     AWSCloudFormationStackSetExecutionRole
  Managed Execution:  Enabled

Template:
  Size:               4,521 bytes
  Resources:          15 resources
  Outputs:            3 outputs

Parameters (3):
  EnableGuardDuty     = true
  EnableSecurityHub   = true
  LogRetentionDays    = 90

Tags (2):
  Environment         = production
  ManagedBy           = fog

Capabilities:
  - CAPABILITY_NAMED_IAM

Auto Deployment:
  Enabled:            true
  Retain Stacks:      false

Drift Detection:
  Status:             DRIFTED
  Last Check:         2025-11-15 14:32:11
  Drifted Instances:  3 of 40

Stack Instances:
  Total:              40
  Current:            37
  Outdated:           3
  Inoperable:         0

Recent Operations:
  ID: b2c3d4e5-f6a7-8901-bcde-fa2345678901
    Action:           UPDATE
    Status:           SUCCEEDED
    Created:          2025-11-15 14:30:00
    Duration:         8m 32s

  ID: a1b2c3d4-e5f6-7890-abcd-ef1234567890
    Action:           CREATE
    Status:           SUCCEEDED
    Created:          2025-11-01 10:20:15
    Duration:         12m 45s
```

---

### 3. `fog stackset instances`

**Purpose:** List stack instances for a StackSet

**Usage:**
```bash
fog stackset instances <stackset-name> [flags]
```

**Flags:**
```
--account string       Filter by account ID
--region string        Filter by region
--status string        Filter by status (CURRENT, OUTDATED, INOPERABLE)
--drift-status string  Filter by drift status (IN_SYNC, DRIFTED, NOT_CHECKED)
--max-items int       Maximum items to return (default: 100)
--output string       Output format (table, csv, json, yaml) (default: table)
--show-overrides      Show parameter overrides for each instance
--profile string      AWS profile
```

**Output Fields:**
- Account ID
- Account Alias (if available)
- Region
- Status
- Drift Status
- Stack ID
- Last Updated
- Status Reason (if failed/inoperable)
- Parameter Overrides (with --show-overrides)

**Examples:**
```bash
# List all instances
fog stackset instances baseline-security

# Filter by account
fog stackset instances baseline-security --account 123456789012

# Filter by region
fog stackset instances baseline-security --region us-east-1

# Show only drifted instances
fog stackset instances baseline-security --drift-status DRIFTED

# Show parameter overrides
fog stackset instances baseline-security --show-overrides

# Export to CSV
fog stackset instances baseline-security --output csv --file instances.csv
```

**Table Output:**
```
ACCOUNT        ACCOUNT ALIAS  REGION      STATUS    DRIFT STATUS  LAST UPDATED         STATUS REASON
123456789012   production     us-east-1   CURRENT   IN_SYNC       2025-11-15 14:35:22
123456789012   production     us-west-2   CURRENT   IN_SYNC       2025-11-15 14:38:41
234567890123   staging        us-east-1   OUTDATED  NOT_CHECKED   2025-11-10 11:20:15
234567890123   staging        us-west-2   CURRENT   DRIFTED       2025-11-15 14:42:03
345678901234   development    us-east-1   CURRENT   IN_SYNC       2025-11-15 14:45:18
```

**With Parameter Overrides:**
```
ACCOUNT        REGION      STATUS    PARAMETER OVERRIDES
123456789012   us-east-1   CURRENT   LogRetentionDays=365
234567890123   us-east-1   CURRENT   LogRetentionDays=30, EnableGuardDuty=false
```

---

### 4. `fog stackset deploy`

**Purpose:** Create or update a StackSet and optionally deploy instances

**Usage:**
```bash
fog stackset deploy [flags]
```

**Flags:**

*StackSet Identity:*
```
--stackset-name string          StackSet name (required)
--description string            StackSet description
```

*Template and Parameters:*
```
--template string               Template file path (required for create)
--parameters string             Parameters file (JSON/YAML)
--parameter-overrides string    Parameter overrides file (per account/region)
--tags string                   Tags file (JSON/YAML)
--default-tags                  Add default tags from config
```

*Deployment Targets:*
```
--accounts strings              Target account IDs (comma-separated)
--regions strings               Target regions (comma-separated)
--organizational-units strings  Target OUs (service-managed only)
--deployment-targets string     Deployment targets file (JSON/YAML)
```

*Permission Model:*
```
--permission-model string       Permission model: SELF_MANAGED or SERVICE_MANAGED
--admin-role-arn string        Admin role ARN (self-managed)
--execution-role-name string   Execution role name (default: AWSCloudFormationStackSetExecutionRole)
```

*Deployment Preferences:*
```
--region-order strings              Region deployment order
--failure-tolerance-count int       Number of failures before stopping
--failure-tolerance-percent int     Percentage of failures before stopping
--max-concurrent-count int          Maximum concurrent operations
--max-concurrent-percent int        Percentage of concurrent operations
--region-concurrency string         Region concurrency: SEQUENTIAL or PARALLEL
--concurrency-mode string           Concurrency mode: STRICT_FAILURE_TOLERANCE or SOFT_FAILURE_TOLERANCE
```

*Operation Options:*
```
--managed-execution             Enable managed execution (default: true)
--auto-deployment               Enable auto deployment (service-managed only)
--operation-id string          Custom operation ID
```

*Fog Options:*
```
--dry-run                      Show what would be deployed without executing
--non-interactive              Skip all confirmations
--deployment-file string       Deployment file with all settings
--create-only                  Only create StackSet, don't deploy instances
--update-instances             Update instances when updating StackSet
--quiet                        Suppress progress output
--profile string               AWS profile
--region string                AWS region
```

**Examples:**

**Create new StackSet and deploy instances:**
```bash
fog stackset deploy \
  --stackset-name baseline-security \
  --template templates/security-baseline.yaml \
  --parameters parameters/security-baseline.json \
  --accounts 123456789012,234567890123,345678901234 \
  --regions us-east-1,us-west-2 \
  --permission-model SERVICE_MANAGED \
  --max-concurrent-count 2 \
  --failure-tolerance-count 1
```

**Update existing StackSet template:**
```bash
fog stackset deploy \
  --stackset-name baseline-security \
  --template templates/security-baseline-v2.yaml \
  --update-instances
```

**Create StackSet only (no instances):**
```bash
fog stackset deploy \
  --stackset-name baseline-security \
  --template templates/security-baseline.yaml \
  --parameters parameters/security-baseline.json \
  --create-only
```

**Deploy to Organizations OUs:**
```bash
fog stackset deploy \
  --stackset-name baseline-security \
  --template templates/security-baseline.yaml \
  --organizational-units ou-xxxx-yyyyyyyy,ou-xxxx-zzzzzzzz \
  --regions us-east-1,us-west-2,eu-west-1 \
  --permission-model SERVICE_MANAGED \
  --auto-deployment
```

**Use deployment file:**
```bash
fog stackset deploy --deployment-file deployments/baseline-security.yaml
```

**Deployment File Format:**
```yaml
# deployments/baseline-security.yaml
stackSetName: baseline-security
description: Security baseline for all accounts
permissionModel: SERVICE_MANAGED

template: templates/security-baseline.yaml
parameters:
  - ParameterKey: EnableGuardDuty
    ParameterValue: "true"
  - ParameterKey: EnableSecurityHub
    ParameterValue: "true"
  - ParameterKey: LogRetentionDays
    ParameterValue: "90"

tags:
  - Key: Environment
    Value: production
  - Key: ManagedBy
    Value: fog

deploymentTargets:
  accounts:
    - "123456789012"
    - "234567890123"
    - "345678901234"
  regions:
    - us-east-1
    - us-west-2
    - eu-west-1

operationPreferences:
  regionOrder:
    - us-east-1
    - us-west-2
    - eu-west-1
  maxConcurrentCount: 2
  failureToleranceCount: 1
  regionConcurrency: SEQUENTIAL
  concurrencyMode: STRICT_FAILURE_TOLERANCE

managedExecution: true
```

**Parameter Overrides File:**
```yaml
# parameter-overrides.yaml
overrides:
  # Override for specific account
  - account: "234567890123"
    parameters:
      - ParameterKey: LogRetentionDays
        ParameterValue: "30"
      - ParameterKey: EnableGuardDuty
        ParameterValue: "false"

  # Override for specific account and region
  - account: "345678901234"
    region: us-west-2
    parameters:
      - ParameterKey: LogRetentionDays
        ParameterValue: "365"
```

**Interactive Workflow:**

See architecture document for detailed interactive deployment flow with progress tracking.

---

### 5. `fog stackset update-instances`

**Purpose:** Update specific stack instances (useful for parameter changes)

**Usage:**
```bash
fog stackset update-instances <stackset-name> [flags]
```

**Flags:**
```
--accounts strings              Target account IDs (comma-separated)
--regions strings               Target regions (comma-separated)
--parameter-overrides string    Parameter overrides file
--operation-preferences string  Operation preferences file
--non-interactive               Skip confirmations
--profile string                AWS profile
```

**Examples:**
```bash
# Update instances in specific accounts
fog stackset update-instances baseline-security \
  --accounts 123456789012,234567890123 \
  --regions us-east-1

# Update with parameter overrides
fog stackset update-instances baseline-security \
  --accounts 234567890123 \
  --regions us-east-1,us-west-2 \
  --parameter-overrides overrides.yaml
```

---

### 6. `fog stackset delete-instances`

**Purpose:** Delete stack instances from specific accounts/regions

**Usage:**
```bash
fog stackset delete-instances <stackset-name> [flags]
```

**Flags:**
```
--accounts strings              Target account IDs (required, comma-separated)
--regions strings               Target regions (required, comma-separated)
--retain-stacks                 Retain stacks (don't delete CloudFormation stacks)
--operation-preferences string  Operation preferences file
--non-interactive               Skip confirmations
--profile string                AWS profile
```

**Examples:**
```bash
# Delete instances from specific accounts/regions
fog stackset delete-instances baseline-security \
  --accounts 123456789012 \
  --regions us-west-2

# Delete but retain the actual stacks
fog stackset delete-instances baseline-security \
  --accounts 234567890123 \
  --regions us-east-1,us-west-2 \
  --retain-stacks
```

**Interactive Confirmation:**
```
⚠️  WARNING: This will delete stack instances

StackSet: baseline-security
Accounts: 123456789012
Regions: us-west-2
Total instances to delete: 1
Retain stacks: No

The following resources will be deleted:
  - Account: 123456789012, Region: us-west-2
    Resources: 15 CloudFormation resources

Are you sure you want to delete these instances? (yes/No):
```

---

### 7. `fog stackset delete`

**Purpose:** Delete a StackSet (all instances must be deleted first)

**Usage:**
```bash
fog stackset delete <stackset-name> [flags]
```

**Flags:**
```
--non-interactive    Skip confirmations
--profile string     AWS profile
--region string      AWS region
```

**Examples:**
```bash
# Delete StackSet
fog stackset delete baseline-security

# Non-interactive
fog stackset delete baseline-security --non-interactive
```

**Validation:**
- Checks that all stack instances have been deleted
- Shows error if instances still exist
- Provides helpful command to delete instances

**Error Output:**
```
❌ Cannot delete StackSet: 40 instances still exist

Delete all instances first:
  fog stackset delete-instances baseline-security --accounts <accounts> --regions <regions>

Or view instances:
  fog stackset instances baseline-security
```

---

### 8. `fog stackset operations`

**Purpose:** List operations for a StackSet

**Usage:**
```bash
fog stackset operations <stackset-name> [flags]
```

**Flags:**
```
--max-items int       Maximum items to return (default: 20)
--output string       Output format (table, csv, json, yaml) (default: table)
--profile string      AWS profile
--region string       AWS region
```

**Output Fields:**
- Operation ID
- Action (CREATE, UPDATE, DELETE)
- Status
- Created Time
- End Time
- Duration
- Succeeded Count
- Failed Count
- In Progress Count

**Examples:**
```bash
# List operations
fog stackset operations baseline-security

# Show more results
fog stackset operations baseline-security --max-items 50

# Export to JSON
fog stackset operations baseline-security --output json
```

**Table Output:**
```
OPERATION ID                          ACTION  STATUS     CREATED              DURATION  SUCCESS  FAILED  IN PROGRESS
b2c3d4e5-f6a7-8901-bcde-fa2345678901 UPDATE  SUCCEEDED  2025-11-15 14:30:00  8m 32s    40       0       0
a1b2c3d4-e5f6-7890-abcd-ef1234567890 CREATE  SUCCEEDED  2025-11-01 10:20:15  12m 45s   40       0       0
c3d4e5f6-a7b8-9012-cdef-ab3456789012 UPDATE  FAILED     2025-10-28 09:15:30  15m 22s   35       5       0
```

---

### 9. `fog stackset operation`

**Purpose:** Describe a specific operation with per-instance results

**Usage:**
```bash
fog stackset operation <stackset-name> <operation-id> [flags]
```

**Flags:**
```
--show-results       Show per-instance results
--output string      Output format (table, json, yaml) (default: table)
--profile string     AWS profile
--region string      AWS region
```

**Examples:**
```bash
# Describe operation
fog stackset operation baseline-security b2c3d4e5-f6a7-8901-bcde-fa2345678901

# Show per-instance results
fog stackset operation baseline-security b2c3d4e5-f6a7-8901-bcde-fa2345678901 --show-results
```

**Output:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Operation: b2c3d4e5-f6a7-8901-bcde-fa2345678901
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Overview:
  StackSet:           baseline-security
  Action:             UPDATE
  Status:             SUCCEEDED
  Created:            2025-11-15 14:30:00
  Completed:          2025-11-15 14:38:32
  Duration:           8m 32s

Operation Preferences:
  Region Order:       us-east-1, us-west-2
  Max Concurrent:     2
  Failure Tolerance:  1
  Region Concurrency: SEQUENTIAL

Results:
  Total Instances:    40
  Succeeded:          40
  Failed:             0
  In Progress:        0

Instance Results:
ACCOUNT        REGION      STATUS     UPDATED
123456789012   us-east-1   SUCCEEDED  2025-11-15 14:32:15
123456789012   us-west-2   SUCCEEDED  2025-11-15 14:35:22
234567890123   us-east-1   SUCCEEDED  2025-11-15 14:33:41
234567890123   us-west-2   SUCCEEDED  2025-11-15 14:36:18
...
```

---

### 10. `fog stackset drift`

**Purpose:** Detect drift across all stack instances

**Usage:**
```bash
fog stackset drift <stackset-name> [flags]
```

**Flags:**
```
--results-only              Don't trigger detection, show existing results
--operation-preferences string  Operation preferences file
--wait                      Wait for drift detection to complete
--timeout duration          Timeout for waiting (default: 30m)
--output string             Output format (table, csv, json, yaml) (default: table)
--profile string            AWS profile
--region string             AWS region
```

**Examples:**
```bash
# Detect drift and wait for completion
fog stackset drift baseline-security --wait

# Show existing drift results
fog stackset drift baseline-security --results-only

# Detect drift with custom operation preferences
fog stackset drift baseline-security --operation-preferences drift-prefs.yaml
```

**Output:**
```
🔍 Starting drift detection for StackSet: baseline-security

Operation ID: d4e5f6a7-b8c9-0123-defg-bc4567890123

Progress: [████████████████████] 100% (40/40 instances)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Drift Detection Results
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Overall Status: DRIFTED

Summary:
  Total Instances:    40
  In Sync:            37
  Drifted:            3
  Not Checked:        0

Drifted Instances:
ACCOUNT        REGION      DRIFTED RESOURCES  LAST CHECKED
234567890123   us-east-1   2                  2025-11-15 15:05:22
345678901234   us-east-1   1                  2025-11-15 15:06:15
345678901234   us-west-2   3                  2025-11-15 15:07:41

View instance details:
  fog stack drift <stack-name> --account 234567890123 --region us-east-1
```

---

## Global Flags

All commands support fog's global flags:

```
--config string        Config file (default: fog.yaml)
--verbose, -v          Verbose output
--output string        Output format (table, csv, json, yaml, etc.)
--file string          Save output to file
--file-format string   File output format
--profile string       AWS profile
--region string        AWS region
--timezone string      Timezone for timestamps
--debug                Debug mode
```

## Command Aliases

Consider creating convenient aliases at the root level:

```bash
fog stacksets         -> fog stackset list
fog ss                -> fog stackset
```

## Exit Codes

- `0` - Success
- `1` - General error
- `2` - Validation error (missing required flags, invalid values)
- `3` - AWS API error
- `4` - Partial failure (some instances failed)
- `5` - User cancelled operation

## Error Handling

### Permission Errors

Provide actionable error messages:

```
❌ Permission denied when creating StackSet

For SELF_MANAGED permission model, ensure:
  1. AWSCloudFormationStackSetAdministrationRole exists in account 123456789012
  2. AWSCloudFormationStackSetExecutionRole exists in target accounts
  3. Trust relationship allows assumption from admin account

For SERVICE_MANAGED permission model, ensure:
  1. AWS Organizations integration is enabled
  2. Account has permissions to access Organizations
  3. Target OUs exist and are accessible

See: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/stacksets-prereqs.html
```

### Validation Errors

```
❌ Invalid deployment configuration

Issues found:
  - Accounts: at least one account ID required
  - Regions: at least one region required
  - Template: file not found: templates/missing.yaml
  - Max Concurrent Count: must be greater than 0
```

### Partial Failures

```
⚠️  Operation completed with failures

Summary:
  Total Instances:    40
  Succeeded:          35
  Failed:             5

Failed Instances:
  Account: 234567890123, Region: us-east-1
    Reason: Parameter validation failed: LogRetentionDays must be between 1 and 3653

  Account: 345678901234, Region: eu-west-1
    Reason: CREATE_FAILED - S3 bucket already exists

View full operation details:
  fog stackset operation baseline-security b2c3d4e5-f6a7-8901-bcde-fa2345678901
```

## Best Practices

### Command Composition

Commands are designed to work together in workflows:

```bash
# Create StackSet and deploy
fog stackset deploy --stackset-name my-stackset --template template.yaml --create-only
fog stackset instances my-stackset  # Verify it was created
fog stackset deploy --stackset-name my-stackset --accounts 123456789012 --regions us-east-1

# Update workflow
fog stackset deploy --stackset-name my-stackset --template template-v2.yaml
fog stackset instances my-stackset --status OUTDATED  # Find outdated instances
fog stackset update-instances my-stackset --accounts ... --regions ...

# Drift detection workflow
fog stackset drift my-stackset --wait
fog stackset instances my-stackset --drift-status DRIFTED  # Find drifted instances
fog stack drift <specific-stack-id>  # Investigate specific instance

# Cleanup workflow
fog stackset instances my-stackset  # Review what will be deleted
fog stackset delete-instances my-stackset --accounts ... --regions ...
fog stackset delete my-stackset
```

### Configuration File Usage

Encourage deployment files for repeatable deployments:

```yaml
# deployment/production-baseline.yaml
stackSetName: production-baseline
template: templates/baseline.yaml
parameters: parameters/production.json
deploymentTargets:
  accounts: accounts/production.txt
  regions: [us-east-1, us-west-2, eu-west-1]
operationPreferences:
  maxConcurrentCount: 3
  failureToleranceCount: 1
```

Then:
```bash
fog stackset deploy --deployment-file deployment/production-baseline.yaml
```

## Future Enhancements

### Potential Future Commands

- `fog stackset wait` - Wait for operation to complete
- `fog stackset stop` - Stop in-progress operation
- `fog stackset import` - Import existing stacks into StackSet
- `fog stackset validate` - Validate template before deployment
- `fog stackset diff` - Show differences between StackSet template and current
- `fog stackset export` - Export StackSet configuration
- `fog stackset clone` - Clone StackSet to new name

### Enhanced Features

- Interactive instance selection (TUI)
- Batch operations across multiple StackSets
- StackSet dependencies visualization
- Automatic parameter override generation based on account tags
- Integration with fog reports for StackSet deployment reports
- Webhook notifications for operation completion

## Conclusion

These commands provide comprehensive StackSet management capabilities while maintaining consistency with fog's existing command patterns and UX principles. The design emphasizes:

- **Clarity**: Clear, descriptive command names and flags
- **Safety**: Confirmations for destructive operations, dry-run mode
- **Flexibility**: Multiple output formats, filtering options
- **Composability**: Commands work together in workflows
- **Discoverability**: Consistent with existing fog commands
