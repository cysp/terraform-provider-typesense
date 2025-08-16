package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
)

type TypesenseAPIKeyHeaderSecuritySource struct {
	APIKey string
}

var _ typesense.SecuritySource = (*TypesenseAPIKeyHeaderSecuritySource)(nil)

func (t *TypesenseAPIKeyHeaderSecuritySource) APIKeyHeader(_ context.Context, _ typesense.OperationName, _ *typesense.Client) (typesense.APIKeyHeader, error) {
	return typesense.APIKeyHeader{
		APIKey: t.APIKey,
	}, nil
}
