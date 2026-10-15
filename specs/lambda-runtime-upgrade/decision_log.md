# Decision log

| ID | Decision | Rationale |
|---|---|---|
| D1 | Maintenance smolspec; proceed under delegated implementation authorization | No product behavior changes, schema/auth changes or contested architecture. Source changes are reversible; AWS deployment is excluded. User authorized feature isolation and implementation. |
| D2 | provided.al2023 and Python 3.14 | AWS supported runtime table lists AL2023 and explicitly marks Python 3.15 preview (checked 2026-10-16 UTC). |
| D3 | Go 1.27.2; Lambda Go v1.55.1 | Official go.dev release JSON marks 1.27.2 stable; official module metadata reports v1.55.1. No guessed versions. |
| D4 | Separate fog/al2023 artifact prefix | Existing public fog/{version}.zip archives contain fog, so selecting that key under the new runtime would fail. Preserve legacy consumers instead of overwriting their archive format. |
| D5 | Preserve inline Python SDK | AWS explicitly supports runtime-provided SDK for inline CloudFormation code; packaging a separate Python artifact is unnecessary here. |
| D6 | Independent clone and feature branch | Original Fog checkout contains unrelated edits; original fogreport-generator has modified go.mod and untracked fullstack.yaml/go.sum. No source working changes copied. |

Transit: initialization at http://localhost:3141/mcp timed out after eight seconds with network access. No creation request sent, no ticket exists from this attempt. Requested ticket: Fog — Upgrade report Lambda runtimes and packaging; project explicitly Fog; type chore. Do not repeatedly block independent work on Transit.

| D7 | Remove CI AWS publication | Explicit no AWS changes boundary: feature pushes and releases only produce GitHub artifacts. Existing every-push publication was unsafe for this authorized task. Future S3 publication requires separate authorization. |
| D8 | Embed timezone database | A statically linked binary can still depend on OS timezone files; embedding preserves named timezones in minimal Lambda hosts. |

Explanation validation classified the publication wording as a scope ambiguity; the explicit no-AWS-writes authorization settles it by removing publication. Other findings were documentation/acceptance gaps, resolved with source URLs, audit target guide, artifact alias mapping, timezone embedding, contract tests and Linux/native smoke-test distinctions. Design critique confirms runtime+archive updates must travel together; no alternative architecture or product behavior decision remains.

Design critique: use a new versioned object key before stack migration; fixed-key S3 overwrites are not CloudFormation code updates. CI performs no AWS writes. Runtime API smoke exercises initialization and error reporting; successful report behavior uses the existing injected generator unit test to avoid AWS calls. Linux amd64 CI runs the exact archive, while macOS runs a native equivalent. Fresh AWS template identity remains unverified due to expired SSO. Exact account/function audit evidence stays outside this public branch.

Pre-push review: reuse, efficiency and spec agents found no issues. Quality review found the mock Runtime API incorrectly assumed Content-Length. Fixed chunked body/trailer decoding; Python 3.14 smoke rerun passed. No production-handler change was needed.

## Publication correction after parent review

D7 is superseded by D9; its original rationale is retained above as history, not as the final design. Removing valid publication capability was broader than necessary for a task that must not deploy. No runtime or security defect required deleting the publisher.

| ID | Decision | Rationale |
|---|---|---|
| D9 | Restore OIDC/S3 publisher with main-push-only guards | Preserve existing fog/unstable.zip publishing and legacy archive layout. Add a separate AL2023 unstable key for the new runtime; feature pushes and PRs skip every publishing step. No main merge or workflow dispatch is authorized or performed. Existing publisher IAM configuration is unchanged; new-prefix permission must be verified before merge. |

The release S3 block was already commented out before this branch, so no active release publishing capability was removed. Release-tag/latest publication remains an operator prerequisite. Exact-head CI on d135972 is inspected separately; the existing Claude review exits is_error:true after one turn, zero cost, empty model usage and zero permission denials, with detailed SDK error output hidden by the action. Logs do not establish authentication or quota as its cause. No review finding was emitted.

Correction verification: independent quality/spec reviews found no findings. actionlint exposed the existing unquoted release version extraction; replaced it with quoted GITHUB_REF_NAME/GITHUB_ENV expansion. Legacy Linux amd64 build succeeds without lambda.norpc, preserving the Go1.x RPC entry point. The prior d135972 exact-head build-and-test run passed; restored publisher is not eligible on this feature branch. No AWS call was made.
