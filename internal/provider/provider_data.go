package provider

import (
	"github.com/typesense/typesense-go/typesense"
)

type TypesenseProviderData struct {
	client *typesense.Client
}
