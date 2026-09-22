# Development

Use the official golangci-lint release matching `.golangci-lint-version` for local and CI checks.

```sh
golangci-lint config verify
golangci-lint run
golangci-lint fmt --diff
go test ./...
```

`golangci-lint fmt` applies formatting changes. The separate format check also covers files excluded from the normal build.
