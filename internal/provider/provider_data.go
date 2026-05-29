package provider

import (
	"github.com/typesense/typesense-go/v3/typesense"
)

type TypesenseProviderData struct {
	client *typesense.Client
}
