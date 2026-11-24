# Release the provider

Releases use the existing tag-triggered signing workflow, with GoReleaser pinned in `.tool-versions`. Validate configuration locally with `goreleaser check`; a local build without publishing or signing is:

```sh
goreleaser release --snapshot --clean --skip=publish,sign
```

Before tagging a release:

1. Select the version and the exact default-branch commit to release. Verify its CI, acceptance matrix, generated documentation and snapshot artifacts.
2. Prepare and review release notes covering user-visible changes, compatibility changes and any migration steps. Link the relevant guides. Save the notes in a GitHub release draft before pushing the tag. Set `tag` to the intended version tag (including the `v` prefix) and `commit` to the full reviewed commit SHA. GoReleaser finds the draft by its title and preserves its notes. Use the tag as both the draft title and tag name.

   ```sh
   gh release create "$tag" --draft --title "$tag" --target "$commit" --notes-file release-notes.md
   ```

3. Check the draft title, target and notes, signing credentials and Terraform Registry publication requirements. Create the Git tag at the same commit and push it to start the publisher.

   ```sh
   git tag "$tag" "$commit"
   git push origin "refs/tags/$tag"
   ```

The [publisher](../.github/workflows/release.yml) builds and uploads the platform archives, signed checksums and versioned Registry manifest to a draft. It verifies the detached signature, checks that the signed inventory matches the local archives and manifest, verifies their hashes, attests build provenance, and publishes the GitHub release. It trusts successful uploads.

If signing, upload, verification or attestation fails, inspect the draft and rerun failed jobs on the original tag. If the publish response was lost, check whether the release is already public before retrying. A source or workflow fix after tagging needs a higher version.

After publication, download the release assets into an empty directory. Verify the signature and compare the signer's fingerprint with the provider's registered signing key. Check that the signed inventory lists the versioned Registry manifest and one archive for each expected platform from the reviewed snapshot, and verify the downloaded files against those checksums.

```sh
asset_dir=$(mktemp -d)
gh release download "$tag" --dir "$asset_dir"
(
  cd "$asset_dir" || exit 1
  checksum="terraform-provider-typesense_${tag#v}_SHA256SUMS"
  gpg --verify "$checksum.sig" "$checksum" && shasum -a 256 --check --strict "$checksum"
)
```

Confirm the version and rendered documentation in the [Terraform Registry](https://registry.terraform.io/providers/cysp/typesense). Install the published version against a disposable service and verify that an apply is followed by a no-op plan. For an upgrade release, also exercise its documented migration.

If a published release needs correction, publish a corrected version or withdraw the affected release as appropriate. Do not move a published tag. A provider downgrade does not undo changes already made to Typesense; test any downgrade with copies of state and data before recommending it.
