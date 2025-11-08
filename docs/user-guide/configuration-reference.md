# Fog Configuration Reference

This document provides a complete reference for all configuration options available in Fog.

## Table of Contents

- [Configuration File Locations](#configuration-file-locations)
- [Configuration File Formats](#configuration-file-formats)
- [Complete Configuration Reference](#complete-configuration-reference)
- [Configuration Precedence](#configuration-precedence)
- [Examples](#examples)

## Configuration File Locations

Fog searches for configuration files in the following order:

1. **Custom location** (via `--config` flag): `fog deploy --config /path/to/config.yaml`
2. **Current directory**: `./fog.yaml`, `./fog.json`, or `./fog.toml`
3. **Home directory**: `~/fog.yaml`, `~/fog.json`, or `~/fog.toml`

The first configuration file found will be used. If no configuration file is found, Fog will use default values.

## Configuration File Formats

Fog supports three configuration file formats:

- **YAML**: `fog.yaml` or `fog.yml` (recommended)
- **JSON**: `fog.json`
- **TOML**: `fog.toml`

All examples in this document use YAML format, but the same structure applies to JSON and TOML.

## Complete Configuration Reference

### Top-Level Settings

#### `output`
**Type**: `string`
**Default**: `table`
**Valid values**: `table`, `csv`, `json`, `dot`, `markdown`, `html`, `yaml`

Sets the default output format for all commands.

```yaml
output: table
```

#### `profile`
**Type**: `string`
**Default**: `""` (uses default AWS profile)

Specifies the AWS profile to use for all operations.

```yaml
profile: my-aws-profile
```

#### `region`
**Type**: `string`
**Default**: `""` (uses AWS SDK default region detection)

Specifies the AWS region to use for all operations.

```yaml
region: us-east-1
```

#### `verbose`
**Type**: `boolean`
**Default**: `false`

Enables verbose output by default for all commands.

```yaml
verbose: true
```

#### `debug`
**Type**: `boolean`
**Default**: `false`

Enables debug mode, providing detailed logging for troubleshooting.

```yaml
debug: false
```

#### `timezone`
**Type**: `string`
**Default**: System timezone

Specifies the timezone to use for displaying times in output. Uses IANA Time Zone Database names.

```yaml
timezone: America/New_York
```

#### `rootdir`
**Type**: `string`
**Default**: `.` (current directory)

Defines the root directory used for calculating the `$TEMPLATEPATH` placeholder in templates and tags.

```yaml
rootdir: /path/to/project
```

#### `use-emoji`
**Type**: `boolean`
**Default**: `false`

Enables emoji in output for success/failure indicators.

```yaml
use-emoji: true
```

#### `use-colors`
**Type**: `boolean`
**Default**: `false`

Enables colored output in tables and other displays.

```yaml
use-colors: true
```

### Output File Settings

#### `output-file`
**Type**: `string`
**Default**: `""` (no file output)

Specifies a file path to save output in addition to stdout.

```yaml
output-file: /path/to/output.json
```

#### `output-file-format`
**Type**: `string`
**Default**: Same as `output`
**Valid values**: `table`, `csv`, `json`, `dot`, `markdown`, `html`, `yaml`

Specifies the format for file output, which can differ from console output.

```yaml
output-file-format: json
```

### Changeset Settings

Configuration for CloudFormation changesets.

```yaml
changeset:
  name-format: fog-$TIMESTAMP
```

#### `changeset.name-format`
**Type**: `string`
**Default**: `fog-$TIMESTAMP`

Defines the naming format for automatically generated changesets. Available placeholders:
- `$TIMESTAMP`: Current time in ISO8601 format (without timezone)

```yaml
changeset:
  name-format: my-project-$TIMESTAMP
```

### Table Settings

Controls the appearance of table output.

```yaml
table:
  style: Default
  max-column-width: 50
```

#### `table.style`
**Type**: `string`
**Default**: `Default`

Specifies the table style. Available styles can be viewed by running `fog demo tables`.

**Common styles**:
- `Default`
- `Bold`
- `Light`
- `Rounded`
- `ColoredBlackOnGreenWhite`
- `ColoredBlackOnCyanWhite`
- `ColoredBright`
- And many more...

```yaml
table:
  style: Bold
```

#### `table.max-column-width`
**Type**: `integer`
**Default**: `50`

Sets the maximum width for table columns. Longer content will be truncated.

```yaml
table:
  max-column-width: 100
```

### Templates Settings

Configuration for CloudFormation template files.

```yaml
templates:
  directory: templates
  extensions:
    - .yaml
    - .yml
    - .json
    - .template
    - .templ
    - .tmpl
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
  stop-on-failed-prechecks: true
```

#### `templates.directory`
**Type**: `string`
**Default**: `templates`

Specifies the directory where template files are stored (relative to execution directory).

```yaml
templates:
  directory: cloudformation/templates
```

#### `templates.extensions`
**Type**: `array of strings`
**Default**: `[.yaml, .yml, .json, .template, .templ, .tmpl]`

Defines the file extensions to search for when looking for template files.

```yaml
templates:
  extensions:
    - .yaml
    - .yml
    - .json
```

#### `templates.prechecks`
**Type**: `array of strings`
**Default**: `[]` (no prechecks)

Specifies commands to run before deployment for template validation. Placeholders:
- `$TEMPLATEPATH`: Full path to the template file

```yaml
templates:
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
    - cfn-guard validate -d $TEMPLATEPATH --rules myrules
    - checkov -f $TEMPLATEPATH
```

#### `templates.stop-on-failed-prechecks`
**Type**: `boolean`
**Default**: `false`

When `true`, stops deployment if any precheck command fails.

```yaml
templates:
  stop-on-failed-prechecks: true
```

### Parameters Settings

Configuration for parameter files.

```yaml
parameters:
  directory: parameters
  extensions:
    - .json
```

#### `parameters.directory`
**Type**: `string`
**Default**: `parameters`

Specifies the directory where parameter files are stored (relative to execution directory).

```yaml
parameters:
  directory: cloudformation/params
```

#### `parameters.extensions`
**Type**: `array of strings`
**Default**: `[.json]`

Defines the file extensions to search for when looking for parameter files.

**Note**: Currently only JSON format is supported for parameters.

```yaml
parameters:
  extensions:
    - .json
```

### Tags Settings

Configuration for tag files and default tags.

```yaml
tags:
  directory: tags
  extensions:
    - .json
  default:
    Environment: production
    ManagedBy: fog
    Source: https://github.com/myorg/myrepo/$TEMPLATEPATH
```

#### `tags.directory`
**Type**: `string`
**Default**: `tags`

Specifies the directory where tag files are stored (relative to execution directory).

```yaml
tags:
  directory: cloudformation/tags
```

#### `tags.extensions`
**Type**: `array of strings`
**Default**: `[.json]`

Defines the file extensions to search for when looking for tag files.

**Note**: Currently only JSON format is supported for tags.

```yaml
tags:
  extensions:
    - .json
```

#### `tags.default`
**Type**: `map of key-value pairs`
**Default**: `{}` (no default tags)

Specifies tags that will be automatically applied to all deployed stacks. Placeholders:
- `$TEMPLATEPATH`: Relative path to template file from rootdir

```yaml
tags:
  default:
    Environment: production
    ManagedBy: fog
    CostCenter: engineering
    Source: https://github.com/myorg/myrepo/$TEMPLATEPATH
```

**Note**: Default tags can be disabled for specific deployments using `--default-tags=false` flag.

### Drift Detection Settings

Configuration for drift detection behavior.

```yaml
drift:
  ignore-tags:
    - AWS::EC2::TransitGatewayAttachment:Application
    - AWS::EC2::Instance:Name
  ignore-blackholes:
    - pcx-0887c71683c64bb22
  detect-unmanaged-resources:
    - AWS::SSO::PermissionSet
    - AWS::SSO::Assignment
  ignore-unmanaged-resources:
    - "arn:aws:sso:::instance/ssoins-xxx|arn:aws:sso:::permissionSet/ssoins-xxx/ps-xxx"
```

#### `drift.ignore-tags`
**Type**: `array of strings`
**Default**: `[]`

Specifies tags to ignore during drift detection. Format: `ResourceType:TagKey` or just `TagKey` for all resources.

```yaml
drift:
  ignore-tags:
    - AWS::EC2::Instance:Name  # Ignore Name tag on EC2 instances
    - TemporaryTag             # Ignore this tag on all resources
```

#### `drift.ignore-blackholes`
**Type**: `array of strings`
**Default**: `[]`

Specifies blackhole routes to ignore in drift detection (e.g., peering connections that may be down).

```yaml
drift:
  ignore-blackholes:
    - pcx-0887c71683c64bb22
    - tgw-attach-0123456789abcdef
```

#### `drift.detect-unmanaged-resources`
**Type**: `array of strings`
**Default**: `[]`

Specifies AWS resource types to check for unmanaged resources (resources not managed by CloudFormation).

```yaml
drift:
  detect-unmanaged-resources:
    - AWS::SSO::PermissionSet
    - AWS::SSO::Assignment
    - AWS::IAM::Role
```

#### `drift.ignore-unmanaged-resources`
**Type**: `array of strings`
**Default**: `[]`

Specifies specific unmanaged resources to ignore. Format varies by resource type.

```yaml
drift:
  ignore-unmanaged-resources:
    - "arn:aws:sso:::instance/ssoins-xxx|arn:aws:sso:::permissionSet/ssoins-xxx/ps-xxx"
    - "arn:aws:iam::123456789012:role/service-role/AWSServiceRoleForECS"
```

## Configuration Precedence

Configuration values are resolved in the following order (highest to lowest priority):

1. **Command-line flags**: Values passed via CLI flags (e.g., `--output json`)
2. **Environment variables**: AWS-specific variables like `AWS_PROFILE`, `AWS_REGION`
3. **Configuration file**: Values defined in `fog.yaml`/`fog.json`/`fog.toml`
4. **Default values**: Built-in defaults

Example:
```bash
# Config file has: output: table
# This command will output JSON (flag overrides config)
fog exports --output json

# Config file has: profile: dev
# This command will use production profile (flag overrides config)
fog deploy --profile production --stackname mystack --template mytemplate
```

## Examples

### Minimal Configuration

A minimal configuration for basic usage:

```yaml
# fog.yaml
output: table
region: us-east-1
```

### Development Configuration

Configuration optimized for development:

```yaml
# fog.yaml
output: table
verbose: true
use-colors: true
region: us-west-2
profile: dev-profile

table:
  style: ColoredBright
  max-column-width: 80

templates:
  directory: templates
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
  stop-on-failed-prechecks: true

tags:
  default:
    Environment: development
    ManagedBy: fog
    Developer: ${USER}

rootdir: .
```

### Production Configuration

Configuration for production deployments:

```yaml
# fog.yaml
output: json
output-file: deployments/logs/deployment-$TIMESTAMP.json
output-file-format: json

region: us-east-1
profile: production

changeset:
  name-format: prod-deployment-$TIMESTAMP

templates:
  directory: cloudformation/templates
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
    - cfn-guard validate -d $TEMPLATEPATH --rules production-rules
    - checkov -f $TEMPLATEPATH
  stop-on-failed-prechecks: true

parameters:
  directory: cloudformation/parameters/production

tags:
  directory: cloudformation/tags
  default:
    Environment: production
    ManagedBy: fog
    CostCenter: infrastructure
    Compliance: required
    Source: https://github.com/myorg/infrastructure/$TEMPLATEPATH

rootdir: .

drift:
  detect-unmanaged-resources:
    - AWS::SSO::PermissionSet
    - AWS::SSO::Assignment
```

### Multi-Environment Configuration

Using different configurations for different environments:

**fog-dev.yaml**:
```yaml
output: table
verbose: true
use-colors: true
region: us-west-2
profile: dev-profile

tags:
  default:
    Environment: development
```

**fog-prod.yaml**:
```yaml
output: json
region: us-east-1
profile: prod-profile

tags:
  default:
    Environment: production
    Compliance: required
```

Usage:
```bash
# Development
fog deploy --config fog-dev.yaml --stackname mystack --template mytemplate

# Production
fog deploy --config fog-prod.yaml --stackname mystack --template mytemplate
```

### CI/CD Configuration

Configuration optimized for CI/CD pipelines:

```yaml
# fog.yaml
output: json
output-file: artifacts/deployment-report.json

# No profile - use IAM role from CI/CD environment
region: us-east-1

# Non-interactive mode will be set via flag
changeset:
  name-format: ci-build-$TIMESTAMP

templates:
  directory: infrastructure/templates
  prechecks:
    - cfn-lint -t $TEMPLATEPATH
    - cfn-guard validate -d $TEMPLATEPATH --rules compliance-rules
  stop-on-failed-prechecks: true

tags:
  default:
    ManagedBy: fog-cicd
    Repository: ${CI_REPOSITORY_URL}
    BuildID: ${CI_BUILD_ID}
    CommitSHA: ${CI_COMMIT_SHA}
```

CI/CD usage:
```bash
fog deploy \
  --stackname $STACK_NAME \
  --template $TEMPLATE_NAME \
  --parameters $ENVIRONMENT \
  --non-interactive \
  --config fog.yaml
```

## See Also

- [Deployment File Format](deployment-files.md) - Details on deployment file structure
- [Advanced Usage](advanced-usage.md) - Complex deployment scenarios
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
