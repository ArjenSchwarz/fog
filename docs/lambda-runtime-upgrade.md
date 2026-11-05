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
   authorized merge/main push. After the authorised PR #216 merge, the main workflow
   successfully published all four unstable archives (run 39035856705).
3. Verify artifact availability for the selected version and region before deployment.
   The main publisher successfully wrote the new unstable keys; no IAM changes were
   needed. Do not use the x86_64 alias for ARM64.
4. Review each owning stack's actual template and source artifact before updating
   runtime and code together. Preserve deployed parameters, architecture, triggers
   and policies. Original-template and function-configuration reads now succeed.
   The deployed builder variants have different logical IDs and packaging; do not
   apply the full maintained builder template over the older standalone stack.
   Preserve its logical ID and external site integration during an authorised upgrade.
5. Validate report generation and Amplify rebuilds only in a separately authorized
   deployment. Plan rollback with a previous compatible bootstrap artifact;
   deprecated runtime rollback may be blocked. Use versioned artifact keys because
   changing bytes behind a fixed key alone is not a CloudFormation code update.

## Deployed-source coverage

The initial renewed-session check confirmed all five stack/function relationships,
but template/configuration reads were denied. After the user updated the read policy,
both reads succeeded for all five targets. No source archives were downloaded.

| Audit target | Runtime / architecture / handler | Source coverage |
|---|---|---|
| PT 2022 report generator, Sydney | go1.x / x86_64 / fog | Maintained report-template variant with an older pinned SAM artifact. Set LambdaArchitecture=x86_64 and migrate its code locator deliberately. |
| PT 2022 report generator, Virginia | go1.x / x86_64 / fog | Maintained report template; only a semantically equivalent artifact-bucket expression differs from the baseline. |
| Demo 2023 report generator, Virginia | provided.al2 / arm64 / bootstrap | Maintained report-template variant; select LambdaArchitecture=arm64 and matching ARM64 ZIP. |
| Demo 2023 builder trigger, Virginia | python3.9 / x86_64 / index.lambda_handler | Original template exactly matches the pre-upgrade maintained builder template; inline Python matches the current source. Python 3.14 target. |
| Fog Report build trigger, Sydney | python3.9 / x86_64 / app.lambda_handler | Owner confirms the same Fog report builder pipeline, deployed as the older standalone fog-reports-trigger-build / HelloWorldFunction stack. Current template packages Python in a SAM ZIP instead of using the maintained inline TriggerFunction. Python 3.14 is the maintained target, but the standalone deployment adaptation and packaged-code comparison remain outstanding. |

The Sydney function's CloudFormation tags identify its exact standalone stack and
logical ID. They do not name a Git file or prove that the packaged Python equals
the maintained inline handler. Its original template contains only the trigger;
its processed template contains the function and generated role. The maintained
`examples/templates/fogreport-builder.yaml` defines the combined site infrastructure
and inline trigger. These are different deployment shapes within the same pipeline.
PR #216 updates the maintained builder runtime, but does not provide a directly
applicable update for the old standalone logical resource. A future authorised
migration must preserve that identity and verify its existing Python and Amplify
application/branch binding before changing runtime or packaging. Describing it as
unrelated or outside Fog's maintained pipeline was an incorrect inference from the
logical ID and packaging differences. Deployed ZIP contents remain unverified;
template provenance supplied by the owner is distinct from byte-for-byte code identity.

## Prepare the older standalone builder update

The combined builder template must not replace the older standalone stack. Use
`scripts/prepare-builder-runtime-template.py` to prepare a candidate from that
stack's existing template. It makes no AWS calls and changes only
`Resources.HelloWorldFunction.Properties.Runtime` from `python3.9` to `python3.14`.
It keeps the packaged Python, handler, code locator, architecture, generated role,
other resources and any existing event permissions unchanged. Therefore any Amplify
application/branch binding contained in the package is retained without guessing it.
This preserves the binding; it does not verify that the old code runs on Python 3.14.

Export the existing **Processed** template with the already authorised read-only
profile and correct region. Substitute the exact owning stack ID from the function's
CloudFormation tags; these commands prepare local files only:

```sh
aws --profile READ_ONLY_PROFILE --region STACK_REGION cloudformation get-template \
  --stack-name EXACT_STACK_ID --template-stage Processed --output json \
  > /tmp/builder-current.json
python3 scripts/prepare-builder-runtime-template.py \
  /tmp/builder-current.json /tmp/builder-python314-candidate.json
```

The helper accepts a JSON template or the JSON GetTemplate response. JSON Original
SAM templates are also accepted; YAML bodies must be exported/converted to JSON
first. It refuses a different logical resource, handler, source runtime or missing
code locator, and refuses to overwrite the export or any existing candidate.
The existing CI unittest discovery checks preservation and refusal cases.

**Prepared candidate is not deployment acceptance.** The current read-only profile
denied `s3:GetObject` for the known source archive, so no packaged code was read and
no alternate source-download route was attempted. Before deploying, recover that
exact archive through separately granted access, compare its Python/dependencies
with the maintained inline handler, and test its Python 3.14 compatibility and
existing Amplify application/branch binding. Keep any resulting code/package change
separate from this runtime-only candidate. Review a future CloudFormation change set
for the existing stack and expect only the function Runtime change; reject resource
replacement, new site resources, role/policy changes or changed event wiring.
Deployment, invocation, change-set creation and IAM changes are outside this helper
and were not performed for this follow-up.

The adjacent `fogreport-generator` repository is an older CDK/site prototype, not
Fog's runtime entry point. Its tracked Go source has no Lambda runtime definition;
a locally untracked `fullstack.yaml` defines site infrastructure without this trigger.
Its modified go.mod and untracked fullstack.yaml/go.sum were preserved. Historical
site report files mention the older stack. The owner has now confirmed its role in
the same Fog builder pipeline.
Other workshop HelloWorld SAM functions found locally do not prove this stack's source.
Do not deploy a guessed template over the legacy stack.

The supplied audit also records EventBridge DescribeRule and S3 notification read
denials. Trigger permissions/targets do not prove current enabled state or activity.
A future deployment must resolve those checks with existing authorized access.
