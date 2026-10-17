# Fog report Lambda runtime migration

The report template now uses `provided.al2023`; the inline Amplify trigger uses
`python3.14`. Go builds use 1.27.2 and `aws-lambda-go` v1.55.1. Architecture stays
the existing x86_64 and arm64 deployments. Report events, environment variables and generated output are unchanged.

## Official version evidence

Checked 16 October 2026 UTC:

- [AWS runtime table](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtimes.html):
  `provided.al2023` and Python 3.14 are supported; Python 3.15 is explicitly preview.
  Python 3.9, go1.x and provided.al2 are deprecated.
- [AWS Go packaging](https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html):
  executable `bootstrap` at ZIP root, matching architecture, optional `lambda.norpc`,
  and `CGO_ENABLED=0` to avoid host libc dependencies.
- [Go release metadata](https://go.dev/dl/?mode=json) reports `go1.27.2`, stable true.
- [Official Lambda Go releases](https://github.com/aws/aws-lambda-go/releases) and
  [module metadata](https://proxy.golang.org/github.com/aws/aws-lambda-go/@latest)
  report v1.55.1. Other AWS SDK/CLI dependencies were not broadly refreshed.
- [Python release lifecycle](https://devguide.python.org/versions/) supplies language
  support context. Lambda controls the managed runtime's Python patch version.
  AWS permits runtime-provided boto3 for inline CloudFormation code; this trigger
  has no separately packaged Python dependency tree.
- Workflow action versions were checked against their official GitHub latest-release
  pages: checkout v7.0.1, setup-go v7.0.0, setup-python v7.0.0,
  upload-artifact v7.0.2, golangci-lint-action v9.3.0, go-release-action v1.55, configure-aws-credentials v6.3.0. The linter is pinned to v2.14.0, built locally with Go 1.27.2.
  Hosted `ubuntu-latest` selects GitHub's maintained stable runner image.

## Build and validate locally

```sh
make package-lambda
python3 -m unittest discover -s test/lambda -v
go test ./... -v
```

`make package-lambda` builds `dist/lambda-x86_64.zip` and `dist/lambda-arm64.zip`.
Each ZIP contains only an executable Linux `bootstrap` for its named architecture.
`dist/lambda.zip` remains an exact copy of the x86_64 package for compatibility.
The script also accepts an output path and explicit AWS architecture:

```sh
./scripts/package-lambda.sh dist/custom-arm64.zip arm64
```

Invalid architectures are rejected before creating an output. Both builds disable
CGO and legacy RPC support, and embed timezone data. Tests inspect ZIP layout,
executable mode, ELF machine type (x86-64 or AArch64) and absence of a dynamic
interpreter. On Linux, the smoke test executes the matching extracted ZIP. GitHub
runs it on x86_64 and `ubuntu-24.04-arm`, using the same uploaded packages. The ARM
job downloads the archives; it does not rebuild them. On macOS, static checks cover
both ZIPs and a native equivalent exercises the Runtime API without AWS calls.

Push/PR CI saves all three ZIPs in `fog-lambda-al2023-{commit}`. Releases attach the
same filenames beside CLI binaries. Main-only OIDC/S3 publishing waits for both
architecture checks. It retains `fog/unstable.zip` with the legacy root `fog`
executable, and the unqualified AL2023 x86_64 alias `fog/al2023/unstable.zip`.
Architecture-specific bootstrap packages publish to:

- `fog/al2023/x86_64/unstable.zip`
- `fog/al2023/arm64/unstable.zip`

Feature pushes and all PR events skip the entire publishing job. Its existing role
and permission levels are unchanged; no AWS policy updates are made. The release
S3 publisher was already disabled and remains so. The GitHub ARM runner and current
download-artifact action were verified against [GitHub runner documentation](https://docs.github.com/en/actions/reference/runners/github-hosted-runners)
and the [official v8.0.2 release](https://github.com/actions/download-artifact/releases/tag/v8.0.2).

## Deployment prerequisites — separate authorization required

1. **Set `LambdaArchitecture` explicitly to the function's current architecture.**
   The template has no default, so adding this parameter cannot silently change an
   ARM64 function to x86_64. The PT report stacks use `x86_64`; the Demo 2023 report
   stack uses `arm64`. This parameter selects both the Lambda architecture and the
   matching S3 key `fog/al2023/${LambdaArchitecture}/${FogVersion}.zip`.
2. For a release tag, obtain the matching `lambda-{architecture}.zip` and publish it
   separately to `fog/al2023/{architecture}/{release-tag}.zip`; use that exact tag as
   `FogVersion`. The default `latest` requires a separately published alias under
   the selected architecture. Main-only publishing supplies `unstable` after an
   authorized merge/main push. No objects were published during this work.
3. Verify bucket region/availability and existing publisher permission for the new
   keys before merge. No IAM changes are made. Do not use the x86_64 alias for ARM64.
4. Review each owning stack's actual template and source artifact before updating
   runtime and code together. Preserve deployed parameters, architecture, triggers
   and policies. Existing runtime/architecture/handler and stack ownership are
   inventoried below; exact deployed template, S3 keys and source equivalence remain
   unverified because `GetTemplate` and `GetFunctionConfiguration` are denied.
5. Validate report generation and Amplify rebuilds only in a separately authorized
   deployment. Plan rollback with a previous compatible bootstrap artifact;
   deprecated runtime rollback may be blocked. Use versioned artifact keys because
   changing bytes behind a fixed key alone is not a CloudFormation code update.

## Deployed-source coverage

The renewed-session read-only check on 17 October independently confirmed all five
stack/logical-resource/function relationships with `ListStackResources`. Permitted
`ListFunctions` reads confirmed the runtime, architecture and handler. Authentication
works, but the current ViewOnlyAccess roles deny `GetTemplate` and
`GetFunctionConfiguration`; no denied operation was retried or circumvented. No code
archives were downloaded. Source-to-stack ownership is confirmed; exact deployed
source/template matching is not.

| Audit target | Runtime / architecture / handler | Source coverage |
|---|---|---|
| PT 2022 report generator, Sydney | go1.x / x86_64 / fog | Confirmed fogreport-lambda / ReportGeneratorFunction ownership matches maintained template. New x86_64 bootstrap supported; exact deployed template/code key unreadable. |
| PT 2022 report generator, Virginia | go1.x / x86_64 / fog | Same confirmed ownership and supported x86_64 migration. |
| Demo 2023 report generator, Virginia | provided.al2 / arm64 / bootstrap | Confirmed same stack/logical ID. Select LambdaArchitecture=arm64 and the ARM64 ZIP; exact deployed definition/source remains unreadable. |
| Demo 2023 builder trigger, Virginia | python3.9 / x86_64 / index.lambda_handler | Confirmed fogreport-builder / TriggerFunction ownership matches maintained inline-handler definition. Python 3.14 target; inline-source equivalence unverified. |
| Fog Report legacy build trigger, Sydney | python3.9 / x86_64 / app.lambda_handler | Separate fog-reports-trigger-build / HelloWorldFunction stack. Description confirms an Amplify build job; no confirmed source found. Remains outside PR coverage. |

The adjacent `fogreport-generator` repository is an older CDK/site prototype, not
Fog's runtime entry point. Its tracked Go source has no Lambda runtime definition;
a locally untracked `fullstack.yaml` defines site infrastructure without this trigger.
Its modified go.mod and untracked fullstack.yaml/go.sum were preserved. Historical
site report files mention the legacy stack but do not establish maintained ownership.
Other workshop HelloWorld SAM functions found locally do not prove this stack's source.
Do not deploy a guessed template over the legacy stack.

The supplied audit also records EventBridge DescribeRule and S3 notification read
denials. Trigger permissions/targets do not prove current enabled state or activity.
A future deployment must resolve those checks with existing authorized access.
