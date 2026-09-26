package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/typesense/typesense-go/v3/typesense"
	typesense_api "github.com/typesense/typesense-go/v3/typesense/api"
)

// KeySchema and KeyResponse extend the SDK's key types with autodelete,
// supported by Typesense 30.2 but absent from typesense-go v3.2.0.
type KeySchema struct {
	typesense_api.ApiKeySchema

	Autodelete *bool `json:"autodelete,omitempty"`
}

type KeyResponse struct {
	typesense_api.ApiKey

	Autodelete *bool `json:"autodelete,omitempty"`
}

var (
	errMissingKeys      = errors.New("key listing response is missing the keys array")
	errMissingKeyID     = errors.New("key response is missing its id")
	errInvalidListedKey = errors.New("key listing response contains a key without an id")
)

// keyAPI uses the same SDK transport, authentication and operation context as
// the other resources. Only key serialization needs to extend the SDK.
type keyAPI struct {
	client *typesense_api.ClientWithResponses
}

func (client keyAPI) create(ctx context.Context, key KeySchema) (*KeyResponse, error) {
	body, err := json.Marshal(key)
	if err != nil {
		return nil, fmt.Errorf("encode key: %w", err)
	}

	response, err := client.client.CreateKeyWithBody(ctx, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create key: %w", err)
	}

	defer response.Body.Close()

	return decodeKeyResponse(response, http.StatusCreated)
}

func (client keyAPI) retrieve(ctx context.Context, id int64) (*KeyResponse, error) {
	response, err := client.client.GetKey(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("retrieve key: %w", err)
	}

	defer response.Body.Close()

	return decodeKeyResponse(response, http.StatusOK)
}

func (client keyAPI) list(ctx context.Context) ([]*KeyResponse, error) {
	response, err := client.client.GetKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}

	defer response.Body.Close()

	var result struct {
		Keys []*KeyResponse `json:"keys"`
	}

	err = decodeKeyJSON(response, http.StatusOK, &result)
	if err != nil {
		return nil, err
	}

	if result.Keys == nil {
		return nil, errMissingKeys
	}

	for _, key := range result.Keys {
		if key == nil || key.Id == nil {
			return nil, errInvalidListedKey
		}
	}

	return result.Keys, nil
}

func decodeKeyResponse(response *http.Response, status int) (*KeyResponse, error) {
	var key KeyResponse

	err := decodeKeyJSON(response, status, &key)
	if err != nil {
		return nil, err
	}

	if key.Id == nil {
		return nil, errMissingKeyID
	}

	return &key, nil
}

func decodeKeyJSON(response *http.Response, status int, target any) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read key response: %w", err)
	}

	if response.StatusCode != status {
		return &typesense.HTTPError{Status: response.StatusCode, Body: body}
	}

	err = json.Unmarshal(body, target)
	if err != nil {
		return fmt.Errorf("decode key response: %w", err)
	}

	return nil
}
