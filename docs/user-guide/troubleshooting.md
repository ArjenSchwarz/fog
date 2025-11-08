# Troubleshooting Guide

This guide helps you diagnose and resolve common issues when using Fog.

## Table of Contents

- [Deployment Issues](#deployment-issues)
- [Configuration Issues](#configuration-issues)
- [AWS Credentials and Permissions](#aws-credentials-and-permissions)
- [Template and Parameter Issues](#template-and-parameter-issues)
- [Drift Detection Issues](#drift-detection-issues)
- [Output and Display Issues](#output-and-display-issues)
- [Performance Issues](#performance-issues)
- [Debug Mode](#debug-mode)

## Deployment Issues

### Changeset Creation Fails

**Problem**: Changeset creation fails with validation errors.

**Possible causes**:
1. Template syntax errors
2. Invalid parameter values
3. Missing required parameters
4. AWS resource limits exceeded
5. Insufficient permissions

**Solutions**:

1. **Enable verbose mode** to see detailed error messages:
   ```bash
   fog deploy --stackname mystack --template mytemplate --verbose
   ```

2. **Validate template separately** using AWS CLI:
   ```bash
   aws cloudformation validate-template --template-body file://templates/mytemplate.yaml
   ```

3. **Use cfn-lint** to catch common issues:
   ```bash
   cfn-lint templates/mytemplate.yaml
   ```

4. **Enable prechecks** in your configuration:
   ```yaml
   # fog.yaml
   templates:
     prechecks:
       - cfn-lint -t $TEMPLATEPATH
     stop-on-failed-prechecks: true
   ```

5. **Check parameter values** in your parameter file:
   ```bash
   # Verify JSON is valid
   jq . parameters/myparams.json

   # Check for required parameters
   fog deploy --stackname mystack --template mytemplate --dry-run
   ```

### Stack Creation Fails Immediately

**Problem**: Stack fails during creation with rollback.

**Possible causes**:
1. Invalid resource configurations
2. Resource naming conflicts
3. Quota limits exceeded
4. Cross-stack dependency issues

**Solutions**:

1. **Check failure details**:
   ```bash
   fog report --stackname mystack --latest
   ```

2. **Review error messages** from failed resources in the deployment output

3. **Use dry-run mode** to validate before deployment:
   ```bash
   fog deploy --stackname mystack --template mytemplate --dry-run
   ```

4. **Check AWS service quotas**:
   ```bash
   aws service-quotas list-service-quotas --service-code cloudformation
   ```

5. **Let Fog delete the failed stack** when prompted, or manually delete:
   ```bash
   aws cloudformation delete-stack --stack-name mystack
   ```

### Stack Update Shows "No Changes"

**Problem**: Deployment completes but says no changes were detected.

**Possible causes**:
1. Template and parameters are identical to current stack
2. Changes are not detectable by CloudFormation
3. Parameter values haven't actually changed

**Solutions**:

1. **Verify actual differences**:
   ```bash
   # Check current stack parameters
   aws cloudformation describe-stacks --stack-name mystack

   # Compare with your parameter file
   cat parameters/myparams.json
   ```

2. **Use verbose mode** to see what Fog is sending:
   ```bash
   fog deploy --stackname mystack --template mytemplate --verbose
   ```

3. **Make a trivial change** to force update (if needed):
   - Update a tag value
   - Add a description to a resource
   - Modify a parameter (if template supports it)

### Changeset Shows Unexpected Changes

**Problem**: Changeset includes changes you didn't expect.

**Possible causes**:
1. Default tags from configuration
2. Previous deployment used different parameters
3. Template changes from previous version
4. CloudFormation auto-generates values

**Solutions**:

1. **Review default tags** in configuration:
   ```bash
   # Check your fog.yaml
   cat fog.yaml | grep -A5 "tags:"
   ```

2. **Disable default tags** if not wanted:
   ```bash
   fog deploy --stackname mystack --template mytemplate --default-tags=false
   ```

3. **Compare with previous deployment**:
   ```bash
   fog report --stackname mystack --latest
   ```

4. **Check git history** of template and parameter files:
   ```bash
   git log -p templates/mytemplate.yaml
   git log -p parameters/myparams.json
   ```

### Deployment Hangs or Takes Too Long

**Problem**: Deployment seems stuck or is taking longer than expected.

**Possible causes**:
1. Resources are waiting for manual actions (e.g., RDS snapshots)
2. Resources are in a retry loop (e.g., EC2 instances failing health checks)
3. Network timeouts
4. Large number of resources

**Solutions**:

1. **Check AWS Console** for resource status

2. **Monitor stack events**:
   ```bash
   # In another terminal
   aws cloudformation describe-stack-events --stack-name mystack --max-items 20
   ```

3. **Look for timeout settings** in template (e.g., CreationPolicy, UpdatePolicy)

4. **Check CloudFormation service events**:
   - Visit AWS Service Health Dashboard
   - Check for regional issues

5. **Be patient with long-running resources**:
   - RDS instances: 10-20 minutes
   - NAT Gateways: 2-5 minutes
   - EC2 instances with UserData: varies

## Configuration Issues

### Configuration File Not Found

**Problem**: Fog doesn't find your configuration file.

**Possible causes**:
1. Wrong file name or location
2. Wrong file extension
3. File permissions issue

**Solutions**:

1. **Check file name** - must be exactly:
   - `fog.yaml`, `fog.yml`, `fog.json`, or `fog.toml`

2. **Check locations** - Fog searches:
   - Current directory: `./fog.yaml`
   - Home directory: `~/fog.yaml`
   - Custom path: `--config /path/to/config.yaml`

3. **Verify file exists**:
   ```bash
   ls -la fog.yaml
   ls -la ~/fog.yaml
   ```

4. **Check file permissions**:
   ```bash
   chmod 644 fog.yaml
   ```

5. **Use explicit path**:
   ```bash
   fog deploy --config ./fog.yaml --stackname mystack --template mytemplate
   ```

### Configuration Values Not Applied

**Problem**: Configuration values in fog.yaml are being ignored.

**Possible causes**:
1. Command-line flags override configuration
2. YAML syntax errors
3. Wrong configuration key names
4. File not being read

**Solutions**:

1. **Validate YAML syntax**:
   ```bash
   # Using Python
   python3 -c "import yaml; yaml.safe_load(open('fog.yaml'))"

   # Using yq
   yq eval '.' fog.yaml
   ```

2. **Check configuration precedence**:
   - Command-line flags > Environment variables > Config file > Defaults
   - Remove conflicting flags to use config file values

3. **Use debug mode** to see which config is loaded:
   ```bash
   fog deploy --debug --stackname mystack --template mytemplate
   ```

4. **Verify key names** against [Configuration Reference](configuration-reference.md)

### Invalid Configuration Values

**Problem**: Fog reports invalid configuration values.

**Possible causes**:
1. Wrong data type (e.g., string instead of boolean)
2. Invalid value for enum field
3. Typo in configuration key

**Solutions**:

1. **Check data types**:
   ```yaml
   # Correct
   verbose: true
   max-column-width: 50

   # Incorrect
   verbose: "true"  # Should be boolean, not string
   max-column-width: "50"  # Should be integer, not string
   ```

2. **Verify enum values**:
   ```yaml
   output: table  # Valid: table, csv, json, dot, markdown, html, yaml
   ```

3. **Reference example configuration**:
   ```bash
   cat example-fog.yaml
   ```

## AWS Credentials and Permissions

### No AWS Credentials Found

**Problem**: Error about missing AWS credentials.

**Possible causes**:
1. AWS credentials not configured
2. Invalid profile name
3. Expired credentials

**Solutions**:

1. **Configure AWS credentials**:
   ```bash
   aws configure
   ```

2. **Verify credentials file**:
   ```bash
   cat ~/.aws/credentials
   cat ~/.aws/config
   ```

3. **Check profile** in configuration or command:
   ```bash
   # In fog.yaml
   profile: my-profile

   # Or via flag
   fog deploy --profile my-profile --stackname mystack --template mytemplate
   ```

4. **Test credentials**:
   ```bash
   aws sts get-caller-identity

   # With specific profile
   aws sts get-caller-identity --profile my-profile
   ```

5. **Use environment variables**:
   ```bash
   export AWS_PROFILE=my-profile
   export AWS_REGION=us-east-1
   fog deploy --stackname mystack --template mytemplate
   ```

### Insufficient Permissions

**Problem**: Access denied errors during deployment.

**Possible causes**:
1. Missing IAM permissions for CloudFormation
2. Missing permissions for resources being created
3. Service role permissions issues
4. SCPs or permission boundaries

**Solutions**:

1. **Required CloudFormation permissions**:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": [
           "cloudformation:CreateStack",
           "cloudformation:UpdateStack",
           "cloudformation:DeleteStack",
           "cloudformation:DescribeStacks",
           "cloudformation:DescribeStackEvents",
           "cloudformation:CreateChangeSet",
           "cloudformation:DescribeChangeSet",
           "cloudformation:ExecuteChangeSet",
           "cloudformation:DeleteChangeSet",
           "cloudformation:ListStacks",
           "cloudformation:ListStackResources",
           "cloudformation:GetTemplate",
           "cloudformation:DetectStackDrift",
           "cloudformation:DescribeStackResourceDrifts"
         ],
         "Resource": "*"
       }
     ]
   }
   ```

2. **Resource-specific permissions**: You need permissions for all resources created by template

3. **Check current permissions**:
   ```bash
   aws iam get-user
   aws iam list-attached-user-policies --user-name YOUR_USERNAME
   ```

4. **Use a service role** (if available):
   - CloudFormation can use a separate IAM role
   - Configure in template with RoleARN property

### Region Mismatch

**Problem**: Resources not found or wrong region errors.

**Possible causes**:
1. Default region not configured
2. Configuration specifies different region
3. Profile has different default region

**Solutions**:

1. **Explicitly specify region**:
   ```bash
   fog deploy --region us-east-1 --stackname mystack --template mytemplate
   ```

2. **Set in configuration**:
   ```yaml
   # fog.yaml
   region: us-east-1
   ```

3. **Check AWS configuration**:
   ```bash
   aws configure get region
   aws configure get region --profile my-profile
   ```

4. **Verify with exports command**:
   ```bash
   fog exports --region us-east-1
   ```

## Template and Parameter Issues

### Template File Not Found

**Problem**: Fog cannot find your template file.

**Possible causes**:
1. Wrong directory configuration
2. Wrong file extension
3. Typo in template name
4. File permissions

**Solutions**:

1. **Check templates directory** configuration:
   ```yaml
   # fog.yaml
   templates:
     directory: templates  # Relative to where you run fog
   ```

2. **Verify file exists**:
   ```bash
   ls -la templates/
   ```

3. **Check supported extensions**:
   ```yaml
   # fog.yaml
   templates:
     extensions:
       - .yaml
       - .yml
       - .json
       - .template
       - .templ
       - .tmpl
   ```

4. **Use relative or absolute path**:
   ```bash
   fog deploy --stackname mystack --template ./path/to/template.yaml
   ```

5. **Check file permissions**:
   ```bash
   chmod 644 templates/mytemplate.yaml
   ```

### Parameter File Not Found

**Problem**: Fog cannot find your parameter file.

**Possible causes**:
1. Wrong directory configuration
2. Wrong file extension (must be .json)
3. Typo in parameter file name

**Solutions**:

1. **Check parameters directory** configuration:
   ```yaml
   # fog.yaml
   parameters:
     directory: parameters
   ```

2. **Verify file exists and is JSON**:
   ```bash
   ls -la parameters/
   jq . parameters/myparams.json  # Validates JSON
   ```

3. **Parameters must be JSON**:
   ```json
   {
     "ParameterKey": "ParameterValue",
     "AnotherKey": "AnotherValue"
   }
   ```

4. **Use full path if needed**:
   ```bash
   fog deploy --stackname mystack --template mytemplate --parameters ./parameters/myparams.json
   ```

### Parameter Type Errors

**Problem**: CloudFormation rejects parameter values.

**Possible causes**:
1. Wrong value type (e.g., string for number)
2. Value doesn't match allowed values
3. Value doesn't match pattern constraint

**Solutions**:

1. **All parameter values must be strings**:
   ```json
   {
     "NumberParameter": "123",
     "BooleanParameter": "true",
     "StringParameter": "value"
   }
   ```

2. **Check template constraints**:
   ```yaml
   # In your template
   Parameters:
     InstanceType:
       Type: String
       AllowedValues:
         - t3.micro
         - t3.small
         - t3.medium
   ```

3. **Validate against template**:
   ```bash
   # Extract parameters from template
   yq eval '.Parameters' templates/mytemplate.yaml
   ```

### Multiple Parameter Files Not Merging

**Problem**: Only the last parameter file is used.

**Note**: This is expected behavior when using comma-separated parameter files.

**Solutions**:

1. **Parameters are merged in order**:
   ```bash
   fog deploy \
     --stackname mystack \
     --template mytemplate \
     --parameters common,environment-specific
   ```
   Files: `parameters/common.json`, `parameters/environment-specific.json`
   Later values override earlier ones.

2. **Verify merge behavior**:
   ```bash
   # common.json
   {
     "Param1": "value1",
     "Param2": "value2"
   }

   # environment-specific.json
   {
     "Param2": "override-value2",
     "Param3": "value3"
   }

   # Result:
   # Param1: value1
   # Param2: override-value2 (overridden)
   # Param3: value3
   ```

3. **Order matters**:
   ```bash
   # Specific values last
   fog deploy --stackname mystack --template mytemplate --parameters defaults,production
   ```

## Drift Detection Issues

### Drift Detection Shows False Positives

**Problem**: Drift detection shows changes that aren't actually drift.

**Possible causes**:
1. Tag order differences
2. Specific tags that auto-update
3. Blackhole routes that are expected
4. Prefix list changes

**Solutions**:

1. **Ignore specific tags**:
   ```yaml
   # fog.yaml
   drift:
     ignore-tags:
       - AWS::EC2::Instance:Name
       - LastUpdated
   ```

2. **Ignore blackhole routes**:
   ```yaml
   # fog.yaml
   drift:
     ignore-blackholes:
       - pcx-0887c71683c64bb22
   ```

3. **Use command-line overrides**:
   ```bash
   fog drift --stackname mystack --ignore-tags TempTag,TestTag
   ```

4. **Use verbose mode** for prefix list details:
   ```bash
   fog drift --stackname mystack --verbose
   ```

### Drift Detection Fails to Start

**Problem**: Cannot start drift detection.

**Possible causes**:
1. Stack is in a non-stable state
2. Previous drift detection still running
3. Insufficient permissions

**Solutions**:

1. **Check stack status**:
   ```bash
   aws cloudformation describe-stacks --stack-name mystack --query 'Stacks[0].StackStatus'
   ```

2. **Wait for stack to stabilize** if it's updating

3. **Use existing results**:
   ```bash
   fog drift --stackname mystack --results-only
   ```

4. **Check permissions**:
   ```json
   {
     "Effect": "Allow",
     "Action": [
       "cloudformation:DetectStackDrift",
       "cloudformation:DescribeStackResourceDrifts"
     ],
     "Resource": "*"
   }
   ```

### Unmanaged Resources Not Detected

**Problem**: Fog doesn't show unmanaged resources.

**Possible causes**:
1. Resource types not configured
2. Resources excluded via ignore list

**Solutions**:

1. **Configure resource detection**:
   ```yaml
   # fog.yaml
   drift:
     detect-unmanaged-resources:
       - AWS::SSO::PermissionSet
       - AWS::SSO::Assignment
   ```

2. **Check ignore list**:
   ```yaml
   # fog.yaml
   drift:
     ignore-unmanaged-resources:
       - "arn:aws:sso:::instance/ssoins-xxx|..."
   ```

3. **Verify permissions** for resource types you want to detect

## Output and Display Issues

### Table Output is Truncated

**Problem**: Table columns are cut off.

**Solutions**:

1. **Increase max column width**:
   ```yaml
   # fog.yaml
   table:
     max-column-width: 100
   ```

2. **Use different output format**:
   ```bash
   fog exports --output json
   fog exports --output csv
   ```

3. **Widen terminal window**

4. **Save to file and view separately**:
   ```bash
   fog exports --output csv --file exports.csv
   ```

### Colors Not Showing

**Problem**: Table output has no colors.

**Solutions**:

1. **Enable colors**:
   ```yaml
   # fog.yaml
   use-colors: true
   ```

2. **Use colored table style**:
   ```yaml
   # fog.yaml
   table:
     style: ColoredBright
   ```

3. **Check terminal supports colors**:
   ```bash
   echo $TERM
   ```

4. **View available styles**:
   ```bash
   fog demo tables
   ```

### JSON Output is Invalid

**Problem**: JSON output cannot be parsed.

**Possible causes**:
1. Mixed output (debug messages + JSON)
2. Color codes in output

**Solutions**:

1. **Disable debug mode**:
   ```bash
   fog exports --output json  # Don't use --debug with JSON
   ```

2. **Colors auto-disabled for JSON**, but verify:
   ```yaml
   # fog.yaml
   use-colors: false
   ```

3. **Pipe through jq** to validate:
   ```bash
   fog exports --output json | jq .
   ```

### File Output Not Created

**Problem**: Output file is not created.

**Possible causes**:
1. No write permissions to directory
2. Invalid file path
3. Directory doesn't exist

**Solutions**:

1. **Create directory first**:
   ```bash
   mkdir -p output
   fog exports --output json --file output/exports.json
   ```

2. **Check permissions**:
   ```bash
   ls -la output/
   ```

3. **Use absolute path**:
   ```bash
   fog exports --output json --file /tmp/exports.json
   ```

4. **Check for error messages** in output

## Performance Issues

### Slow Template Validation

**Problem**: Prechecks are slow.

**Solutions**:

1. **Disable prechecks for testing**:
   ```yaml
   # fog.yaml
   templates:
     prechecks: []
   ```

2. **Reduce number of prechecks**:
   ```yaml
   # fog.yaml
   templates:
     prechecks:
       - cfn-lint -t $TEMPLATEPATH  # Fast
       # - checkov -f $TEMPLATEPATH  # Slower, comment out if not needed
   ```

3. **Run specific prechecks manually**

### Slow Drift Detection

**Problem**: Drift detection takes a long time.

**Possible causes**:
1. Large stack with many resources
2. Complex resource types (e.g., VPCs, Transit Gateways)

**Solutions**:

1. **Use results-only mode** to view previous results:
   ```bash
   fog drift --stackname mystack --results-only
   ```

2. **Be patient** - CloudFormation drift detection takes time:
   - Simple stacks: 1-2 minutes
   - Complex stacks: 5-10 minutes

3. **Run in background** and check results later:
   ```bash
   # Start detection
   aws cloudformation detect-stack-drift --stack-name mystack

   # Check later
   fog drift --stackname mystack --results-only
   ```

## Debug Mode

### Enabling Debug Mode

For any unexplained issue, enable debug mode for detailed logging:

```bash
fog deploy --debug --stackname mystack --template mytemplate
```

Or in configuration:
```yaml
# fog.yaml
debug: true
```

### Debug Output Includes

- Configuration values being used
- File paths being searched
- AWS API calls and responses
- Template preprocessing steps
- Parameter resolution

### Using Debug Output

1. **Check file paths**:
   - Verify Fog is looking in the right directories
   - Confirm file extensions are recognized

2. **Verify configuration**:
   - See which config file is loaded
   - Check final merged configuration values

3. **Trace AWS calls**:
   - See exact API calls to AWS
   - Identify permission errors

4. **Share debug output** when reporting issues:
   - Helps maintainers diagnose problems
   - Redact sensitive information first

## Getting Help

If you're still stuck after trying these solutions:

1. **Check GitHub Issues**: [https://github.com/ArjenSchwarz/fog/issues](https://github.com/ArjenSchwarz/fog/issues)
2. **Open a new issue**: Include:
   - Fog version: `fog --version`
   - Command you ran
   - Debug output (redacted)
   - Expected vs actual behavior
3. **Review documentation**: [docs/user-guide/](../user-guide/)

## See Also

- [Configuration Reference](configuration-reference.md) - Complete configuration options
- [Deployment Files](deployment-files.md) - Deployment file format
- [Advanced Usage](advanced-usage.md) - Complex scenarios
