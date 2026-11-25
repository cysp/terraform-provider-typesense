//nolint:wrapcheck // Resource diagnostics supply the operation. Preserve SDK transport and decoding errors verbatim.
package provider

import (
	"context"

	"github.com/typesense/typesense-go/v3/typesense/api"
)

func (a typesenseAPI) createCollection(ctx context.Context, schema *api.CollectionSchema) (*api.CollectionResponse, error) {
	response, err := a.client.CreateCollectionWithResponse(ctx, *schema)
	if err != nil {
		return nil, err
	}

	return responseValue(response.JSON201, response.StatusCode(), response.Body)
}

func (a typesenseAPI) getCollection(ctx context.Context, name string) (*api.CollectionResponse, error) {
	return retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, func(ctx context.Context) (*api.CollectionResponse, error) {
		response, err := a.client.GetCollectionWithResponse(ctx, name)
		if err != nil {
			return nil, err
		}

		return responseValue(response.JSON200, response.StatusCode(), response.Body)
	})
}

func (a typesenseAPI) updateCollection(ctx context.Context, name string, schema api.CollectionUpdateSchema) (*api.CollectionUpdateSchema, error) {
	response, err := a.client.UpdateCollectionWithResponse(ctx, name, schema)
	if err != nil {
		return nil, err
	}

	return responseValue(response.JSON200, response.StatusCode(), response.Body)
}

func (a typesenseAPI) deleteCollection(ctx context.Context, name string) (*api.CollectionResponse, error) {
	response, err := a.client.DeleteCollectionWithResponse(ctx, name)
	if err != nil {
		return nil, err
	}

	return responseValue(response.JSON200, response.StatusCode(), response.Body)
}
