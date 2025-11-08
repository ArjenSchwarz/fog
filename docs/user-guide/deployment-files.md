# Deployment File Format

This document describes the deployment file format used by Fog, which is compatible with AWS CloudFormation's stack deployment files used in Git sync.

## Table of Contents

- [Overview](#overview)
- [File Format](#file-format)
- [Field Reference](#field-reference)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Comparison with Traditional Approach](#comparison-with-traditional-approach)

## Overview

Deployment files provide a single-file approach to defining CloudFormation stack deployments. Instead of specifying template, parameters, and tags separately via command-line flags, you can define everything in one file.

### When to Use Deployment Files

**Use deployment files when:**
- You want a single source of truth for a stack's deployment configuration
- You're using AWS CloudFormation Git sync and want compatibility
- You prefer declarative configuration over command-line flags
- You need to version control complete deployment specifications

**Use traditional approach when:**
- You want to reuse parameter/tag files across multiple templates
- You need to compose tags from multiple files
- You prefer explicit command-line control

## File Format

Deployment files support the same formats as configuration files:
- **YAML**: `deployment-name.yaml` or `deployment-name.yml` (recommended)
- **JSON**: `deployment-name.json`

### File Location

By default, Fog looks for deployment files in:
- Current directory
- `deployments/` directory
- Custom directory specified in configuration

You can specify a deployment file using the `--deployment-file` flag:

```bash
fog deploy --stackname mystack --deployment-file vpc-production
```

Fog will search for:
- `vpc-production.yaml`
- `vpc-production.yml`
- `vpc-production.json`
- `deployments/vpc-production.yaml`
- `deployments/vpc-production.yml`
- `deployments/vpc-production.json`

## Field Reference

A deployment file contains three main fields:

### `template-file-path` (Required)

**Type**: `string`

Specifies the path to the CloudFormation template file, relative to the deployment file location.

```yaml
template-file-path: "../templates/vpc.yaml"
```

**Supported template formats**:
- YAML: `.yaml`, `.yml`
- JSON: `.json`
- CloudFormation extensions: `.template`, `.templ`, `.tmpl`

**Path resolution**:
- Paths are relative to the deployment file's directory
- Use `../` to navigate up directories
- Absolute paths are supported but not recommended for portability

### `parameters` (Optional)

**Type**: `map of key-value pairs`

Defines CloudFormation stack parameters as key-value pairs.

```yaml
parameters:
  VpcCidr: "10.0.0.0/16"
  EnvironmentName: production
  EnableNatGateway: "true"
```

**Important notes**:
- All values must be strings (CloudFormation requirement)
- Boolean values must be quoted: `"true"` or `"false"`
- Numeric values should be quoted for consistency: `"3"`
- Empty values are allowed: `KeyName: ""`

### `tags` (Optional)

**Type**: `map of key-value pairs`

Defines CloudFormation stack tags as key-value pairs.

```yaml
tags:
  Environment: production
  CostCenter: engineering
  ManagedBy: fog
  Owner: infrastructure-team
```

**Important notes**:
- Tag keys and values are strings
- Maximum 50 tags per stack (AWS limit)
- Default tags from configuration file are still applied unless `--default-tags=false` is used
- Tag keys are case-sensitive

## Examples

### Basic Deployment File

Minimal deployment file with template and parameters:

```yaml
# deployments/vpc-dev.yaml
template-file-path: "../templates/vpc.yaml"

parameters:
  EnvironmentName: development
  VpcCidr: "10.0.0.0/16"
  EnableNatGateway: "false"

tags:
  Environment: development
  CostCenter: engineering
```

Usage:
```bash
fog deploy --stackname dev-vpc --deployment-file vpc-dev
```

### Production Deployment File

Production configuration with all optional features:

```yaml
# deployments/vpc-production.yaml
template-file-path: "../templates/vpc.yaml"

parameters:
  EnvironmentName: production
  VpcCidr: "10.0.0.0/16"
  EnableNatGateway: "true"
  NatGatewayCount: "3"
  EnableVpcFlowLogs: "true"
  EnableDnsHostnames: "true"
  EnableDnsSupport: "true"

tags:
  Environment: production
  CostCenter: infrastructure
  Compliance: required
  Owner: infrastructure-team
  Project: core-networking
  ManagedBy: fog
  BackupPolicy: daily
```

Usage:
```bash
fog deploy --stackname prod-vpc --deployment-file vpc-production --non-interactive
```

### Multi-Region Deployment

Deployment files for the same template across different regions:

**deployments/vpc-us-east-1.yaml**:
```yaml
template-file-path: "../templates/vpc.yaml"

parameters:
  EnvironmentName: production
  VpcCidr: "10.0.0.0/16"
  AvailabilityZones: "us-east-1a,us-east-1b,us-east-1c"

tags:
  Environment: production
  Region: us-east-1
```

**deployments/vpc-us-west-2.yaml**:
```yaml
template-file-path: "../templates/vpc.yaml"

parameters:
  EnvironmentName: production
  VpcCidr: "10.1.0.0/16"
  AvailabilityZones: "us-west-2a,us-west-2b,us-west-2c"

tags:
  Environment: production
  Region: us-west-2
```

Usage:
```bash
# Deploy to us-east-1
fog deploy --region us-east-1 --stackname prod-vpc --deployment-file vpc-us-east-1

# Deploy to us-west-2
fog deploy --region us-west-2 --stackname prod-vpc --deployment-file vpc-us-west-2
```

### Complex Application Stack

Deployment file for a complex multi-tier application:

```yaml
# deployments/app-production.yaml
template-file-path: "../templates/application-stack.yaml"

parameters:
  # Network Configuration
  VpcId: vpc-0123456789abcdef
  PrivateSubnetIds: "subnet-111,subnet-222,subnet-333"
  PublicSubnetIds: "subnet-444,subnet-555,subnet-666"

  # Application Configuration
  ApplicationName: my-application
  EnvironmentName: production
  DockerImage: "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app:v1.2.3"

  # Compute Configuration
  DesiredCount: "6"
  MinCapacity: "3"
  MaxCapacity: "12"
  InstanceType: t3.large

  # Database Configuration
  DatabaseInstanceClass: db.r5.xlarge
  DatabaseName: appdb
  DatabaseUsername: appadmin
  DatabaseMultiAZ: "true"

  # Storage Configuration
  BackupRetentionDays: "30"
  StorageEncrypted: "true"

  # Monitoring Configuration
  EnableDetailedMonitoring: "true"
  AlarmEmail: ops-team@example.com

tags:
  Environment: production
  Application: my-application
  CostCenter: product-engineering
  Owner: platform-team
  Compliance: pci-dss
  BackupPolicy: daily
  MonitoringLevel: detailed
  MaintenanceWindow: "sun:03:00-sun:04:00"
```

Usage:
```bash
fog deploy \
  --stackname prod-my-application \
  --deployment-file app-production \
  --config fog-production.yaml \
  --non-interactive
```

### Deployment File with No Parameters

Some templates don't require parameters:

```yaml
# deployments/s3-bucket.yaml
template-file-path: "../templates/s3-bucket.yaml"

tags:
  Environment: production
  Purpose: application-data
  Encryption: AES256
```

### JSON Format Example

Same deployment file in JSON format:

```json
{
  "template-file-path": "../templates/vpc.yaml",
  "parameters": {
    "EnvironmentName": "production",
    "VpcCidr": "10.0.0.0/16",
    "EnableNatGateway": "true"
  },
  "tags": {
    "Environment": "production",
    "CostCenter": "engineering",
    "ManagedBy": "fog"
  }
}
```

## Best Practices

### 1. Organize by Environment

Create separate deployment files for each environment:

```
deployments/
├── app-dev.yaml
├── app-staging.yaml
├── app-production.yaml
├── vpc-dev.yaml
├── vpc-staging.yaml
└── vpc-production.yaml
```

### 2. Use Descriptive Naming

Name deployment files to clearly indicate what they deploy:

```
deployments/
├── vpc-production-us-east-1.yaml
├── vpc-production-us-west-2.yaml
├── eks-cluster-production.yaml
├── rds-database-production.yaml
└── s3-buckets-production.yaml
```

### 3. Keep Templates Separate

Store templates separately from deployment files:

```
infrastructure/
├── templates/
│   ├── vpc.yaml
│   ├── eks-cluster.yaml
│   └── rds-database.yaml
└── deployments/
    ├── vpc-production.yaml
    ├── eks-cluster-production.yaml
    └── rds-database-production.yaml
```

### 4. Version Control Everything

- Commit deployment files to version control
- Track changes to parameters and tags over time
- Use meaningful commit messages when changing deployment files

### 5. Use Comments for Complex Parameters

Add comments to explain non-obvious parameter values:

```yaml
template-file-path: "../templates/eks-cluster.yaml"

parameters:
  ClusterVersion: "1.28"  # Update with caution - requires testing
  NodeGroupSize: "5"      # Sized for current traffic + 20% headroom
  NodeInstanceType: m5.xlarge  # Cost-optimized for our workload

tags:
  Environment: production
```

### 6. Validate Before Deploying

Use prechecks in your configuration to validate templates:

```yaml
# fog.yaml
templates:
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
  stop-on-failed-prechecks: true
```

### 7. Combine with Default Tags

Let Fog add default tags automatically:

**fog.yaml**:
```yaml
tags:
  default:
    ManagedBy: fog
    Repository: https://github.com/myorg/infrastructure
    Team: platform-engineering
```

**deployments/app.yaml**:
```yaml
template-file-path: "../templates/app.yaml"

parameters:
  AppName: my-app

tags:
  Environment: production  # Deployment-specific tags
  CostCenter: product
```

Result: Stack gets both default tags and deployment-specific tags.

### 8. Handle Secrets Carefully

Never commit secrets in deployment files. Use parameter placeholders:

```yaml
# deployments/database.yaml
template-file-path: "../templates/rds.yaml"

parameters:
  DatabaseUsername: admin
  # Don't include DatabasePassword here!
  # Pass it via AWS Secrets Manager reference in template
  # or use AWS Systems Manager Parameter Store
  DatabasePasswordParameter: /prod/database/master-password
```

## Comparison with Traditional Approach

### Deployment File Approach

```bash
# deployments/vpc-production.yaml contains template, parameters, and tags
fog deploy --stackname prod-vpc --deployment-file vpc-production
```

**Advantages**:
- Single source of truth
- Easier to version control complete deployments
- Compatible with AWS CloudFormation Git sync
- Cleaner command-line interface

**Disadvantages**:
- Cannot reuse parameter files across templates
- Cannot compose from multiple tag files
- Less flexibility for ad-hoc deployments

### Traditional Approach

```bash
fog deploy \
  --stackname prod-vpc \
  --template vpc \
  --parameters vpc-production \
  --tags globaltags/production,vpc-tags
```

**Advantages**:
- Reuse parameter files across templates
- Compose tags from multiple files
- More flexible for variations

**Disadvantages**:
- More complex command-line
- Deployment configuration spread across multiple files
- Requires understanding of file search paths

### Hybrid Approach

You can use both approaches together:

```bash
# Use deployment file but override with config default tags
fog deploy --stackname prod-vpc --deployment-file vpc-production

# Deployment file + additional runtime config
fog deploy \
  --stackname prod-vpc \
  --deployment-file vpc-production \
  --config fog-production.yaml \
  --non-interactive
```

## Limitations

1. **No parameter file merging**: You cannot use both `--parameters` flag and deployment file parameters
2. **No tag file merging**: You cannot use both `--tags` flag and deployment file tags
3. **JSON only for separate files**: If not using deployment files, parameter and tag files must be JSON
4. **No parameter references**: Cannot reference outputs from other stacks (use template Fn::ImportValue instead)

## See Also

- [Configuration Reference](configuration-reference.md) - Complete configuration options
- [Advanced Usage](advanced-usage.md) - Complex deployment scenarios
- [Examples](../examples/deployments/) - Example deployment files
- [AWS Documentation](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/git-sync-concepts-terms.html) - CloudFormation Git sync
