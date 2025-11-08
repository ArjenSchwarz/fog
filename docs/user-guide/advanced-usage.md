# Advanced Usage Guide

This guide covers advanced features and complex deployment scenarios for Fog.

## Table of Contents

- [Multi-Stack Deployments](#multi-stack-deployments)
- [Cross-Stack References](#cross-stack-references)
- [Multi-Region Deployments](#multi-region-deployments)
- [CI/CD Integration](#cicd-integration)
- [Advanced Drift Detection](#advanced-drift-detection)
- [Template Preprocessing](#template-preprocessing)
- [Complex Tagging Strategies](#complex-tagging-strategies)
- [Large Template Handling](#large-template-handling)
- [Environment Management](#environment-management)
- [Rollback Strategies](#rollback-strategies)

## Multi-Stack Deployments

### Sequential Stack Deployments

Deploy multiple dependent stacks in sequence:

```bash
#!/bin/bash
set -e

# Deploy VPC first
fog deploy \
  --stackname production-vpc \
  --template vpc \
  --parameters vpc-prod \
  --tags common,vpc-tags \
  --non-interactive

# Wait for VPC and deploy application stack
fog deploy \
  --stackname production-app \
  --template application \
  --parameters app-prod \
  --tags common,app-tags \
  --non-interactive

# Deploy monitoring stack
fog deploy \
  --stackname production-monitoring \
  --template monitoring \
  --parameters monitoring-prod \
  --tags common,monitoring-tags \
  --non-interactive
```

### Parallel Stack Deployments

Deploy independent stacks in parallel for faster deployment:

```bash
#!/bin/bash

# Deploy stacks in parallel
fog deploy --stackname prod-s3 --template s3-buckets --deployment-file s3-prod --non-interactive &
fog deploy --stackname prod-dynamodb --template dynamodb-tables --deployment-file dynamodb-prod --non-interactive &
fog deploy --stackname prod-sns --template sns-topics --deployment-file sns-prod --non-interactive &

# Wait for all background jobs to complete
wait

echo "All independent stacks deployed successfully"
```

### Dependency Management

Use Fog's dependencies command to understand stack relationships:

```bash
# View all stack dependencies
fog dependencies --output table

# Get dependencies for specific stack
fog dependencies --stackname production-vpc

# Generate visual dependency graph
fog dependencies --output dot | dot -T png -o stack-dependencies.png

# Export dependencies as JSON for processing
fog dependencies --output json > dependencies.json
```

## Cross-Stack References

### Using CloudFormation Exports

**VPC Stack** (exports values):

```yaml
# templates/vpc.yaml
Resources:
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16

Outputs:
  VPCId:
    Description: VPC ID
    Value: !Ref VPC
    Export:
      Name: !Sub ${AWS::StackName}-VPC-ID

  PrivateSubnetIds:
    Description: Private Subnet IDs
    Value: !Join [",", [!Ref PrivateSubnetA, !Ref PrivateSubnetB, !Ref PrivateSubnetC]]
    Export:
      Name: !Sub ${AWS::StackName}-Private-Subnet-IDs
```

**Application Stack** (imports values):

```yaml
# templates/application.yaml
Parameters:
  VPCStackName:
    Type: String
    Description: Name of the VPC stack to import values from

Resources:
  ApplicationLoadBalancer:
    Type: AWS::ElasticLoadBalancingV2::LoadBalancer
    Properties:
      Subnets: !Split [",", !ImportValue {"Fn::Sub": "${VPCStackName}-Private-Subnet-IDs"}]
      VpcId: !ImportValue {"Fn::Sub": "${VPCStackName}-VPC-ID"}
```

Deployment:

```bash
# Deploy VPC stack
fog deploy --stackname production-vpc --template vpc --deployment-file vpc-prod --non-interactive

# Verify exports are created
fog exports --stackname production-vpc --verbose

# Deploy application stack with VPC stack name as parameter
# parameters/app-prod.json:
# {
#   "VPCStackName": "production-vpc"
# }
fog deploy --stackname production-app --template application --deployment-file app-prod --non-interactive
```

### Managing Export Dependencies

Check which stacks use your exports before deletion:

```bash
# See all exports and their imports
fog exports --verbose

# Check specific stack's exports
fog exports --stackname production-vpc --verbose

# View dependency tree
fog dependencies --stackname production-vpc
```

## Multi-Region Deployments

### Same Stack, Multiple Regions

Deploy identical infrastructure across regions:

```bash
#!/bin/bash
set -e

REGIONS=("us-east-1" "us-west-2" "eu-west-1")
STACK_NAME="production-app"
TEMPLATE="application"

for region in "${REGIONS[@]}"; do
  echo "Deploying to $region..."

  fog deploy \
    --region "$region" \
    --stackname "$STACK_NAME" \
    --deployment-file "app-${region}" \
    --non-interactive

  echo "Deployed to $region successfully"
done
```

### Region-Specific Configuration

Use different deployment files per region:

**deployments/app-us-east-1.yaml**:
```yaml
template-file-path: "../templates/application.yaml"

parameters:
  Region: us-east-1
  AvailabilityZones: "us-east-1a,us-east-1b,us-east-1c"
  AMIId: ami-0c55b159cbfafe1f0  # Region-specific AMI

tags:
  Region: us-east-1
  RegionType: primary
```

**deployments/app-us-west-2.yaml**:
```yaml
template-file-path: "../templates/application.yaml"

parameters:
  Region: us-west-2
  AvailabilityZones: "us-west-2a,us-west-2b,us-west-2c"
  AMIId: ami-0d1cd67c26f5fca19  # Region-specific AMI

tags:
  Region: us-west-2
  RegionType: secondary
```

### Cross-Region Stack References

For cross-region references, use AWS Systems Manager Parameter Store or custom solutions:

```yaml
# Store value in Parameter Store
Resources:
  StoreVPCId:
    Type: AWS::SSM::Parameter
    Properties:
      Name: /cross-region/vpc-id
      Type: String
      Value: !Ref VPC
      Description: VPC ID for cross-region reference
```

Retrieve in another region via custom resource or application logic.

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/deploy.yml
name: Deploy CloudFormation Stacks

on:
  push:
    branches:
      - main
    paths:
      - 'infrastructure/**'
  pull_request:
    branches:
      - main
    paths:
      - 'infrastructure/**'

env:
  AWS_REGION: us-east-1

jobs:
  validate:
    name: Validate Templates
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Install fog
        run: |
          wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-linux-amd64
          chmod +x fog-linux-amd64
          sudo mv fog-linux-amd64 /usr/local/bin/fog

      - name: Install cfn-lint
        run: pip install cfn-lint

      - name: Validate templates
        working-directory: infrastructure
        run: |
          for template in templates/*.yaml; do
            echo "Validating $template"
            cfn-lint "$template"
          done

  plan:
    name: Create ChangeSets
    runs-on: ubuntu-latest
    needs: validate
    if: github.event_name == 'pull_request'
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Install fog
        run: |
          wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-linux-amd64
          chmod +x fog-linux-amd64
          sudo mv fog-linux-amd64 /usr/local/bin/fog

      - name: Create ChangeSets
        working-directory: infrastructure
        run: |
          fog deploy \
            --stackname production-vpc \
            --deployment-file vpc-production \
            --create-changeset \
            --output json \
            --file ../changeset-vpc.json

          fog deploy \
            --stackname production-app \
            --deployment-file app-production \
            --create-changeset \
            --output json \
            --file ../changeset-app.json

      - name: Upload ChangeSets
        uses: actions/upload-artifact@v3
        with:
          name: changesets
          path: changeset-*.json

  deploy:
    name: Deploy to Production
    runs-on: ubuntu-latest
    needs: validate
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Configure AWS Credentials
        uses: aws-actions/configure-aws-credentials@v2
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Install fog
        run: |
          wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-linux-amd64
          chmod +x fog-linux-amd64
          sudo mv fog-linux-amd64 /usr/local/bin/fog

      - name: Deploy Stacks
        working-directory: infrastructure
        run: |
          # Deploy VPC first
          fog deploy \
            --stackname production-vpc \
            --deployment-file vpc-production \
            --non-interactive \
            --output json

          # Deploy application
          fog deploy \
            --stackname production-app \
            --deployment-file app-production \
            --non-interactive \
            --output json
```

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - validate
  - plan
  - deploy

variables:
  AWS_REGION: us-east-1

.install_fog: &install_fog
  - wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-linux-amd64
  - chmod +x fog-linux-amd64
  - mv fog-linux-amd64 /usr/local/bin/fog

validate:
  stage: validate
  image: python:3.9
  script:
    - pip install cfn-lint
    - cd infrastructure
    - for template in templates/*.yaml; do cfn-lint "$template"; done

plan:
  stage: plan
  image: amazon/aws-cli:latest
  before_script:
    - *install_fog
  script:
    - cd infrastructure
    - fog deploy --stackname production-vpc --deployment-file vpc-production --create-changeset
    - fog deploy --stackname production-app --deployment-file app-production --create-changeset
  only:
    - merge_requests

deploy:
  stage: deploy
  image: amazon/aws-cli:latest
  before_script:
    - *install_fog
  script:
    - cd infrastructure
    - fog deploy --stackname production-vpc --deployment-file vpc-production --non-interactive
    - fog deploy --stackname production-app --deployment-file app-production --non-interactive
  only:
    - main
  when: manual
```

### Jenkins Pipeline

```groovy
// Jenkinsfile
pipeline {
    agent any

    environment {
        AWS_REGION = 'us-east-1'
        FOG_CONFIG = 'infrastructure/fog.yaml'
    }

    stages {
        stage('Install Tools') {
            steps {
                sh '''
                    wget https://github.com/ArjenSchwarz/fog/releases/latest/download/fog-linux-amd64
                    chmod +x fog-linux-amd64
                    sudo mv fog-linux-amd64 /usr/local/bin/fog
                    pip install cfn-lint
                '''
            }
        }

        stage('Validate Templates') {
            steps {
                dir('infrastructure') {
                    sh '''
                        for template in templates/*.yaml; do
                            echo "Validating $template"
                            cfn-lint "$template"
                        done
                    '''
                }
            }
        }

        stage('Create ChangeSets') {
            when {
                changeRequest()
            }
            steps {
                dir('infrastructure') {
                    sh '''
                        fog deploy --config ${FOG_CONFIG} \
                            --stackname production-vpc \
                            --deployment-file vpc-production \
                            --create-changeset

                        fog deploy --config ${FOG_CONFIG} \
                            --stackname production-app \
                            --deployment-file app-production \
                            --create-changeset
                    '''
                }
            }
        }

        stage('Deploy') {
            when {
                branch 'main'
            }
            steps {
                dir('infrastructure') {
                    sh '''
                        fog deploy --config ${FOG_CONFIG} \
                            --stackname production-vpc \
                            --deployment-file vpc-production \
                            --non-interactive

                        fog deploy --config ${FOG_CONFIG} \
                            --stackname production-app \
                            --deployment-file app-production \
                            --non-interactive
                    '''
                }
            }
        }

        stage('Generate Reports') {
            steps {
                dir('infrastructure') {
                    sh '''
                        fog report --stackname production-vpc --output markdown --file reports/vpc-report.md
                        fog report --stackname production-app --output markdown --file reports/app-report.md
                    '''
                }
                archiveArtifacts artifacts: 'infrastructure/reports/*.md'
            }
        }
    }

    post {
        always {
            cleanWs()
        }
    }
}
```

### Scripting with Fog

When using fog in scripts and CI/CD pipelines, understanding output streams and quiet mode is essential for robust automation.

#### Stream Separation

Fog follows Unix conventions by separating output into two streams:

- **stderr**: Progress messages, status updates, interactive prompts, and informational output
- **stdout**: Structured data output (JSON, YAML, CSV, etc.) when using `--output` flag

This separation allows you to:
- Pipe structured output to other commands without noise
- Redirect progress logs separately from data
- Parse machine-readable output reliably

**Examples:**

```bash
# Capture only JSON output, show progress on terminal
fog deploy --stackname myapp --output json > deployment.json

# Hide progress, capture only data
fog deploy --stackname myapp --output json 2>/dev/null > deployment.json

# Capture both streams separately
fog deploy --stackname myapp --output json > deployment.json 2> deployment.log

# Combine streams for grep/searching
fog deploy --stackname myapp 2>&1 | grep "ERROR"

# Parse JSON output with jq
fog deploy --stackname myapp --output json | jq '.status'
```

#### Quiet Mode

The `--quiet` flag suppresses all progress output (stderr), showing only the final structured result:

```bash
# Suppress progress messages, show only final result
fog deploy --stackname myapp --quiet --output json

# Perfect for CI/CD where you only want the result
fog deploy --stackname myapp --quiet --output json > result.json
```

**Quiet Mode Behavior:**
- ✅ Suppresses: Progress indicators, status updates, changeset details, interactive prompts
- ⚠️ Still shows: Warning messages, error messages (critical information)
- 📝 Auto-enables: Non-interactive mode (no prompts, auto-approve changes)

**Use Cases:**
- CI/CD pipelines where progress clutters logs
- Scripts that parse structured output
- Automated deployments without human oversight
- Cron jobs and scheduled tasks

**Example CI/CD Usage:**

```bash
#!/bin/bash
# deploy-production.sh

set -euo pipefail

echo "Starting deployment..."

# Deploy with quiet mode, capture JSON output
if result=$(fog deploy \
    --stackname production-app \
    --deployment-file app-production \
    --quiet \
    --output json 2>&1); then

    echo "Deployment successful!"
    echo "$result" | jq -r '.outputs[] | "\(.OutputKey): \(.OutputValue)"'

    # Extract specific outputs for downstream use
    export API_URL=$(echo "$result" | jq -r '.outputs[] | select(.OutputKey=="ApiUrl") | .OutputValue')

else
    echo "Deployment failed!"
    echo "$result"
    exit 1
fi
```

#### Output Format Selection

Choose output formats based on your use case:

- **table** (default): Human-readable, good for terminal viewing and logs
- **json**: Machine-readable, perfect for parsing with `jq` or scripts
- **yaml**: Human-readable structured data, good for configs
- **csv**: Spreadsheet-compatible, good for bulk data
- **markdown**: Documentation-friendly, good for reports

```bash
# For manual review
fog deploy --stackname myapp --dry-run

# For parsing in scripts
fog deploy --stackname myapp --output json | jq '.changeset'

# For documentation
fog deploy --stackname myapp --output markdown > deployment-report.md

# For spreadsheet import
fog describe changeset --changeset-name my-changeset --output csv > changes.csv
```

#### Error Handling in Scripts

Always check exit codes and handle errors appropriately:

```bash
#!/bin/bash
set -e  # Exit on error

# Capture output and check success
if fog deploy --stackname myapp --quiet --output json > /tmp/result.json 2>&1; then
    echo "✅ Deployment succeeded"
    cat /tmp/result.json | jq '.status'
else
    exit_code=$?
    echo "❌ Deployment failed with exit code $exit_code"
    cat /tmp/result.json
    exit $exit_code
fi
```

#### Best Practices for CI/CD

1. **Always use `--output json`** for machine-readable results
2. **Use `--quiet`** to reduce log noise in CI/CD
3. **Capture both streams** when debugging (omit in production)
4. **Check exit codes** explicitly
5. **Use `--create-changeset`** for review workflows
6. **Set timeouts** for long-running operations
7. **Store outputs** as artifacts for auditing

**Complete GitHub Actions Example:**

```yaml
- name: Deploy Stack
  id: deploy
  run: |
    set -e
    fog deploy \
      --stackname ${{ env.STACK_NAME }} \
      --deployment-file ${{ env.DEPLOYMENT_FILE }} \
      --quiet \
      --output json > deployment-result.json 2>&1

    # Extract outputs for later steps
    API_URL=$(jq -r '.outputs[] | select(.OutputKey=="ApiUrl") | .OutputValue' deployment-result.json)
    echo "api_url=$API_URL" >> $GITHUB_OUTPUT

- name: Upload Deployment Result
  uses: actions/upload-artifact@v3
  if: always()
  with:
    name: deployment-result
    path: deployment-result.json

- name: Use Deployment Output
  run: |
    echo "API URL: ${{ steps.deploy.outputs.api_url }}"
    curl -f "${{ steps.deploy.outputs.api_url }}/health"
```

## Advanced Drift Detection

### Automated Drift Detection

Run drift detection on a schedule and alert on changes:

```bash
#!/bin/bash
# drift-check.sh

STACKS=("production-vpc" "production-app" "production-database")
DRIFT_DETECTED=false

for stack in "${STACKS[@]}"; do
  echo "Checking drift for $stack..."

  # Run drift detection and save output
  fog drift --stackname "$stack" --output json --file "/tmp/drift-${stack}.json"

  # Check if drift was detected (simplified - check JSON for drift status)
  if jq -e '.DriftStatus == "DRIFTED"' "/tmp/drift-${stack}.json" > /dev/null 2>&1; then
    echo "DRIFT DETECTED in $stack"
    DRIFT_DETECTED=true
  fi
done

if [ "$DRIFT_DETECTED" = true ]; then
  # Send alert (example: email, Slack, etc.)
  echo "Drift detected in one or more stacks!"
  # aws sns publish --topic-arn arn:aws:sns:us-east-1:123456789012:drift-alerts --message "Drift detected"
  exit 1
fi

echo "No drift detected"
```

### Drift Detection with Filtering

Ignore expected differences while catching unexpected ones:

```yaml
# fog.yaml
drift:
  # Ignore tags that frequently change
  ignore-tags:
    - AWS::EC2::Instance:LastPatchedDate
    - AWS::EC2::Instance:MaintenanceWindow
    - LastModified

  # Ignore known blackhole routes (e.g., decommissioned peering connections)
  ignore-blackholes:
    - pcx-old-connection-id

  # Detect unmanaged SSO resources
  detect-unmanaged-resources:
    - AWS::SSO::PermissionSet
    - AWS::SSO::Assignment

  # Ignore AWS-managed SSO resources
  ignore-unmanaged-resources:
    - "arn:aws:sso:::permissionSet/ssoins-*/ps-aws-managed-*"
```

Run with additional runtime filters:

```bash
# Ignore additional tags for this run
fog drift --stackname production-vpc --ignore-tags TemporaryTag,TestTag

# Separate properties for better readability
fog drift --stackname production-app --separate-properties

# Use existing results (don't trigger new detection)
fog drift --stackname production-database --results-only
```

### Drift Detection Reports

Generate comprehensive drift reports:

```bash
#!/bin/bash
# generate-drift-report.sh

STACKS=("production-vpc" "production-app" "production-database")
REPORT_DIR="drift-reports/$(date +%Y-%m-%d)"

mkdir -p "$REPORT_DIR"

for stack in "${STACKS[@]}"; do
  echo "Generating drift report for $stack..."

  # Generate markdown report
  fog drift \
    --stackname "$stack" \
    --separate-properties \
    --verbose \
    --output markdown \
    --file "${REPORT_DIR}/${stack}-drift.md"

  # Generate JSON for programmatic processing
  fog drift \
    --stackname "$stack" \
    --results-only \
    --output json \
    --file "${REPORT_DIR}/${stack}-drift.json"
done

# Create index file
cat > "${REPORT_DIR}/index.md" <<EOF
# Drift Detection Report - $(date +%Y-%m-%d)

$(for stack in "${STACKS[@]}"; do
  echo "- [$stack](./${stack}-drift.md)"
done)
EOF

echo "Reports generated in $REPORT_DIR"
```

## Template Preprocessing

### Using $TEMPLATEPATH Placeholder

Track template sources in tags automatically:

```yaml
# fog.yaml
tags:
  default:
    Source: https://github.com/myorg/infrastructure/$TEMPLATEPATH
    TemplateVersion: v1.2.3
    ManagedBy: fog
rootdir: infrastructure
```

When deploying `infrastructure/templates/vpc.yaml`, the Source tag becomes:
```
Source: https://github.com/myorg/infrastructure/templates/vpc.yaml
```

### Prechecks with Template Path

Run validation tools before deployment:

```yaml
# fog.yaml
templates:
  prechecks:
    # Validate CloudFormation syntax
    - cfn-lint -t $TEMPLATEPATH

    # Check security best practices
    - cfn-guard validate -d $TEMPLATEPATH --rules security-rules

    # Check compliance requirements
    - checkov -f $TEMPLATEPATH --framework cloudformation

    # Custom validation script
    - ./scripts/validate-template.sh $TEMPLATEPATH

  stop-on-failed-prechecks: true
```

Example custom validation script:

```bash
#!/bin/bash
# scripts/validate-template.sh

TEMPLATE=$1

# Check for required tags in template
if ! grep -q "Environment:" "$TEMPLATE"; then
  echo "ERROR: Template must define Environment tag"
  exit 1
fi

# Check for required metadata
if ! grep -q "Metadata:" "$TEMPLATE"; then
  echo "WARNING: Template should include Metadata section"
fi

# Validate specific resources
if grep -q "AWS::IAM::Role" "$TEMPLATE"; then
  if ! grep -q "PermissionsBoundary" "$TEMPLATE"; then
    echo "ERROR: IAM Roles must have PermissionsBoundary"
    exit 1
  fi
fi

echo "Template validation passed"
exit 0
```

## Complex Tagging Strategies

### Hierarchical Tagging

Combine multiple tag files for hierarchical structure:

```bash
# Tag hierarchy:
# tags/global.json - Organization-wide tags
# tags/department/engineering.json - Department tags
# tags/team/platform.json - Team tags
# tags/environment/production.json - Environment tags
# tags/project/vpc.json - Project-specific tags

fog deploy \
  --stackname production-platform-vpc \
  --template vpc \
  --parameters vpc-prod \
  --tags global,department/engineering,team/platform,environment/production,project/vpc
```

**tags/global.json**:
```json
{
  "Organization": "MyCompany",
  "ManagedBy": "fog",
  "CostTracking": "enabled"
}
```

**tags/department/engineering.json**:
```json
{
  "Department": "Engineering",
  "CostCenter": "ENG-001",
  "BudgetOwner": "engineering-lead@example.com"
}
```

**tags/team/platform.json**:
```json
{
  "Team": "Platform",
  "TeamEmail": "platform-team@example.com",
  "OnCallRotation": "platform-oncall"
}
```

**tags/environment/production.json**:
```json
{
  "Environment": "production",
  "Compliance": "required",
  "BackupPolicy": "daily",
  "MaintenanceWindow": "sun:03:00-sun:04:00"
}
```

**tags/project/vpc.json**:
```json
{
  "Project": "CoreNetworking",
  "Component": "VPC"
}
```

### Dynamic Tagging with Environment Variables

Use environment variables in tag files (via preprocessing or external tools):

```bash
#!/bin/bash
# deploy-with-dynamic-tags.sh

# Generate dynamic tags
cat > /tmp/dynamic-tags.json <<EOF
{
  "DeployedBy": "${USER}",
  "DeploymentTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "GitCommit": "$(git rev-parse HEAD)",
  "GitBranch": "$(git rev-parse --abbrev-ref HEAD)",
  "BuildID": "${CI_BUILD_ID:-manual}"
}
EOF

# Deploy with dynamic tags
fog deploy \
  --stackname production-app \
  --template application \
  --parameters app-prod \
  --tags common,environment/production,/tmp/dynamic-tags \
  --non-interactive
```

## Large Template Handling

### Automatic S3 Upload

Fog automatically uploads templates larger than 51,200 bytes to S3:

```bash
# Fog will detect large template and upload to S3 bucket
fog deploy \
  --stackname production-app \
  --template large-application \
  --parameters app-prod \
  --bucket my-cloudformation-templates \
  --non-interactive
```

### Nested Stacks

For very complex infrastructure, use nested stacks:

**templates/root-stack.yaml**:
```yaml
Resources:
  VPCStack:
    Type: AWS::CloudFormation::Stack
    Properties:
      TemplateURL: https://s3.amazonaws.com/my-bucket/templates/vpc.yaml
      Parameters:
        VpcCidr: !Ref VpcCidr
      Tags:
        - Key: Component
          Value: VPC

  ApplicationStack:
    Type: AWS::CloudFormation::Stack
    Properties:
      TemplateURL: https://s3.amazonaws.com/my-bucket/templates/application.yaml
      Parameters:
        VpcId: !GetAtt VPCStack.Outputs.VPCId
      Tags:
        - Key: Component
          Value: Application
```

## Environment Management

### Environment-Specific Configurations

Maintain separate configurations per environment:

**Project structure**:
```
infrastructure/
├── config/
│   ├── fog-dev.yaml
│   ├── fog-staging.yaml
│   └── fog-prod.yaml
├── deployments/
│   ├── dev/
│   │   ├── vpc.yaml
│   │   └── app.yaml
│   ├── staging/
│   │   ├── vpc.yaml
│   │   └── app.yaml
│   └── prod/
│       ├── vpc.yaml
│       └── app.yaml
└── templates/
    ├── vpc.yaml
    └── app.yaml
```

**Deployment script**:
```bash
#!/bin/bash
# deploy.sh

ENVIRONMENT=$1

if [ -z "$ENVIRONMENT" ]; then
  echo "Usage: $0 <dev|staging|prod>"
  exit 1
fi

CONFIG="config/fog-${ENVIRONMENT}.yaml"
DEPLOYMENT_DIR="deployments/${ENVIRONMENT}"

echo "Deploying to $ENVIRONMENT environment..."

fog deploy \
  --config "$CONFIG" \
  --stackname "${ENVIRONMENT}-vpc" \
  --deployment-file "${DEPLOYMENT_DIR}/vpc" \
  --non-interactive

fog deploy \
  --config "$CONFIG" \
  --stackname "${ENVIRONMENT}-app" \
  --deployment-file "${DEPLOYMENT_DIR}/app" \
  --non-interactive

echo "Deployment to $ENVIRONMENT complete!"
```

Usage:
```bash
./deploy.sh dev
./deploy.sh staging
./deploy.sh prod
```

## Rollback Strategies

### Automated Rollback on Failure

```bash
#!/bin/bash
# deploy-with-rollback.sh

STACK_NAME=$1
DEPLOYMENT_FILE=$2

# Capture current stack status
PREVIOUS_STATUS=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --query 'Stacks[0].StackStatus' \
  --output text 2>/dev/null)

# Deploy
if fog deploy --stackname "$STACK_NAME" --deployment-file "$DEPLOYMENT_FILE" --non-interactive; then
  echo "Deployment successful"

  # Verify deployment health (custom checks)
  if ./scripts/verify-deployment.sh "$STACK_NAME"; then
    echo "Deployment verification passed"
    exit 0
  else
    echo "Deployment verification failed, rolling back..."
    aws cloudformation cancel-update-stack --stack-name "$STACK_NAME"
    exit 1
  fi
else
  echo "Deployment failed"

  if [ "$PREVIOUS_STATUS" = "CREATE_COMPLETE" ] || [ "$PREVIOUS_STATUS" = "UPDATE_COMPLETE" ]; then
    echo "Stack was previously stable, monitoring rollback..."
    # CloudFormation automatically rolls back failed updates
  fi

  exit 1
fi
```

### Manual Rollback Procedures

```bash
# 1. Identify the last successful deployment
fog report --stackname production-app --output markdown

# 2. Get parameters and tags from last successful deployment
aws cloudformation describe-stacks --stack-name production-app

# 3. Redeploy with previous configuration
fog deploy \
  --stackname production-app \
  --deployment-file app-production-previous \
  --non-interactive

# 4. Verify rollback
fog report --stackname production-app --latest
```

## See Also

- [Configuration Reference](configuration-reference.md) - Complete configuration options
- [Deployment Files](deployment-files.md) - Deployment file format
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
