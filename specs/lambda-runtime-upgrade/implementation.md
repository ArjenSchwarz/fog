# Implementation explanation

## Beginner

Fog's report generator needs a current AWS environment and a correctly named start
program. The ZIP now contains that start program, and the site rebuild trigger uses
a supported Python version. Reports keep their existing inputs and output formats.

## Intermediate

The SAM report template pairs provided.al2023 with a root bootstrap built for Linux
amd64 and arm64 without CGO. The existing main function enters aws-lambda-go's Runtime API
loop; no separate report implementation was introduced. Timezone data is embedded.
The inline trigger stays an Amplify main-branch RELEASE request. CI saves GitHub
artifacts for validation. Main-only pushes preserve OIDC/S3 publication of both the
legacy fog archive and the separately named AL2023 bootstrap archives. Feature pushes
and PR events skip credentials and S3 publication.

## Expert

Runtime, architecture, executable mode and S3 key must be migrated as one compatible
unit. The separate al2023 prefix avoids replacing legacy fog executables. The static
ELF check excludes a libc interpreter; Linux x86_64 and ARM64 CI execute the matching extracted
ZIP, while macOS validates that archive and runs a host-native equivalent. The fake
Runtime API must decode chunked request bodies and trailers used by aws-lambda-go.
Handler unit tests check settings, success and error forwarding; smoke validation
checks the Runtime API error path without AWS network calls. Full end-to-end report
and Amplify execution still require separately authorized deployed validation.

## Completeness assessment

Maintained Fog templates, library/toolchain, archive builder, CI and contract tests
are implemented. No AWS function has been upgraded by these source changes. Current
template/configuration reads now succeed. The owner confirms the older standalone
Sydney trigger is part of the same builder pipeline. Its packaged app.lambda_handler
contents remain unverified; the combined maintained template's inline TriggerFunction
is not a drop-in update for that logical resource. A runtime-only standalone
candidate can now be prepared with scripts/prepare-builder-runtime-template.py;
it preserves all existing fields except Runtime. Exact source archive s3:GetObject
was denied. Code/dependency compatibility must be verified before a separately
authorised deployment. The authorised PR #216
merge and main publisher completed successfully; versioned release keys remain an
operator prerequisite. Transit ticket creation is
blocked before request submission by a local MCP initialization timeout.

The required LambdaArchitecture parameter selects both the function architecture
and its architecture-qualified artifact key; no default silently migrates existing
ARM64 deployments. Both ZIPs are built once, checked for correct ELF targets, then
executed on matching Linux GitHub runners. Main-only publication waits for both jobs
and retains legacy and unqualified x86_64 aliases.

## Standalone builder follow-up

Beginner: prepare an updated copy of the old builder stack without replacing its
site or Python package. The existing function identity and site binding stay intact.

Intermediate: the local helper accepts the deployed JSON template or GetTemplate
response and changes only HelloWorldFunction.Properties.Runtime to python3.14.
It refuses a different handler/runtime/resource and will not overwrite any file.
Existing CI discovery includes preservation and failure-path tests.

Expert: original SAM and processed CloudFormation forms retain their existing code
locator and all role/resource attributes; a semantic full-template comparison after
restoring the old runtime must equal the input. The helper makes no AWS calls.
The exact archive read was denied, so this is a reviewed runtime-only candidate,
not proof of Python/dependency compatibility or approval to deploy.
