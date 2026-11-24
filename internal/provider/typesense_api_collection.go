//nolint:wrapcheck // Resource diagnostics supply the operation. Preserve SDK transport and decoding errors verbatim.
package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/typesense/typesense-go/v3/typesense"
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

func (a typesenseAPI) getCollectionForAlteration(ctx context.Context, name string) (*observedCollection, error) {
	return retrieveWithNotFoundConfirmation(ctx, notFoundConfirmationTimeout, func(ctx context.Context) (*observedCollection, error) {
		response, err := a.client.GetCollection(ctx, name)
		if err != nil {
			return nil, err
		}

		if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Type"), "json") {
			// Preserve the generated client's response and error handling, including malformed 404s.
			parsed, err := api.ParseGetCollectionResponse(response)
			if err != nil {
				return nil, err
			}

			return nil, &typesense.HTTPError{Status: parsed.StatusCode(), Body: parsed.Body}
		}

		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		return decodeObservedCollection(body)
	})
}

func (a typesenseAPI) updateCollection(ctx context.Context, name string, body []byte) error {
	response, err := a.client.UpdateCollectionWithBodyWithResponse(ctx, name, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	_, err = responseValue(response.JSON200, response.StatusCode(), response.Body)

	return err
}

func (a typesenseAPI) deleteCollection(ctx context.Context, name string) (*api.CollectionResponse, error) {
	response, err := a.client.DeleteCollectionWithResponse(ctx, name)
	if err != nil {
		return nil, err
	}

	return responseValue(response.JSON200, response.StatusCode(), response.Body)
}
