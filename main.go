package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cysp/terraform-provider-typesense/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Root generation requires Terraform on PATH; see DEVELOPMENT.md.
//go:generate terraform fmt -recursive ./examples/

//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs --provider-name=terraform-provider-typesense

// set by goreleaser.
var version = "dev"

//nolint:gochecknoglobals // Set by GoReleaser.
var commit = "unknown"

func formatVersion(version, commit string) string {
	return fmt.Sprintf("terraform-provider-typesense %s (commit %s)\n", version, commit)
}

func main() {
	var (
		debug       bool
		showVersion bool
	)

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.BoolVar(&showVersion, "version", false, "print the provider version and exit")
	flag.Parse()

	if showVersion {
		_, err := fmt.Fprint(os.Stdout, formatVersion(version, commit))
		if err != nil {
			log.Fatal(err.Error())
		}

		return
	}

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/cysp/typesense",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
