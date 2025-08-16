package provider

import (
	"context"

	"github.com/cysp/terraform-provider-typesense/internal/typesense-go"
)

type TypesenseAPIKeyHeaderSecuritySource struct {
	APIKey string
}

var _ typesense.SecuritySource = (*TypesenseAPIKeyHeaderSecuritySource)(nil)

func (t *TypesenseAPIKeyHeaderSecuritySource) APIKeyHeader(ctx context.Context, operationName typesense.OperationName, client *typesense.Client) (typesense.APIKeyHeader, error) {
	return typesense.APIKeyHeader{
		APIKey: t.APIKey,
	}, nil
}
