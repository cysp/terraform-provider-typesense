//nolint:wrapcheck // Resource diagnostics supply the operation. Preserve SDK transport and decoding errors verbatim.
package provider

import (
	"context"

	"github.com/typesense/typesense-go/v3/typesense/api"
)

func (a typesenseAPI) upsertAlias(ctx context.Context, name string, schema api.CollectionAliasSchema) (*api.CollectionAlias, error) {
	response, err := a.client.UpsertAliasWithResponse(ctx, name, schema)
	if err != nil {
		return nil, err
	}

	return responseValue(response.JSON200, response.StatusCode(), response.Body)
}

func (a typesenseAPI) getAlias(ctx context.Context, name string) (*api.CollectionAlias, error) {
	return retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, func(ctx context.Context) (*api.CollectionAlias, error) {
		response, err := a.client.GetAliasWithResponse(ctx, name)
		if err != nil {
			return nil, err
		}

		return responseValue(response.JSON200, response.StatusCode(), response.Body)
	})
}

func (a typesenseAPI) deleteAlias(ctx context.Context, name string) (*api.CollectionAlias, error) {
	response, err := a.client.DeleteAliasWithResponse(ctx, name)
	if err != nil {
		return nil, err
	}

	return responseValue(response.JSON200, response.StatusCode(), response.Body)
}
