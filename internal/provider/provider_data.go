package provider

import (
	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
)

type TypesenseProviderData struct {
	client *typesense.Client
}
