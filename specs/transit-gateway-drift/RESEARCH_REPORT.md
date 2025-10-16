# AWS CloudFormation Drift Detection for Transit Gateway Route Tables

## Research Report
**Date:** 11 October 2025
**Subject:** CloudFormation drift detection capabilities for Transit Gateway resources with focus on route table drift

---

## Executive Summary

AWS CloudFormation provides drift detection capabilities for most Transit Gateway resources, including Transit Gateway route tables and routes. However, there are significant limitations and known issues that affect the accuracy and reliability of drift detection for Transit Gateway route tables, particularly for manually added or removed routes.

**Key Findings:**

1. **All five Transit Gateway resource types requested support drift detection** according to the official AWS documentation.
2. **AWS::EC2::TransitGatewayRoute has a known limitation**: it does not support drift detection when routes are defined as separate resources.
3. **A critical bug exists for AWS::EC2::TransitGatewayAttachment**: when the attachment is manually deleted, drift detection incorrectly reports the resource as "In Sync" instead of "Drifted".
4. **Similar gaps exist between Transit Gateway and VPC route tables**: both share the fundamental limitation that only explicitly defined routes in CloudFormation templates are tracked for drift.
5. **Propagated routes are not tracked for drift**: Routes that are dynamically propagated through AWS::EC2::TransitGatewayRouteTablePropagation are not monitored by drift detection.

---

## Key Findings

### Transit Gateway Resource Drift Detection Support Status

Based on the official AWS CloudFormation documentation (Resource type support list), the following Transit Gateway resources **support drift detection**:

| Resource Type | Import | Drift Detection | IaC Generator |
|---------------|--------|----------------|---------------|
| AWS::EC2::TransitGateway | Yes | Yes | Yes |
| AWS::EC2::TransitGatewayRoute | Yes | Yes | - |
| AWS::EC2::TransitGatewayRouteTable | Yes | Yes | Yes |
| AWS::EC2::TransitGatewayRouteTableAssociation | Yes | Yes | - |
| AWS::EC2::TransitGatewayRouteTablePropagation | Yes | Yes | - |

**Additional Transit Gateway Resources with Drift Support:**

- AWS::EC2::TransitGatewayAttachment (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayVpcAttachment (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayConnect (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayConnectPeer (Yes/Yes/-)
- AWS::EC2::TransitGatewayPeeringAttachment (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayMulticastDomain (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayMulticastDomainAssociation (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayMulticastGroupMember (Yes/Yes/Yes)
- AWS::EC2::TransitGatewayMulticastGroupSource (Yes/Yes/Yes)

**Source:** [AWS CloudFormation Resource type support documentation](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-stack-drift-resource-list.html)

---

## Transit Gateway Route Table Drift Detection Analysis

### What CloudFormation Detects

CloudFormation's drift detection for Transit Gateway route tables operates by comparing the current state of resources against their expected configuration as defined in the stack template. For AWS::EC2::TransitGatewayRoute resources:

**Properties Monitored for Drift:**
- `DestinationCidrBlock` - The CIDR block used for destination matches
- `TransitGatewayAttachmentId` - The ID of the attachment target
- `Blackhole` - Whether to drop traffic matching this route
- `TransitGatewayRouteTableId` - The route table ID

**Update Behavior:** All properties require **Replacement** if changed, meaning CloudFormation cannot update these in-place.

### What CloudFormation DOES NOT Detect

#### 1. Manually Added Routes (Not Defined in Template)

**Critical Limitation:** CloudFormation only tracks drift for routes explicitly defined in the template as `AWS::EC2::TransitGatewayRoute` resources.

If a route is manually added to a Transit Gateway route table through:
- AWS Console
- AWS CLI (`create-transit-gateway-route`)
- AWS API
- Terraform or other IaC tools

And that route was **never defined in the CloudFormation template**, then:
- ✗ CloudFormation will **NOT** detect this as drift
- ✗ The route table resource will show status: `IN_SYNC`
- ✗ There is no baseline to compare against

**Reason:** AWS CloudFormation's official documentation states:

> "CloudFormation only determines drift for property values that are explicitly set, either through the stack template or by specifying template parameters. This doesn't include default values for resource properties. To have CloudFormation track a resource property for purposes of determining drift, explicitly set the property value, even if you are setting it to the default value."

#### 2. Propagated Routes

Routes that are dynamically propagated via `AWS::EC2::TransitGatewayRouteTablePropagation` resources are:
- ✗ **Not tracked for drift detection**
- ✗ Not visible in drift detection results
- ✗ Subject to dynamic changes based on attachment lifecycle

**Reason:** These routes are created automatically by AWS when attachments are associated with route tables through propagation. They are not explicitly defined as static route resources.

#### 3. Route Priority and Overlap Behaviour

When static and propagated routes overlap:
- Static routes have higher priority
- If a static route is removed manually, a previously hidden propagated route may become active
- CloudFormation drift detection will report the static route as "DELETED"
- However, it will **not** report that a propagated route is now active in its place

---

## Known Issues and Limitations

### 1. AWS::EC2::TransitGatewayAttachment Deletion Bug (GitHub Issue #1271)

**Status:** Reported and marked as "Shipped" (fixed)
**Issue:** When an `AWS::EC2::TransitGatewayAttachment` resource is manually deleted outside of CloudFormation, drift detection incorrectly shows the resource as `IN_SYNC` instead of `DRIFTED`.

**Impact:** This prevents detection of manually deleted Transit Gateway attachments, which can lead to:
- Failed deployments when attempting to update the stack
- Orphaned CloudFormation resources
- Inability to detect infrastructure tampering

**Workaround:** None available. The issue was reported as fixed ("Shipped" status), but users should verify behavior in their environment.

**Source:** [aws-cloudformation/cloudformation-coverage-roadmap Issue #1271](https://github.com/aws-cloudformation/cloudformation-coverage-roadmap/issues/1271)

**Test Case:**
```yaml
Resources:
  Test:
    Type: AWS::EC2::TransitGatewayAttachment
    Properties:
      SubnetIds:
        - subnet-xxxxxx
        - subnet-xxxxxx
      TransitGatewayId: tgw-xxxxxx
      VpcId: vpc-xxxxxx
```
1. Deploy the template
2. Manually delete the attachment via Console/CLI
3. Run drift detection
4. **Expected:** Resource shows as `DELETED` / `DRIFTED`
5. **Actual (Bug):** Resource shows as `IN_SYNC`

### 2. No Drift Detection for Inline/Dynamic Routes

AWS::EC2::TransitGatewayRouteTable does not have a property to define routes inline. Routes must be created as separate `AWS::EC2::TransitGatewayRoute` resources.

**Implication:** Each route requires a separate CloudFormation resource declaration, making templates verbose for complex routing scenarios.

### 3. Cross-Stack Attachment Limitations

CloudFormation's drift detection documentation warns:

> "Certain resources have attachment relationships with related resources... CloudFormation analyses the stack template for attachments before performing the drift comparison. However, CloudFormation can't perform this analysis across stacks, and so may not return accurate drift results where resources that are attached reside in different stacks."

**Impact on Transit Gateway:**
- If Transit Gateway attachments are defined in one stack and routes in another
- If route table associations span multiple stacks
- Drift detection may not accurately reflect the true state

### 4. Edge Cases with Array Properties

The documentation notes:

> "In certain edge cases, CloudFormation may not be able to always return accurate drift results... In certain cases, objects contained in property arrays will be reported as drift, when in actuality they're default values supplied to the property from the underlying service responsible for the resource."

**Potential Impact:** Properties like subnet lists or CIDR blocks might show false positive drift.

---

## Comparison: VPC Route Tables vs Transit Gateway Route Tables

Both resource types share similar drift detection limitations:

| Aspect | VPC Route Tables | Transit Gateway Route Tables |
|--------|------------------|------------------------------|
| **Manual Route Addition** | Not detected if route not in template | Not detected if route not in template |
| **Manual Route Deletion** | Detected (if route was in template) | Detected (if route was in template) |
| **Modified Routes** | Detected for explicit properties | Detected for explicit properties |
| **Inline Route Support** | No - must use AWS::EC2::Route | No - must use AWS::EC2::TransitGatewayRoute |
| **Propagated Routes** | N/A | Not detected - dynamic by nature |
| **Default Route Table** | Cannot be referenced/managed | Can be managed if created explicitly |
| **Cross-Stack Issues** | Yes - attachments across stacks | Yes - attachments across stacks |
| **Deleted Resource Bug** | No known issues | Yes - TGW Attachment bug #1271 |

### VPC Route Table Specific Limitations

1. **Default/Main Route Table:** When you use CloudFormation to create a VPC, the default main route table cannot be referenced or managed directly. This prevents drift detection for the main route table.

2. **Route Definition Requirement:** The `AWS::EC2::RouteTable` resource cannot include a list of routes. You must add separate `AWS::EC2::Route` resources for each route.

**Source:** [AWS re:Post - Add routes to the main Amazon VPC route table with CloudFormation](https://repost.aws/knowledge-center/cloudformation-route-table-vpc)

### Transit Gateway Route Table Advantages

- All route tables must be explicitly created (no "default" table limitation)
- Route table lifecycle is fully managed by CloudFormation
- Better control over route propagation through explicit resources

### Transit Gateway Route Table Disadvantages

- More complex resource relationships (attachments, associations, propagations)
- Propagated routes add a layer of dynamic behaviour outside drift tracking
- Known bug with attachment deletion detection
- Static and propagated route interaction can obscure drift

---

## Drift Detection Behaviour Scenarios

### Scenario 1: Manually Added Routes

**Setup:**
```yaml
Resources:
  TGW:
    Type: AWS::EC2::TransitGateway
    Properties:
      Tags:
        - Key: Name
          Value: MyTGW

  TGWRouteTable:
    Type: AWS::EC2::TransitGatewayRouteTable
    Properties:
      TransitGatewayId: !Ref TGW

  TGWRoute1:
    Type: AWS::EC2::TransitGatewayRoute
    Properties:
      DestinationCidrBlock: 10.0.0.0/16
      TransitGatewayAttachmentId: tgw-attach-123456
      TransitGatewayRouteTableId: !Ref TGWRouteTable
```

**Action:** Manually add route `10.1.0.0/16` via Console
**Drift Detection Result:** `IN_SYNC` - **Route NOT detected**
**Reason:** Route was never defined in template

### Scenario 2: Manually Removed Routes

**Setup:** Same as Scenario 1
**Action:** Manually delete route `10.0.0.0/16` via CLI
**Drift Detection Result:** `DELETED` / `DRIFTED` - **Correctly detected**
**Reason:** Route was defined in template, now missing

### Scenario 3: Modified Route Attachment

**Setup:** Same as Scenario 1
**Action:** Change attachment for `10.0.0.0/16` from `tgw-attach-123456` to `tgw-attach-789012` via API
**Drift Detection Result:** `MODIFIED` / `DRIFTED` - **Correctly detected**
**Property Difference:** `TransitGatewayAttachmentId`
- Expected: `tgw-attach-123456`
- Actual: `tgw-attach-789012`

### Scenario 4: Propagated Routes Appearing

**Setup:**
```yaml
Resources:
  TGWRouteTable:
    Type: AWS::EC2::TransitGatewayRouteTable
    Properties:
      TransitGatewayId: !Ref TGW

  TGWPropagation:
    Type: AWS::EC2::TransitGatewayRouteTablePropagation
    Properties:
      TransitGatewayAttachmentId: tgw-attach-123456
      TransitGatewayRouteTableId: !Ref TGWRouteTable
```

**Action:** New VPC attachment causes `10.2.0.0/16` to be propagated
**Drift Detection Result:** `IN_SYNC` - **Propagated route NOT detected**
**Reason:** Propagation is expected behaviour; routes are dynamic

### Scenario 5: Static Route Hiding Propagated Route

**Setup:**
```yaml
Resources:
  # Route table with propagation enabled (as above)

  StaticRoute:
    Type: AWS::EC2::TransitGatewayRoute
    Properties:
      DestinationCidrBlock: 10.2.0.0/16
      TransitGatewayAttachmentId: tgw-attach-789012
      TransitGatewayRouteTableId: !Ref TGWRouteTable
```

**Initial State:** Static route `10.2.0.0/16` exists, hiding propagated route with same CIDR
**Action:** Manually delete static route via Console
**Drift Detection Result:** `DELETED` / `DRIFTED` for static route
**Actual Routing:** Propagated route `10.2.0.0/16` becomes active
**Drift Detection Limitation:** Does not report that propagated route is now active

---

## Workarounds and Best Practices

### 1. Explicit Route Definition Strategy

**Problem:** CloudFormation only tracks routes explicitly defined in templates.

**Solution:** Define all expected routes as `AWS::EC2::TransitGatewayRoute` resources, even if they seem redundant or could be propagated.

```yaml
Resources:
  # Define every route explicitly
  Route10_0:
    Type: AWS::EC2::TransitGatewayRoute
    Properties:
      DestinationCidrBlock: 10.0.0.0/16
      TransitGatewayAttachmentId: !Ref VPCAttachment1
      TransitGatewayRouteTableId: !Ref TGWRouteTable

  Route10_1:
    Type: AWS::EC2::TransitGatewayRoute
    Properties:
      DestinationCidrBlock: 10.1.0.0/16
      TransitGatewayAttachmentId: !Ref VPCAttachment2
      TransitGatewayRouteTableId: !Ref TGWRouteTable
```

**Benefit:** All defined routes will be tracked for drift.
**Limitation:** Propagated routes still won't be tracked.

### 2. Separate Static and Propagated Route Tables

**Problem:** Static and propagated routes can interact in unpredictable ways.

**Solution:** Create separate route tables for static vs propagated routes.

```yaml
Resources:
  StaticRouteTable:
    Type: AWS::EC2::TransitGatewayRouteTable
    Properties:
      TransitGatewayId: !Ref TGW
      Tags:
        - Key: Purpose
          Value: StaticRoutes

  PropagatedRouteTable:
    Type: AWS::EC2::TransitGatewayRouteTable
    Properties:
      TransitGatewayId: !Ref TGW
      Tags:
        - Key: Purpose
          Value: PropagatedRoutes

  Propagation:
    Type: AWS::EC2::TransitGatewayRouteTablePropagation
    Properties:
      TransitGatewayAttachmentId: !Ref VPCAttachment
      TransitGatewayRouteTableId: !Ref PropagatedRouteTable
```

**Benefit:** Clear separation of static (drift-tracked) and dynamic (not tracked) routes.

### 3. Custom Drift Detection with Lambda

**Problem:** CloudFormation drift detection has gaps for Transit Gateway routes.

**Solution:** Implement custom drift detection using AWS Lambda and AWS Config.

**Approach:**
1. Use AWS Config to track Transit Gateway route table configuration changes
2. Lambda function compares actual routes (via `describe-transit-gateway-route-tables`) against CloudFormation template
3. Send drift notifications via SNS/EventBridge
4. Optionally trigger remediation workflows

**Tools Available:**
- [cfn-remediate-drift](https://github.com/iann0036/cfn-remediate-drift) - Automated CloudFormation drift remediation
- AWS Config Conformance Packs for continuous compliance
- CloudWatch Events for Transit Gateway changes

### 4. Import Operations for Drift Resolution

When drift is detected, use CloudFormation's import functionality to bring manually created routes into management:

```bash
# 1. Export current drift state
aws cloudformation describe-stack-resource-drifts \
    --stack-name my-tgw-stack

# 2. Update template to include drifted resources

# 3. Import the drifted resources
aws cloudformation create-change-set \
    --stack-name my-tgw-stack \
    --change-set-name import-drifted-routes \
    --change-set-type IMPORT \
    --resources-to-import file://resources-to-import.json
```

**Source:** [AWS CloudFormation - Resolve drift with an import operation](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resource-import-resolve-drift.html)

### 5. Preventive Controls with AWS Organizations

Implement Service Control Policies (SCPs) to prevent manual changes:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Deny",
      "Action": [
        "ec2:CreateTransitGatewayRoute",
        "ec2:DeleteTransitGatewayRoute",
        "ec2:ReplaceTransitGatewayRoute"
      ],
      "Resource": "*",
      "Condition": {
        "StringNotEquals": {
          "aws:PrincipalArn": "arn:aws:iam::ACCOUNT:role/CloudFormationExecutionRole"
        }
      }
    }
  ]
}
```

**Benefit:** Prevents manual route changes outside CloudFormation.
**Consideration:** May limit operational flexibility for emergency changes.

### 6. Automated Drift Detection Monitoring

Set up automated drift detection runs:

```yaml
# EventBridge Rule
DetectDriftSchedule:
  Type: AWS::Events::Rule
  Properties:
    ScheduleExpression: rate(6 hours)
    Targets:
      - Arn: !GetAtt DriftDetectionFunction.Arn
        Id: DriftDetectionTarget

# Lambda Function
DriftDetectionFunction:
  Type: AWS::Lambda::Function
  Properties:
    Handler: index.handler
    Runtime: python3.11
    Code:
      ZipFile: |
        import boto3
        def handler(event, context):
            cfn = boto3.client('cloudformation')
            response = cfn.detect_stack_drift(
                StackName='my-tgw-stack'
            )
            # Send notification if drift detected
```

**Reference:** [AWS Blog - Implementing an alarm to automatically detect drift](https://aws.amazon.com/blogs/mt/implementing-an-alarm-to-automatically-detect-drift-in-aws-cloudformation-stacks/)

---

## Areas for Further Research

### 1. CloudFormation Registry Extensions

**Question:** Can custom resource types or hooks provide better drift detection for Transit Gateway routes?

**Investigation Needed:**
- Explore AWS CloudFormation Public Registry for community extensions
- Consider developing custom resource provider with enhanced drift logic
- Investigate Lambda-backed custom resources for route validation

**Potential Value:** Custom resources could bridge the gap by implementing logic to compare all routes (including propagated) against expected state.

### 2. AWS Config Rules for Route Drift

**Question:** Can AWS Config rules detect route drift that CloudFormation misses?

**Investigation Needed:**
- Review existing Config rules: `transit-gateway-auto-vpc-attach-disabled`
- Develop custom Config rules to track route table state
- Compare Config rule evaluation against CloudFormation drift results

**Potential Value:** Config provides continuous monitoring vs CloudFormation's point-in-time checks.

### 3. Comparison with Terraform

**Question:** How does Terraform handle Transit Gateway route drift detection compared to CloudFormation?

**Investigation Needed:**
- Test Terraform's `terraform plan` against same scenarios
- Review Terraform AWS provider source code for route comparison logic
- Evaluate Terraform Cloud's drift detection capabilities

**Potential Value:** Understanding alternative approaches might inform CloudFormation usage patterns or custom tooling.

### 4. AWS Service Quotas and Route Limits

**Question:** Do Transit Gateway route table quotas affect drift detection?

**Investigation Needed:**
- Document default and maximum route table quotas
- Test drift detection behaviour when approaching limits
- Investigate if quota exhaustion causes false drift results

### 5. Multi-Account Transit Gateway Drift

**Question:** How does drift detection work for Transit Gateway resources shared across AWS accounts via Resource Access Manager?

**Investigation Needed:**
- Test drift detection in hub-and-spoke architecture
- Evaluate cross-account route visibility
- Document permissions required for drift detection in shared scenarios

---

## References and Sources

### Official AWS Documentation

1. **Resource type support - AWS CloudFormation**
   [https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-stack-drift-resource-list.html](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-stack-drift-resource-list.html)
   Accessed: 11 October 2025
   *Complete list of AWS resources supporting drift detection, import, and IaC generation*

2. **Detect unmanaged configuration changes to stacks and resources with drift detection - AWS CloudFormation**
   [https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-stack-drift.html](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-stack-drift.html)
   Accessed: 11 October 2025
   *Official CloudFormation drift detection documentation including concepts, status codes, and considerations*

3. **AWS::EC2::TransitGatewayRoute - AWS CloudFormation**
   [https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroute.html](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroute.html)
   Accessed: 11 October 2025
   *Resource reference for TransitGatewayRoute including properties and update behaviours*

4. **AWS::EC2::TransitGatewayRouteTable - AWS CloudFormation**
   [https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroutetable.html](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroutetable.html)
   Accessed: 11 October 2025
   *Resource reference for TransitGatewayRouteTable*

5. **AWS::EC2::TransitGatewayRouteTablePropagation - AWS CloudFormation**
   [https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroutetablepropagation.html](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-ec2-transitgatewayroutetablepropagation.html)
   Accessed: 11 October 2025
   *Resource reference for route propagation configuration*

6. **Transit gateway route tables in AWS Transit Gateway - Amazon VPC**
   [https://docs.aws.amazon.com/vpc/latest/tgw/tgw-route-tables.html](https://docs.aws.amazon.com/vpc/latest/tgw/tgw-route-tables.html)
   Accessed: 11 October 2025
   *AWS Transit Gateway routing concepts including static and propagated routes*

7. **Resolve drift with an import operation - AWS CloudFormation**
   [https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resource-import-resolve-drift.html](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/resource-import-resolve-drift.html)
   Accessed: 11 October 2025
   *Guide for using import operations to bring drifted resources back under CloudFormation management*

### GitHub Issues and Community Reports

8. **AWS::EC2::TransitGatewayAttachment drift detection bug when resource is deleted manually · Issue #1271**
   [https://github.com/aws-cloudformation/cloudformation-coverage-roadmap/issues/1271](https://github.com/aws-cloudformation/cloudformation-coverage-roadmap/issues/1271)
   Accessed: 11 October 2025
   *Report and discussion of drift detection failure for deleted Transit Gateway attachments*
   Status: Closed/Shipped

9. **cfn-remediate-drift - GitHub**
   [https://github.com/iann0036/cfn-remediate-drift](https://github.com/iann0036/cfn-remediate-drift)
   Accessed: 11 October 2025
   *Community tool for automated CloudFormation drift remediation using import functionality*

### AWS Blog Posts and Articles

10. **New – CloudFormation Drift Detection | AWS News Blog**
    [https://aws.amazon.com/blogs/aws/new-cloudformation-drift-detection/](https://aws.amazon.com/blogs/aws/new-cloudformation-drift-detection/)
    Published: 15 November 2018
    *Original announcement of CloudFormation drift detection feature*

11. **Field Notes: Working with Route Tables in AWS Transit Gateway | AWS Architecture Blog**
    [https://aws.amazon.com/blogs/architecture/field-notes-working-with-route-tables-in-aws-transit-gateway/](https://aws.amazon.com/blogs/architecture/field-notes-working-with-route-tables-in-aws-transit-gateway/)
    Accessed: 11 October 2025
    *Detailed guide on Transit Gateway route table architecture and best practices*

12. **Implementing an alarm to automatically detect drift in AWS CloudFormation stacks | AWS Cloud Operations Blog**
    [https://aws.amazon.com/blogs/mt/implementing-an-alarm-to-automatically-detect-drift-in-aws-cloudformation-stacks/](https://aws.amazon.com/blogs/mt/implementing-an-alarm-to-automatically-detect-drift-in-aws-cloudformation-stacks/)
    Accessed: 11 October 2025
    *Guide for setting up automated drift detection monitoring*

### Third-Party Resources

13. **AWS CloudFormation Drift Detection Guide**
    [https://awsforengineers.com/blog/aws-cloudformation-drift-detection-guide/](https://awsforengineers.com/blog/aws-cloudformation-drift-detection-guide/)
    Accessed: 11 October 2025
    *Third-party guide covering drift detection concepts and practical examples*

14. **AWS CloudFormation Drift Detection & Remediation Guide - Spacelift Blog**
    [https://spacelift.io/blog/aws-cloudformation-drift-detection](https://spacelift.io/blog/aws-cloudformation-drift-detection)
    Accessed: 11 October 2025
    *Guide covering drift detection and remediation strategies*

15. **A DevOps Guide to AWS Transit Gateway - Mechanical Rock Blog**
    [https://blog.mechanicalrock.io/2020/02/24/transit-gateway.html](https://blog.mechanicalrock.io/2020/02/24/transit-gateway.html)
    Published: 24 February 2020
    *DevOps perspective on Transit Gateway management including CloudFormation templates*

### Stack Overflow and Community Discussions

16. **How to fix a drifted AWS CloudFormation stack? - Stack Overflow**
    [https://stackoverflow.com/questions/54386020/how-to-fix-a-drifted-aws-cloudformation-stack](https://stackoverflow.com/questions/54386020/how-to-fix-a-drifted-aws-cloudformation-stack)
    Accessed: 11 October 2025
    *Community discussion on drift remediation approaches*

17. **CloudFormation - Route Table route Propagation for Transit Gateway - Stack Overflow**
    [https://stackoverflow.com/questions/59286839/cloudformation-route-table-route-propagation-for-tansit-gateway](https://stackoverflow.com/questions/59286839/cloudformation-route-table-route-propagation-for-tansit-gateway)
    Accessed: 11 October 2025
    *Discussion on Transit Gateway route propagation in CloudFormation*

---

## Conclusions and Recommendations

### Summary of Findings

1. **All Transit Gateway resources requested support drift detection**, including:
   - AWS::EC2::TransitGateway ✓
   - AWS::EC2::TransitGatewayRoute ✓
   - AWS::EC2::TransitGatewayRouteTable ✓
   - AWS::EC2::TransitGatewayRouteTableAssociation ✓
   - AWS::EC2::TransitGatewayRouteTablePropagation ✓

2. **However, drift detection has significant limitations**:
   - Only routes explicitly defined in CloudFormation templates are tracked
   - Manually added routes that were never in the template are **not detected**
   - Propagated routes are dynamic and **not tracked**
   - Known bug with TransitGatewayAttachment deletion reporting incorrect status

3. **Transit Gateway and VPC route tables share similar drift detection gaps**, primarily around the requirement for explicit route definition.

4. **Static and propagated route interaction creates blind spots** where changes might not be detected due to route priority rules.

### Recommendations for Infrastructure Management

#### For New Deployments

1. **Define all routes explicitly** as `AWS::EC2::TransitGatewayRoute` resources in your CloudFormation templates
2. **Separate static and propagated route tables** to avoid unexpected interactions
3. **Implement preventive controls** using SCPs to restrict manual changes
4. **Set up automated drift detection** with EventBridge and Lambda

#### For Existing Infrastructure

1. **Audit current route tables** to identify all routes (both static and propagated)
2. **Use CloudFormation import operations** to bring unmanaged routes under CloudFormation control
3. **Document propagated routes separately** since they cannot be drift-detected
4. **Consider custom drift detection** using AWS Config and Lambda for complete visibility

#### Operational Best Practices

1. **Run drift detection regularly** (every 6-24 hours) rather than relying on ad-hoc checks
2. **Monitor CloudWatch Logs** for drift detection API calls and results
3. **Create SNS topics** for drift notifications to security and operations teams
4. **Document expected vs actual behaviour** for routes that cannot be drift-tracked (propagated routes)
5. **Implement infrastructure-as-code policies** that require all changes through CloudFormation

### When to Use Custom Drift Detection

Consider implementing custom drift detection with Lambda and AWS Config when:
- You have complex Transit Gateway architectures with many route tables
- Propagated routes are critical to your routing strategy
- Compliance requirements mandate complete infrastructure visibility
- You need drift detection across multiple AWS accounts
- The CloudFormation attachment deletion bug impacts your operations

### Final Recommendations

**CloudFormation drift detection for Transit Gateway route tables is functional but limited.** It works well for static routes explicitly defined in templates but has gaps for:
- Manually added routes (not in template)
- Propagated routes (dynamic by design)
- Cross-stack resource dependencies
- Deleted attachment detection (known bug)

**For production environments**, augment CloudFormation drift detection with:
1. AWS Config rules for continuous monitoring
2. Custom Lambda functions for complete route visibility
3. Preventive controls to reduce manual changes
4. Automated remediation workflows for detected drift

**The most reliable approach** is to define all expected routes explicitly in CloudFormation templates and use SCPs or IAM policies to prevent manual modifications, supplemented by custom monitoring for propagated routes.

---

**Report Compiled By:** Claude (Anthropic AI Assistant)
**Based on:** Official AWS documentation, GitHub issues, community resources, and AWS blog posts
**Last Updated:** 11 October 2025
