# Fog report Lambda runtime migration

The report template now uses `provided.al2023`; the inline Amplify trigger uses
`python3.14`. Go builds use 1.27.2 and `aws-lambda-go` v1.55.1. Architecture stays
x86_64. Report events, environment variables and generated output are unchanged.

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
  upload-artifact v7.0.2, golangci-lint-action v9.3.0, go-release-action v1.55.
  Hosted `ubuntu-latest` selects GitHub's maintained stable runner image.

## Build and validate locally

```sh
make package-lambda
python3 -m unittest discover -s test/lambda -v
go test ./... -v
```

The package is `dist/lambda.zip`, containing only an executable Linux amd64
`bootstrap`. The build excludes legacy Go RPC support and embeds timezone data,
so named report timezones remain available without OS timezone files. Archive tests
check executable permissions, ELF architecture and absence of a dynamic interpreter.
On Linux amd64 the smoke test executes the exact extracted archive. On macOS/other
hosts it builds a native executable with the same Runtime API build tag, alongside
archive inspection; that is not an AL2023 execution test. The smoke sends a synthetic
invocation and verifies a handler error returns through the Runtime API, without AWS
calls. Go unit tests separately verify handler success and report failure propagation.
Inline trigger tests execute the template's actual Python source with mocked boto3.

Push/PR CI uses Python 3.14 and saves `fog-lambda-al2023-{commit}` as a GitHub artifact.
Releases attach `lambda.zip` beside the CLI binaries. No workflow publishes to AWS.

## Deployment prerequisites — separate authorization required

1. Obtain the reviewed release `lambda.zip` and publish it to the artifact bucket
   under `fog/al2023/{release-tag}.zip`; use that exact tag as `FogVersion`.
   The template default `latest` requires a separately published
   `fog/al2023/latest.zip` alias. These objects are not created by this PR.
2. Verify bucket region/availability and existing publisher permission for that
   prefix. Do not assume the existing S3 role permits it. No policy changes are made.
3. Review each owning CloudFormation/SAM stack and its actual code source. Update
   runtime and bootstrap code together through the owning stack, preserving deployed
   parameters, architecture, triggers and policies. Changing the template alone does
   not upgrade any existing function. Do not pair AL2023 with old `fog/{version}.zip`.
4. Validate report generation and Amplify rebuilds in a separately authorized test
   deployment. Review initialization logs, named timezone output and event targets.
   Plan rollback using a previous compatible bootstrap artifact; deprecated-runtime
   rollback may be blocked. Versioned artifact keys avoid CloudFormation missing an
   update when only the bytes behind a fixed S3 key change.

Legacy `fog/{version}.zip` objects are deliberately untouched. CI's former every-push
S3 upload has been removed; an operator must separately authorize future publication.

## Deployed-source coverage

The supplied 15 October audit established these functions. No live resource changed.
Fresh `GetTemplate` reads using only the existing elle ViewOnlyAccess profiles failed
because their SSO sessions expired. No login or alternative profile was used. Therefore
source mapping below is by local definitions and supplied resource/logical names,
not independently verified deployed-template identity.

| Audit target | Audit runtime | Source coverage |
|---|---|---|
| PT 2022 report generator, Sydney | go1.x | Logical name matches examples/templates/fogreport-lambda.yaml; maintained source updated. Deployed template/code key needs verification. |
| PT 2022 report generator, Virginia | go1.x | Same maintained report template; deployed-template identity unverified. |
| Demo 2023 report generator, Virginia | provided.al2 | Same logical name, but deployed runtime already differs from local template. Actual owning deployed definition must be verified. |
| Demo 2023 builder trigger, Virginia | python3.9 | Logical name matches examples/templates/fogreport-builder.yaml; maintained inline source updated. |
| Fog Report legacy build trigger, Sydney | python3.9 | No matching maintained stack/HelloWorldFunction trigger source found in Fog. Separate legacy SAM stack; source recovery and runtime upgrade remain outstanding. |

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
