# Fog User Guide

Welcome to the comprehensive user guide for Fog, a tool for managing CloudFormation deployments.

## Quick Start

- [README](../../README.md) - Project overview and basic usage
- [Installation](#installation) - How to install Fog
- [Your First Deployment](#your-first-deployment) - Get started quickly

## Core Documentation

### Configuration
- **[Configuration Reference](configuration-reference.md)** - Complete guide to all configuration options
  - File locations and formats
  - All configuration parameters explained
  - Environment-specific examples
  - CI/CD configuration patterns

### Deployments
- **[Deployment Files](deployment-files.md)** - Deployment file format specification
  - File format and structure
  - Field reference
  - Examples for various scenarios
  - Best practices

### Advanced Features
- **[Advanced Usage](advanced-usage.md)** - Complex deployment scenarios
  - Multi-stack deployments
  - Cross-stack references
  - Multi-region deployments
  - CI/CD integration
  - Advanced drift detection
  - Environment management

### Help & Support
- **[Troubleshooting Guide](troubleshooting.md)** - Solutions to common problems
  - Deployment issues
  - Configuration problems
  - AWS credentials and permissions
  - Template and parameter issues
  - Drift detection troubleshooting
  - Debug mode

## Installation

### Download Pre-built Binary

Download the latest release for your platform:

- **Linux (amd64)**: `wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-linux-amd64`
- **macOS (amd64)**: `wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-darwin-amd64`
- **macOS (arm64)**: `wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-darwin-arm64`
- **Windows (amd64)**: Download `fog-windows-amd64.exe`

Make it executable (Linux/macOS):
```bash
chmod +x fog-*
sudo mv fog-* /usr/local/bin/fog
```

### Build from Source

Requirements:
- Go 1.21 or later

```bash
git clone https://github.com/ArjenSchwarz/fog.git
cd fog
go build
```

### Verify Installation

```bash
fog --version
```

## Your First Deployment

### 1. Set Up Directory Structure

Create a basic project structure:

```bash
mkdir -p my-infrastructure/{templates,parameters,tags}
cd my-infrastructure
```

### 2. Create a Simple Template

Create `templates/s3-bucket.yaml`:

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Description: Simple S3 bucket

Parameters:
  BucketName:
    Type: String
    Description: Name for the S3 bucket

Resources:
  Bucket:
    Type: AWS::S3::Bucket
    Properties:
      BucketName: !Ref BucketName
      VersioningConfiguration:
        Status: Enabled
      PublicAccessBlockConfiguration:
        BlockPublicAcls: true
        BlockPublicPolicy: true
        IgnorePublicAcls: true
        RestrictPublicBuckets: true

Outputs:
  BucketName:
    Description: Name of the S3 bucket
    Value: !Ref Bucket
  BucketArn:
    Description: ARN of the S3 bucket
    Value: !GetAtt Bucket.Arn
```

### 3. Create Parameters

Create `parameters/my-bucket.json`:

```json
{
  "BucketName": "my-unique-bucket-name-12345"
}
```

### 4. Create Tags

Create `tags/common.json`:

```json
{
  "Environment": "development",
  "ManagedBy": "fog",
  "Project": "my-project"
}
```

### 5. Create Configuration (Optional)

Create `fog.yaml`:

```yaml
output: table
region: us-east-1

templates:
  directory: templates

parameters:
  directory: parameters

tags:
  directory: tags
  default:
    ManagedBy: fog
```

### 6. Deploy

```bash
fog deploy \
  --stackname my-first-stack \
  --template s3-bucket \
  --parameters my-bucket \
  --tags common
```

Fog will:
1. Find your template, parameters, and tags
2. Create a changeset
3. Show you what will be created
4. Ask for confirmation
5. Deploy the stack
6. Show real-time progress
7. Display outputs when complete

### 7. Verify Deployment

Check your stack:

```bash
# View stack resources
fog resources --stackname my-first-stack

# View deployment report
fog report --stackname my-first-stack --latest
```

## Common Workflows

### Development Workflow

```bash
# 1. Create/edit template
vim templates/my-service.yaml

# 2. Validate template
cfn-lint templates/my-service.yaml

# 3. Create changeset to preview changes
fog deploy --stackname dev-my-service --template my-service --create-changeset

# 4. Review changeset in console, then deploy
fog deploy --stackname dev-my-service --template my-service --deploy-changeset
```

### Production Workflow

```bash
# 1. Use deployment file for consistency
cat > deployments/prod-my-service.yaml <<EOF
template-file-path: "../templates/my-service.yaml"
parameters:
  Environment: production
  InstanceType: t3.large
tags:
  Environment: production
  Compliance: required
EOF

# 2. Non-interactive deployment (for CI/CD)
fog deploy \
  --stackname prod-my-service \
  --deployment-file prod-my-service \
  --config fog-production.yaml \
  --non-interactive

# 3. Generate deployment report
fog report --stackname prod-my-service --output markdown --file reports/deployment.md
```

### Drift Detection Workflow

```bash
# 1. Run drift detection
fog drift --stackname my-stack

# 2. Review drift details with separate properties
fog drift --stackname my-stack --separate-properties

# 3. Generate drift report
fog drift --stackname my-stack --output markdown --file drift-report.md

# 4. Check for unmanaged resources (if configured)
fog drift --stackname my-stack --verbose
```

## Feature Overview

### Deployments

Fog simplifies CloudFormation deployments by:
- Combining create and update operations into one command
- Automatically creating changesets for review
- Providing real-time deployment progress
- Showing detailed error information on failure
- Offering to clean up failed stacks

**Key features**:
- Interactive and non-interactive modes
- Dry-run mode for testing
- Template prechecks (cfn-lint, cfn-guard, etc.)
- Automatic S3 upload for large templates
- Support for deployment files

See: [Deployment Files](deployment-files.md)

### Reports

Generate detailed deployment reports showing:
- Stack events grouped by action
- Timeline visualizations (Mermaid diagrams)
- Success/failure indicators
- Duration information

**Output formats**: Table, Markdown, HTML, JSON

```bash
fog report --stackname my-stack --output markdown --file report.md
```

### Exports

View and manage CloudFormation exports:
- List all exports across stacks
- Filter by stack name or export name (with wildcards)
- Show which stacks import each export
- Identify blocking dependencies before deletion

```bash
# View all exports
fog exports

# Filter by stack
fog exports --stackname "production-*"

# Show import relationships
fog exports --verbose
```

### Resources

List all resources managed by CloudFormation:
- View resources across all stacks
- Filter by specific stack
- See resource types, IDs, and status

```bash
# All resources
fog resources --output json

# Specific stack
fog resources --stackname my-stack --verbose
```

### Dependencies

Visualize stack dependencies:
- Show export/import relationships
- Identify dependency chains
- Generate visual graphs

```bash
# View dependencies
fog dependencies --stackname my-stack

# Generate graph
fog dependencies --output dot | dot -T png -o dependencies.png
```

### Drift Detection

Enhanced drift detection beyond AWS native capabilities:
- Tag order normalization
- VPC route table monitoring
- Transit Gateway route table monitoring
- NACL rule monitoring
- Unmanaged resource detection
- Configurable ignore lists

**Supported unmanaged resource types**:
- AWS::SSO::PermissionSet
- AWS::SSO::Assignment

See: [Advanced Usage - Drift Detection](advanced-usage.md#advanced-drift-detection)

## Configuration Management

### Configuration Precedence

Values are resolved in order (highest to lowest):

1. **Command-line flags**: `--output json`
2. **Environment variables**: `AWS_PROFILE`, `AWS_REGION`
3. **Configuration file**: `fog.yaml`
4. **Default values**: Built-in defaults

### Multiple Environments

Manage multiple environments with separate configs:

```bash
# Development
fog deploy --config fog-dev.yaml --stackname dev-vpc --template vpc

# Staging
fog deploy --config fog-staging.yaml --stackname staging-vpc --template vpc

# Production
fog deploy --config fog-prod.yaml --stackname prod-vpc --template vpc
```

### Default Tags

Apply tags automatically to all deployments:

```yaml
# fog.yaml
tags:
  default:
    ManagedBy: fog
    Organization: MyCompany
    CostCenter: engineering
    Source: https://github.com/myorg/infrastructure/$TEMPLATEPATH
```

## Best Practices

### 1. Version Control Everything

```bash
git add templates/ parameters/ tags/ deployments/ fog.yaml
git commit -m "Add infrastructure configuration"
```

### 2. Use Deployment Files for Production

```yaml
# deployments/prod-vpc.yaml
template-file-path: "../templates/vpc.yaml"
parameters:
  Environment: production
  VpcCidr: "10.0.0.0/16"
tags:
  Environment: production
  Compliance: required
```

### 3. Enable Template Validation

```yaml
# fog.yaml
templates:
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
    - cfn-guard validate -d $TEMPLATEPATH --rules production-rules
  stop-on-failed-prechecks: true
```

### 4. Use Non-Interactive Mode in CI/CD

```bash
fog deploy \
  --stackname prod-app \
  --deployment-file app-production \
  --non-interactive
```

### 5. Generate Reports for Audit Trail

```bash
fog report \
  --stackname prod-app \
  --output markdown \
  --file "reports/deployment-$(date +%Y%m%d-%H%M%S).md"
```

### 6. Regular Drift Detection

```bash
# Run weekly drift checks
fog drift --stackname prod-vpc --output json --file drift-results.json
```

### 7. Document Stack Dependencies

```bash
# Generate dependency graph
fog dependencies --output dot | dot -T png -o architecture/stack-dependencies.png
```

## Output Formats

Fog supports multiple output formats for different use cases:

| Format   | Use Case                          | Commands                     |
|----------|-----------------------------------|------------------------------|
| table    | Interactive terminal viewing      | All commands                 |
| json     | Programmatic processing, CI/CD    | All commands                 |
| csv      | Spreadsheet import, data analysis | exports, resources           |
| markdown | Documentation, reports            | report                       |
| html     | Web viewing, sharing              | report                       |
| dot      | Visual graphs                     | dependencies                 |
| yaml     | Configuration export              | Selected commands            |

### Examples

```bash
# JSON for jq processing
fog exports --output json | jq '.[] | select(.Imported == true)'

# CSV for Excel
fog resources --output csv --file resources.csv

# Markdown for documentation
fog report --stackname my-stack --output markdown --file docs/deployment.md

# Dot for visualization
fog dependencies --output dot | dot -T png -o deps.png
```

## Getting Help

### Built-in Help

```bash
# General help
fog --help

# Command-specific help
fog deploy --help
fog drift --help
fog report --help
```

### Demo Commands

```bash
# See all table styles
fog demo tables

# View example configuration
fog demo settings
```

### Documentation

- [Configuration Reference](configuration-reference.md) - All configuration options
- [Deployment Files](deployment-files.md) - Deployment file format
- [Advanced Usage](advanced-usage.md) - Complex scenarios
- [Troubleshooting](troubleshooting.md) - Common issues

### Community & Support

- **GitHub Issues**: [https://github.com/ArjenSchwarz/fog/issues](https://github.com/ArjenSchwarz/fog/issues)
- **Discussions**: [https://github.com/ArjenSchwarz/fog/discussions](https://github.com/ArjenSchwarz/fog/discussions)

## Examples

The [examples directory](../../examples/) contains:

- **templates/**: Sample CloudFormation templates
- **parameters/**: Example parameter files
- **tags/**: Example tag files
- **deployments/**: Example deployment files
- **testconf/**: Example configuration with prechecks

## Next Steps

1. **Learn the basics**: Follow "Your First Deployment" above
2. **Understand configuration**: Read [Configuration Reference](configuration-reference.md)
3. **Explore advanced features**: See [Advanced Usage](advanced-usage.md)
4. **Set up CI/CD**: Follow examples in [Advanced Usage - CI/CD Integration](advanced-usage.md#cicd-integration)
5. **Enable drift detection**: Configure [drift detection](advanced-usage.md#advanced-drift-detection)

## Contributing

Contributions are welcome! See the main [README](../../README.md#contributions) for guidelines.

## License

Fog is open source software. See [LICENSE](../../LICENSE) file for details.
