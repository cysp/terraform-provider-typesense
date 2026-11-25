package provider

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/typesense/typesense-go/v3/typesense"
	"github.com/typesense/typesense-go/v3/typesense/api"
)

type typesenseAPI struct {
	client *api.ClientWithResponses
}

func newTypesenseAPI(endpoint, apiKey string) (typesenseAPI, error) {
	// Operation contexts own deadlines. A redirect can replay a mutation or
	// forward credentials, so return the original response instead.
	client, err := api.NewClientWithResponses(endpoint,
		api.WithAPIKey(apiKey),
		api.WithHTTPClient(&http.Client{CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}}),
	)
	if err != nil {
		return typesenseAPI{}, fmt.Errorf("initialize Typesense API client: %w", err)
	}

	return typesenseAPI{client: client}, nil
}

// The generated client sets the typed response only for the documented
// success status and content type. Preserve the status and body on failure.
func responseValue[T any](value *T, status int, body []byte) (*T, error) {
	if value == nil {
		return nil, &typesense.HTTPError{Status: status, Body: body}
	}

	return value, nil
}

func typesenseNotFound(err error) bool {
	response, ok := errors.AsType[*typesense.HTTPError](err)

	return ok && response.Status == http.StatusNotFound
}
