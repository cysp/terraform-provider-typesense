# Release the provider

Releases use the existing tag-triggered signing workflow, with GoReleaser pinned in `.tool-versions`. Validate configuration locally with `goreleaser check`; a local build without publishing or signing is:

```sh
goreleaser release --snapshot --clean --skip=publish,sign
```

Before tagging a release:

1. Select the version and the exact default-branch commit to release. Verify its CI, acceptance matrix, generated documentation and snapshot artifacts.
2. Prepare release notes covering user-visible changes, compatibility changes and any migration steps. Link the relevant guides.
3. Verify signing credentials and Terraform Registry publication requirements. Pushing a `v`-prefixed tag starts the release workflow.

The [publisher](../.github/workflows/release.yml) builds and uploads the platform archives, signed checksums and versioned Registry manifest to a draft. It verifies the detached signature, checks that the signed inventory matches the local archives and manifest, verifies their hashes, attests build provenance, and publishes the GitHub release. It trusts successful uploads.

If signing, upload, verification or attestation fails, inspect the draft and rerun failed jobs on the original tag. If the publish response was lost, check whether the release is already public before retrying. A source or workflow fix after tagging needs a higher version.

After publication, confirm the version and rendered documentation in the [Terraform Registry](https://registry.terraform.io/providers/cysp/typesense). Install the published version against a disposable service and verify that an apply is followed by a no-op plan. For an upgrade release, also exercise its documented migration. A snapshot build checks packaging; signing and Registry publication require separate verification.

If a published release needs correction, publish a corrected version or withdraw the affected release as appropriate. Do not move a published tag. A provider downgrade does not undo changes already made to Typesense; test any downgrade with copies of state and data before recommending it.
