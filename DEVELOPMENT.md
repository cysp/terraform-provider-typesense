# Development

Use Go and tfplugindocs from `go.mod`, a Terraform version from the [acceptance matrix](#acceptance-tests), the official golangci-lint release from `.golangci-lint-version`, and GoReleaser from `.tool-versions`.

```sh
go mod download
golangci-lint config verify
golangci-lint run
golangci-lint fmt --diff
go test -race -shuffle=on -count=1 -timeout=10m ./...
goreleaser check
```

Apply formatting with `golangci-lint fmt`.

## Vulnerability checks

Run the pinned `go run` command in the [govulncheck workflow](.github/workflows/govulncheck.yml) from the repository root with the Go version in `go.mod`. The scanner uses the current Go vulnerability database.

## Documentation

Edit schema descriptions, `templates/`, and `examples/`, then regenerate:

```sh
go generate ./...
go tool tfplugindocs validate --provider-name=terraform-provider-typesense
git diff
git status --short --untracked-files=all
```

Review and commit generated documentation with its source changes. To check committed documentation, regenerate from a clean checkout and verify that `git diff --exit-code` and `git status --short --untracked-files=all` show no changes. CI checks both tracked and untracked output.

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
and coverage flags. Mocked acceptance runs on Terraform 1.15, 1.16 and 1.17.
Live acceptance runs each of those versions against Typesense 29.1 and 30.2.

The [CI workflow](.github/workflows/ci.yml) runs on pull requests, pushes to
main and manual dispatch. The `ci` gate requires generation, linting, vulnerability
scanning, tests, fuzz campaigns and the GoReleaser configuration check to
succeed. A skipped or cancelled prerequisite also fails the gate. CodeQL and
Codecov report separately.

During the required-check migration, `generate` and `lint` mirror their
reusable workflows. Both `test` and `testacc` require the entire test workflow,
including the mocked and live matrices. Before removing these four checks,
verify the replacements on the reviewed revision and main, then update the
repository rules. Preserve the separate CodeQL requirement.

## Releases

Releases use the existing tag-triggered signing workflow, with GoReleaser pinned in `.tool-versions`. Validate configuration locally with `goreleaser check`; a local build without publishing or signing is:

```sh
goreleaser release --snapshot --clean --skip=publish,sign
```

Before tagging a release:

1. Select the version and the exact default-branch commit to release. Verify its CI, acceptance matrix, generated documentation and snapshot artifacts.
2. Prepare release notes covering user-visible changes, compatibility changes and any migration steps. Link the relevant guides.
3. Verify signing credentials and Terraform Registry publication requirements. Pushing a `v`-prefixed tag starts the release workflow.

After publication, verify the signed checksums, registry manifest, platform archives and Terraform Registry ingestion. Install the published version against a disposable service and verify that an apply is followed by a no-op plan. For an upgrade release, also exercise its documented migration. A snapshot build checks packaging; signing and Registry publication require separate verification.

If a published release needs correction, publish a corrected version or withdraw the affected release as appropriate. Do not move a published tag. A provider downgrade does not undo changes already made to Typesense; test any downgrade with copies of state and data before recommending it.
