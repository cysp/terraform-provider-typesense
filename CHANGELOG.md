# Changelog

## [0.1.1](https://github.com/cysp/terraform-provider-typesense/compare/v0.1.0...v0.1.1) (2026-11-24)


### Features

* report the provider version and commit ([ad441c6](https://github.com/cysp/terraform-provider-typesense/commit/ad441c647ef202419434a62f8286a982aa0da4af))


### Bug fixes

* **collection:** stop reindexing fields with masked embedding credentials ([02869ce](https://github.com/cysp/terraform-provider-typesense/commit/02869ce9880c40e57c500707fc96c62c76979f01))
* **data-source:** read key prefixes from server metadata ([62e4659](https://github.com/cysp/terraform-provider-typesense/commit/62e4659f41d0686625f1ef6ecf4e758e93de1e06))


### Documentation

* **agents:** align contributor guidance and validation practices ([8c7b678](https://github.com/cysp/terraform-provider-typesense/commit/8c7b678aceea3ef3c098ed7ad801fa4f13ae90c8))
* **key:** explain recovery after an uncertain create ([9412b07](https://github.com/cysp/terraform-provider-typesense/commit/9412b0795ac82615b1c1b7f778161e46e1319389))
* **release:** verify published assets and preserve reviewed notes ([2a9c5c6](https://github.com/cysp/terraform-provider-typesense/commit/2a9c5c6286cb6b253906f7c60d9992747ae9c62b))


### Build

* consolidate Go formatting with golangci-lint 2.15.0 ([a3a5ab6](https://github.com/cysp/terraform-provider-typesense/commit/a3a5ab6dfa2dafbe375b6157067093fa062d4318))
* **generate:** use the shared pinned generator commands ([7583a2c](https://github.com/cysp/terraform-provider-typesense/commit/7583a2ce6fcfde312e906bae106f9de947d14464))
* update GoReleaser to 2.20.3 ([80b7186](https://github.com/cysp/terraform-provider-typesense/commit/80b7186b1fac037a9d93f952dd116b2029c4d8e3))


### Continuous integration

* align generation and lifecycle scenario names ([4622f21](https://github.com/cysp/terraform-provider-typesense/commit/4622f21a5dcf4a5c3a96a6c5ea6f4e7f0a8338b7))
* check provider vulnerabilities with Go 1.27.4 ([0b892f8](https://github.com/cysp/terraform-provider-typesense/commit/0b892f860e9dd723cdf20cd1f2fa8d3d02b79015))
* gate provider checks and separate Terraform acceptance coverage ([8e28b93](https://github.com/cysp/terraform-provider-typesense/commit/8e28b9361f821d98d1a371c17589c0f404c090ac))
* install Terraform for provider protocol tests ([6b94f24](https://github.com/cysp/terraform-provider-typesense/commit/6b94f24053701f6314e4fa82e9dbd833e51b58b3))
* **release:** prepare Release Please and publish after verified CI ([76dc773](https://github.com/cysp/terraform-provider-typesense/commit/76dc773988a5ef3c591eb5a5372aff09dba76d70))
* **release:** reuse the registered repository signing credentials ([38ea585](https://github.com/cysp/terraform-provider-typesense/commit/38ea58513ea7a13be9ed7f210418556f62278193))
* **release:** verify and attest the draft before publication ([afd32ee](https://github.com/cysp/terraform-provider-typesense/commit/afd32ee13e26e6bec0de0c6eaeee7fd5876881bc))
* **test:** use default acceptance matrix job names ([0321e8e](https://github.com/cysp/terraform-provider-typesense/commit/0321e8ea0948456301e9ee5f23bcb7d9a4b5e513))
* update declared indirect dependencies and propagate failures ([93df1f7](https://github.com/cysp/terraform-provider-typesense/commit/93df1f7ff33354ea4b870ccac821940318267678))
* use the aggregate check for branch protection ([36a533c](https://github.com/cysp/terraform-provider-typesense/commit/36a533cd8e51dd1af12fcf38916b62c81e9c11d1))


### Tests

* cover Terraform 1.17 in the Typesense acceptance matrix ([525a768](https://github.com/cysp/terraform-provider-typesense/commit/525a768c51b5f0206faa2123851158b9acd51539))
* **provider:** create a fresh provider per acceptance factory call ([4be37b3](https://github.com/cysp/terraform-provider-typesense/commit/4be37b3a81803209c45f2cc7cd6095fc163c1b03))
* **provider:** retain race detection for mocked acceptance ([1fe7171](https://github.com/cysp/terraform-provider-typesense/commit/1fe717188bf3d93c6d7e9eba0494d09c326c2cfb))


### Maintenance

* **deps:** bump actions/setup-go from 7.0.0 to 7.1.0 ([9267a64](https://github.com/cysp/terraform-provider-typesense/commit/9267a64f540f98d7967ae1337ff2315dee225f4b))
* **deps:** bump github.com/hashicorp/terraform-plugin-sdk/v2 ([39846cd](https://github.com/cysp/terraform-provider-typesense/commit/39846cd916aecda23e6bd4e5958f65fb11ca6af2))
* **deps:** bump goreleaser/goreleaser-action from 7.2.3 to 7.3.0 ([789b466](https://github.com/cysp/terraform-provider-typesense/commit/789b46693dcb5ac763e8288dd4e58badbd75e868))
* **deps:** bump indirect dependencies ([9bfb017](https://github.com/cysp/terraform-provider-typesense/commit/9bfb0176a6bd0c0208825b9975b4c126e68d69bf))
* **deps:** bump peter-evans/create-pull-request from 8.1.1 to 8.1.3 ([6332e3b](https://github.com/cysp/terraform-provider-typesense/commit/6332e3b8cf8b77c6580fcc5cab03c62571078bfe))
