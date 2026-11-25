# Development

For a provider change, start with the [validation scope](#validation-scope). For test design, see [Testing](docs/testing.md).

## Prerequisites

Tool requirements come from [go.mod](go.mod) (Go and generators),
[.golangci-lint-version](.golangci-lint-version), [.tool-versions](.tool-versions)
and the Terraform matrix in the [test workflow](.github/workflows/test.yml).
Root documentation generation requires Terraform on `PATH`. Use `go mod download`
to populate the module cache.

Repository-wide guidance lives in [AGENTS.md](AGENTS.md). See [Reviewing changes](docs/reviewing.md) for code review.

## Commit conventions

- Use Conventional Commit messages. The type describes the kind of change, the scope identifies the affected component or maintenance concern, and the subject describes the specific change.
- For components under `internal/`, use the immediate directory's exact name as the scope, such as `provider`. Name individual resources in the subject.
- Tests, examples, generated files, and documentation about a component use that component's scope, regardless of their location.
- For repository maintenance, use the established concern name, such as `deps`, `lint`, `generate`, `release`, or `agents`. Scope workflow changes by their purpose.
- When a cohesive change spans components, use its primary concern as the scope if one clearly owns the change; otherwise omit the scope. Split independently useful changes into separate commits where appropriate.
- For a new concern, prefer an existing directory, tool, or workflow name. Use recent history to resolve choices left open by these rules.

## Repository map

Provider schemas, Terraform lifecycle methods, request and response models, and local HTTP fixtures live in [internal/provider](internal/provider/). The [Typesense API adapter](internal/provider/typesense_api.go) configures the generated SDK client and handles shared HTTP response semantics. Its operations are grouped by [alias](internal/provider/typesense_api_alias.go), [collection](internal/provider/typesense_api_collection.go), and [key](internal/provider/typesense_api_key.go).

## Validation scope

Select checks by the changed behavior. When preparing a code change for merge, inspect the remote checks for that revision. The [workflows](.github/workflows/) define the current CI matrix.

| Change | Local verification |
| --- | --- |
| Agent instructions or prose outside generation inputs | Review instruction consistency, links, and the final diff. Run `git diff --check`. |
| Go behavior | Test the affected packages and run the [lint and format checks](#linting). Use `go test -race -shuffle=on -count=1 -timeout=10m ./...` and `go build .` for shared behavior or when preparing for merge. |
| Terraform planning, state, or lifecycle | Exercise the affected transitions through Terraform against local fixtures. Extend coverage where existing tests do not establish the changed behavior. |
| Schema, examples, templates or other generation inputs | Regenerate, validate the Registry pages and review the tracked and untracked output. Run checks for the affected behavior. |
| A claim about live Typesense behavior | Use primary documentation or a disposable real-server experiment. Record the server version and the behavior observed. |
| Packaging | Run the pinned `goreleaser check` and inspect a snapshot build. See [Releases](#releases). |

If a check cannot run, record the command and concrete blocker. Distinguish a passing check from a skipped test or an environment failure.

## Documentation authoring

Follow the [documentation practices](docs/design/provider-documentation.md), edit the authoritative input, then [regenerate](#code-generation) and [verify the output](#documentation-verification).

| Documentation concern | Authoritative input |
| --- | --- |
| Provider, resource, data-source, function and attribute contracts | Schema and function descriptions under `internal/provider/` |
| Terraform configuration and import syntax | `examples/` |
| Provider page | `templates/index.md.tmpl` |
| Common resource layout | `templates/resources.md.tmpl` |
| Common data-source layout | `templates/data-sources.md.tmpl` |
| Common function layout | `templates/functions.md.tmpl` |
| Practitioner guides | `templates/guides/` |

Keep handwritten contributor pages outside the directories managed by tfplugindocs.

## Code generation

After changing a schema or any other generation input, run full generation from the repository root with Terraform on `PATH`. Review the tracked changes and any new files listed by Git.

```sh
go generate ./...
git status --short --untracked-files=all
```

To check reproducibility, repeat full generation and the status check in a clean checkout containing your intended changes. The status output should be empty.

## Documentation verification

```sh
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate --provider-name=terraform-provider-typesense
git diff --check
```

Review the generated pages and validate runnable examples in an isolated configuration.

## Local provider build

`go build .` produces `terraform-provider-typesense` in the repository root.
For manual Terraform use, the provider address is `cysp/typesense`; see
HashiCorp's [development overrides](https://developer.hashicorp.com/terraform/cli/config/config-file#development-overrides-for-provider-developers).

## Linting

Install the [official golangci-lint binary](https://golangci-lint.run/docs/welcome/install/local/#binaries)
at the version recorded in [`.golangci-lint-version`](.golangci-lint-version),
then run the repository checks:

```sh
golangci-lint run
golangci-lint fmt --diff
```

Run `golangci-lint fmt` to apply the configured formatting, including gofumpt.

## Vulnerability checks

For a local vulnerability check, run the `go run` command from the
[govulncheck workflow](.github/workflows/govulncheck.yml) at the repository root.
The workflow pins the scanner version; update that pin explicitly when upgrading
the scanner. Each run still consults the current Go vulnerability database.

## Tests

### Unit and local integration tests

With `TF_ACC` unset, run the normal unit and local integration test suite:

```sh
go test -race -shuffle=on -count=1 -timeout=10m ./...
```

Run a focused package or test while iterating:

```sh
go test ./internal/provider -run '^TestProtocol6ProviderServerSchemaVersion$' -count=1 -timeout=5m
```

Use the current test name from the source. A successful command with
`[no tests to run]` has not checked the intended behavior.

For order-dependent failures, rerun with the `-shuffle=<seed>` printed by Go.
Add `-v` for test and subtest progress. Use `TF_LOG=DEBUG` or
`TF_ACC_PERSIST_WORKING_DIR=1` for focused acceptance debugging. Logs and
retained Terraform state may contain credentials.

### Acceptance tests

#### Mocked acceptance tests

Run a focused mocked lifecycle test, then broaden to the mocked acceptance suite
when the change requires it:

```sh
TF_ACC=1 TF_ACC_MOCKED=1 go test ./internal/provider -run '^TestAccAliasResourceCreateUpdateDelete$' -count=1 -timeout=5m
TF_ACC=1 TF_ACC_MOCKED=1 go test ./internal/provider -run '^TestAcc' -count=1 -timeout 15m
```

#### Live acceptance tests

Use a disposable Typesense server. Each run creates and deletes resources.
Typesense serializes schema alterations cluster-wide, so run the live suite with
`-parallel=1`. Start a separate server for each run:

```sh
docker run --rm --name typesense-acceptance --tmpfs /data \
  -p 127.0.0.1:8108:8108 \
  -e TYPESENSE_API_KEY=local-test \
  -e TYPESENSE_DATA_DIR=/data typesense/typesense:30.2
```

In another terminal, clear any inherited mocked setting:

```sh
env -u TF_ACC_MOCKED TF_ACC=1 TYPESENSE_URL=http://127.0.0.1:8108 TYPESENSE_API_KEY=local-test \
  go test ./internal/provider -run '^TestAcc' -count=1 -parallel=1 -timeout 15m
```

### Interpreting test results

With `TF_ACC` unset, ordinary tests cover models, schemas, provider protocol
calls, local HTTP integration, Rapid properties and fuzz seeds without invoking
Terraform. All Terraform CLI tests require `TF_ACC` and use the `TestAcc`
prefix. Mock-only cases use isolated fixtures in either acceptance mode.
Call `requireLiveAcceptance(t)` before live fixture setup. Live-only cases skip when `TF_ACC_MOCKED` is nonempty. Inspect individual
subtest results when checking what actually executed.

Install Terraform on `PATH` or set `TF_ACC_TERRAFORM_PATH` to select an
installed executable. Live upgrade tests download the pinned released provider
from the Registry. Each live CI job has its own disposable server and API key,
including on pull requests from forks.

### Property and fuzz campaigns

Rapid properties run with the ordinary tests. To increase their iteration budget
locally, use `RAPID_CHECKS` so packages that do not import Rapid still accept the
ordinary suite command:

```sh
RAPID_CHECKS=1000 go test -race -shuffle=on -count=1 -timeout=10m ./...
```

Use an anchored name to fuzz one target without repeating ordinary tests:

```sh
go test ./internal/provider -run '^$' -fuzz '^FuzzScopedSearchParametersRoundTrip$' \
  -fuzztime 30s -fuzzminimizetime 10s -parallel 2 -timeout 3m
```

The [ordinary test workflow](.github/workflows/test.yml) sets the PR property
budget. The [campaign workflow](.github/workflows/fuzz.yml) owns the selected
fuzz targets, time and worker limits, and the larger scheduled/manual budgets.
CI calls it for short campaigns on PRs and main; its daily schedule and manual
dispatch run longer fuzz campaigns and the provider ordinary suite with more
Rapid checks. Campaigns do not enable acceptance tests or receive Typesense
credentials.

Failed jobs retain logs and generated reproductions as artifacts for the
retention period configured in those workflows.
Rapid prints a seed and writes `testdata/rapid/<test>/*.fail` relative to the
tested package. Restore a failure file there and rerun the named test, or use
`-rapid.seed=<seed>` with `go test ./internal/provider -run '^TestName$' -count=1`.
Go fuzzing writes failures beneath `testdata/fuzz/<target>/`; restore the corpus
entry beneath `internal/provider/` and use the `-run` command printed in the log.
Review minimized failures and retain useful cases as regression tests or corpus
entries. A campaign that passes establishes only the inputs it exercised.

### CI coverage

The [test workflow](.github/workflows/test.yml) defines the Terraform matrix
and coverage flags. Mocked acceptance runs with the Go race detector on Terraform 1.15, 1.16 and 1.17.
Live acceptance runs each of those versions against Typesense 29.1 and 30.2.

The [CI workflow](.github/workflows/ci.yml) runs on pull requests, pushes to
main and manual dispatch. The `ci` gate requires generation, linting, vulnerability
scanning, tests, fuzz campaigns and the GoReleaser configuration check to
succeed. A skipped or cancelled prerequisite also fails the gate. Merges also
require Codecov’s `codecov/project` check and the repository’s CodeQL rule.

## Releases

See [Provider releases](docs/releasing.md) for the publishing and verification workflow.
