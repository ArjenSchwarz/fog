# Lambda runtime upgrade

## Overview
Fog report functions use deprecated Go and Python Lambda runtimes. Update maintained templates, runtime library, toolchain and packaging while preserving report behavior and the x86_64 architecture.

## Requirements
- Maintained report templates MUST select generally available supported Lambda runtimes, verified against official documentation.
- The Go deployment archive MUST contain an executable Linux amd64 bootstrap at its root and run the existing report handler through the Lambda Runtime API.
- The Python trigger MUST request the same Amplify main branch RELEASE job and propagate service failures.
- Feature branch and pull request validation MUST NOT publish to AWS.
- Changes MUST preserve report event, environment and output contracts; no AWS deployment, IAM changes, deletion or merge is in scope.

## Implementation Approach
- Update examples/templates/fogreport-lambda.yaml to provided.al2023 and bootstrap; use fog/al2023/{FogVersion}.zip so old go1.x archive keys remain separate.
- Update examples/templates/fogreport-builder.yaml to python3.14, preserving inline boto3 usage supported by AWS for inline templates.
- Update go.mod/go.sum to official stable Go 1.27.2 and aws-lambda-go v1.55.1. Other SDK modules remain unless compatibility requires changes.
- Add scripts/package-lambda.sh and a Makefile target using CGO_ENABLED=0, linux/amd64 and lambda.norpc. Reuse main.go handler selection and existing handler tests.
- Update .github/workflows/push.yml and release.yaml to use the module toolchain and package bootstrap; validate with PRs and publish build artifacts to GitHub. Preserve existing OIDC/S3 publication only for main pushes, retain the legacy fog archive at fog/unstable.zip, and publish bootstrap separately at fog/al2023/unstable.zip. Feature/PR events skip all publisher steps.
- Add executable archive/Runtime API smoke tests and inline Python success/failure tests; document migration in docs/lambda-runtime-upgrade.md and README.md.
- Audit the five known deployed functions through bounded metadata reads; document external legacy sources without editing their unrelated working changes.
- Dependencies: aws-lambda-go implements the Runtime API; inline Python uses runtime-provided boto3. Out of scope: wholesale SDK/CLI dependency refresh, Amplify/site redesign and external legacy repository changes.

Official sources checked 2026-10-16 UTC: https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html ; https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html ; https://go.dev/dl/?mode=json ; https://proxy.golang.org/github.com/aws/aws-lambda-go/@latest . Audit targets and limitations are recorded in docs/lambda-runtime-upgrade.md. Embed Go timezone data to retain named report timezones in minimal images. GitHub release lambda.zip maps to fog/al2023/{release-tag}.zip; latest is a separately published alias. No templates are deployable until a compatible object is published.

## Risks and Assumptions
- Risk: changing runtime without changing the archive fails initialization. Mitigation: verify archive contents, executable mode, ELF architecture and a local Runtime API exchange using the extracted Linux archive on Linux amd64; on other hosts validate the archive and run a native binary using the same build tag. Verify a Runtime API error response without AWS calls, and handler success/error results with the existing report-generator test seam.
- Risk: AWS publication might run on feature pushes. Mitigation: guard legacy build, credential configuration and S3 writes with push plus refs/heads/main conditions.
- Prerequisite: separately authorize publication of the new artifact prefix and stack updates; existing hosted latest.zip is unsuitable for provided.al2023.
- Assumption: x86_64 remains the deployed architecture. Audit records exact mapping confidence and missing source ownership.

## Escalation Note
This change was scoped as a smolspec. If implementation reveals ambiguity only the user can resolve, an irreversible boundary (public API, persisted schema, auth path), or a contested architectural choice, stop and escalate to the full spec workflow rather than deciding it inline.
