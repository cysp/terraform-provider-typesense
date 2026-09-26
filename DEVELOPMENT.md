# Development

Use Go from `go.mod`, Terraform 1.15 or 1.16, the official golangci-lint release from `.golangci-lint-version`, and GoReleaser from `.tool-versions`. Go tools, including tfplugindocs and gofumpt, are pinned in `go.mod`.

```sh
go mod download
golangci-lint config verify
golangci-lint run
golangci-lint fmt --diff
formatting_diff="$(go tool gofumpt -d .)"
printf '%s\n' "$formatting_diff"
test -z "$formatting_diff"
go test -race -shuffle=on -count=1 -timeout=10m ./...
goreleaser check
```

Apply formatting with `golangci-lint fmt` and `go tool gofumpt -w .`. Run both checks using the pinned tool versions. The gofumpt diff command does not fail on differences, so CI checks its output explicitly.

## Documentation

Edit schema descriptions, `templates/`, and `examples/`, then regenerate:

```sh
go generate ./...
go tool tfplugindocs validate --provider-name=terraform-provider-typesense
git diff
git status --short --untracked-files=all
```

Review and commit generated documentation with its source changes. To check committed documentation, regenerate from a clean checkout and verify that `git diff --exit-code` and `git status --short --untracked-files=all` show no changes. CI checks both tracked and untracked output.

## Acceptance tests

Unit tests include Terraform protocol tests against local HTTP servers. Real API acceptance tests require a disposable Typesense instance and create and delete resources. Use a separate instance for each test run. Typesense serializes schema alterations cluster-wide, so run acceptance tests with `-parallel=1`.

```sh
docker run --rm --name typesense-acceptance --tmpfs /data \
  -p 127.0.0.1:8108:8108 \
  -e TYPESENSE_API_KEY=local-test \
  -e TYPESENSE_DATA_DIR=/data typesense/typesense:30.2
```

In another terminal:

```sh
TF_ACC=1 TYPESENSE_URL=http://127.0.0.1:8108 TYPESENSE_API_KEY=local-test \
  go test -count=1 -parallel=1 -timeout=10m ./internal/provider/
```

CI runs every combination of Terraform 1.15/1.16 and Typesense 29.1/30.2. The required `testacc` aggregate covers all four combinations. The provider release targets Typesense 30.2; passing tests on 29.1 provide compatibility coverage without establishing 29.1 support. Set `TF_ACC_TERRAFORM_PATH` to test a specific Terraform binary locally.

## Required checks

The `testacc` aggregate succeeds only when every acceptance matrix job succeeds. It runs even when a matrix job fails or is cancelled. Required workflows must run for every pull request.

When changing CI job names, coordinate the workflow and repository protection settings. Verify all replacement checks on the reviewed commit before changing required contexts. Preserve unrelated protection settings, and confirm that the default branch produces the required check names after the change. Reverting a workflow may also require reverting its required-check configuration.

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
